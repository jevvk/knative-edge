package edge

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"edge.jevv.dev/pkg/controllers"
)

type kindGenerator[T client.Object] func() T
type kindMerger[T client.Object] func(src, dst T) error
type kindPreProcessor[T client.Object] func(ctx context.Context, kind T) kindPreProcessorResult

type MirroringReconciler[T client.Object] struct {
	client.Client

	Log           logr.Logger
	Scheme        *runtime.Scheme
	Recorder      record.EventRecorder
	RemoteCluster cluster.Cluster

	KindGenerator     kindGenerator[T]
	KindMerger        kindMerger[T]
	KindPreProcessors *[]kindPreProcessor[T]

	Envs []string
}

type kindPreProcessorResult struct {
	Result       ctrl.Result
	Err          error
	ShouldUpdate bool
}

func (r *MirroringReconciler[T]) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.Log.V(1).Info("Started reconciling remote and local cluster.", "resource", req.NamespacedName.String())

	if r.Client == nil {
		return ctrl.Result{}, fmt.Errorf("no local kube client")
	}

	if r.RemoteCluster.GetClient() == nil {
		return ctrl.Result{}, fmt.Errorf("no remote kube client")
	}

	localKind, remoteKind := r.KindGenerator(), r.KindGenerator()

	shouldCreate := false
	shouldUpdate := false
	shouldDelete := false

	if err := r.RemoteCluster.GetClient().Get(ctx, req.NamespacedName, remoteKind); err != nil {
		if apierrors.IsNotFound(err) {
			shouldDelete = true
		} else {
			return ctrl.Result{}, err
		}
	}

	if err := r.Get(ctx, req.NamespacedName, localKind); err != nil {
		if apierrors.IsNotFound(err) {
			shouldCreate = !shouldDelete
			shouldDelete = false // can't delete something that doesn't exist
		} else {
			return ctrl.Result{}, err
		}
	} else if !shouldDelete && IsManagedObject(localKind) && !HasEdgeSyncLabel(remoteKind, r.Envs) {
		// delete is edge sync label no longer valid (e.g. if envs change)
		shouldDelete = true
	}

	if !shouldDelete {
		kindAnnotations := localKind.GetAnnotations()

		if kindAnnotations == nil {
			kindAnnotations = make(map[string]string)
		}

		remoteGeneration := fmt.Sprint(remoteKind.GetResourceVersion())
		lastRemoteGeneration, lastRemoteGenerationExists := kindAnnotations[controllers.LastRemoteGenerationAnnotation]

		if remoteGeneration != lastRemoteGeneration {
			shouldUpdate = lastRemoteGenerationExists
		}

		r.KindMerger(remoteKind, localKind)

		if r.KindPreProcessors != nil {
			for _, preprocessor := range *r.KindPreProcessors {
				res := preprocessor(ctx, localKind)

				shouldUpdate = shouldUpdate || res.ShouldUpdate

				if res.Err != nil || res.Result.Requeue || res.Result.RequeueAfter > 0 {
					return res.Result, res.Err
				}
			}
		}

		controllers.UpdateLastGenerationAnnotation(localKind)
		controllers.UpdateLastRemoteGenerationAnnotation(localKind, remoteKind)
		controllers.UpdateLabels(localKind)
	}

	r.Log.V(3).Info("debug kind", "resource", req.NamespacedName.String(), "localKind", localKind, "remoteKind", remoteKind)
	r.Log.V(3).Info("debug bool", "resource", req.NamespacedName.String(), "shouldCreate", shouldCreate, "shouldUpdate", shouldUpdate, "shouldDelete", shouldDelete)

	if shouldCreate {
		r.Log.V(1).Info("Creating local resource.", "name", req.NamespacedName.String())
		if err := r.Create(ctx, localKind); err != nil {
			if apierrors.IsConflict(err) {
				return ctrl.Result{Requeue: true}, nil
			}

			return ctrl.Result{}, err
		}
	} else if shouldDelete {
		r.Log.V(1).Info("Deleting local resource.", "name", req.NamespacedName.String())
		if err := r.Delete(ctx, localKind); err != nil {
			if apierrors.IsNotFound(err) {
				return ctrl.Result{}, nil
			}

			return ctrl.Result{}, err
		}
	} else if shouldUpdate {
		r.Log.V(1).Info("Updating local resource.", "name", req.NamespacedName.String())
		if err := r.Update(ctx, localKind); err != nil {
			if apierrors.IsConflict(err) {
				return ctrl.Result{Requeue: true}, nil
			}

			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *MirroringReconciler[T]) NewControllerManagedBy(mgr ctrl.Manager, predicates ...predicate.Predicate) *builder.Builder {
	predicates = append(predicates, predicate.ResourceVersionChangedPredicate{})

	return ctrl.NewControllerManagedBy(mgr).
		// local watch
		For(
			r.KindGenerator(),
			builder.WithPredicates(
				IsManagedByEdgeControllers,
				NotChangedByEdgeControllers{},
			),
		).
		// remote watch
		Watches(
			source.NewKindWithCache(r.KindGenerator(), r.RemoteCluster.GetCache()),
			&handler.EnqueueRequestForObject{},
			builder.WithPredicates(
				predicates...,
			),
		)
}
