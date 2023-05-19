package edge

import (
	"context"
	"fmt"
	"reflect"

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
	"edge.jevv.dev/pkg/controllers/utils"
)

type kindGenerator[T client.Object] func() T
type kindMerger[T client.Object] func(src, dst T) error
type kindPreProcessor[T client.Object] func(ctx context.Context, kind T) (ctrl.Result, error)

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

func (r *MirroringReconciler[T]) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	debug := r.Log.V(controllers.DebugLevel)
	log := r.Log.V(controllers.InfoLevel)

	debug.Info("Started reconciling remote and local cluster.", "resource", req.NamespacedName.String())

	result := ctrl.Result{}

	if r.Client == nil {
		return result, fmt.Errorf("no local kube client")
	}

	if r.RemoteCluster.GetClient() == nil {
		return result, fmt.Errorf("no remote kube client")
	}

	localKind, remoteKind := r.KindGenerator(), r.KindGenerator()

	shouldCreate := false
	shouldUpdate := false
	shouldDelete := false

	if err := r.RemoteCluster.GetClient().Get(ctx, req.NamespacedName, remoteKind); err != nil {
		if apierrors.IsNotFound(err) {
			shouldDelete = true
		} else {
			return result, err
		}
	}

	if err := r.Get(ctx, req.NamespacedName, localKind); err != nil {
		if !apierrors.IsNotFound(err) {
			return result, err
		}

		// exit early if both local and remote kind don't exist
		if shouldDelete {
			return result, nil
		}

		shouldCreate = true
	} else if !shouldDelete && IsManagedObject(localKind) && !HasEdgeSyncLabel(remoteKind, r.Envs) {
		// delete is edge sync label no longer valid (e.g. if envs change)
		shouldDelete = true
	}

	// make a copy to compare after the changes
	localKindCopy, ok := localKind.DeepCopyObject().(T)

	if !ok {
		// this shouldn't happen, ever
		err := fmt.Errorf("cannot cast copy of localKind")
		r.Log.Error(err, "Error occured while copying localKind", "kind", localKind.GetObjectKind())

		// shouldUpdate becomes useless if this happens, but at least we don't break the control loop
		localKindCopy = localKind
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

		r.KindMerger(remoteKind, localKindCopy)

		if r.KindPreProcessors != nil {
			for _, preprocessor := range *r.KindPreProcessors {
				res, err := preprocessor(ctx, localKindCopy)

				debug.Info("preprocessor", "resource", req.NamespacedName.String(), "result", res)

				if err != nil {
					return res, err
				}

				result.Requeue = result.Requeue || res.Requeue

				if res.RequeueAfter > 0 && (res.RequeueAfter < result.RequeueAfter || result.RequeueAfter == 0) {
					result.RequeueAfter = res.RequeueAfter
				}
			}

			debug.Info("preprocessor end", "resource", req.NamespacedName.String(), "result", result)
		}

		utils.UpdateLastRemoteGenerationAnnotation(localKindCopy, remoteKind)
		utils.UpdateLabels(localKindCopy)

		shouldUpdate = shouldUpdate || !reflect.DeepEqual(localKind, localKindCopy)

		// if shouldUpdate {
		// 	debug.Info("debug local", "kind", localKind)
		// 	debug.Info("debug copy", "kind", localKindCopy)
		// }

		// this will always make reflect.DeepEqual return false
		// (at least while it's using resourceVersion, not generation)
		utils.UpdateLastGenerationAnnotation(localKindCopy)
	}

	// debug.Info("debug remote kind", "resource", req.NamespacedName.String(), "remoteKind", remoteKind)
	// debug.Info("debug local kind", "resource", req.NamespacedName.String(), "localKind", localKind)
	debug.Info("debug result", "resource", req.NamespacedName.String(), "result", result)
	debug.Info("debug bool", "resource", req.NamespacedName.String(), "shouldCreate", shouldCreate, "shouldUpdate", shouldUpdate, "shouldDelete", shouldDelete)

	if shouldCreate {
		log.Info("Creating local resource.", "name", req.NamespacedName.String())
		if err := r.Create(ctx, localKindCopy); err != nil {
			if apierrors.IsConflict(err) {
				return ctrl.Result{Requeue: true}, nil
			}

			return result, err
		}
	} else if shouldDelete {
		log.Info("Deleting local resource.", "name", req.NamespacedName.String())
		if err := r.Delete(ctx, localKindCopy); err != nil {
			if apierrors.IsNotFound(err) {
				return result, nil
			}

			return result, err
		}
	} else if shouldUpdate {
		log.Info("Updating local resource.", "name", req.NamespacedName.String())
		if err := r.Update(ctx, localKindCopy); err != nil {
			if apierrors.IsConflict(err) {
				return ctrl.Result{Requeue: true}, nil
			}

			return result, err
		}
	}

	return result, nil
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
