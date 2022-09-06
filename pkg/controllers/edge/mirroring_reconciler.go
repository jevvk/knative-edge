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
type kindPreProcessor[T client.Object] func(ctx context.Context, kind T) error

type MirroringReconciler[T client.Object] struct {
	client.Client
	controllers.EdgeReconciler

	Log           logr.Logger
	Scheme        *runtime.Scheme
	Recorder      record.EventRecorder
	RemoteCluster cluster.Cluster

	KindGenerator    kindGenerator[T]
	KindMerger       kindMerger[T]
	KindPreProcessor kindPreProcessor[T]
}

func (r *MirroringReconciler[T]) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.Log.Info("Started reconciling remote and local cluster.", "resource", req.NamespacedName.String())

	if r.Client == nil {
		r.Log.Info("nil client")
		return ctrl.Result{}, nil
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
			shouldCreate = true
		} else {
			return ctrl.Result{}, err
		}
	}

	if !shouldDelete {
		r.KindMerger(remoteKind, localKind)

		kindLabels := localKind.GetLabels()
		kindAnnotations := localKind.GetAnnotations()

		if kindLabels == nil {
			kindLabels = make(map[string]string)
			localKind.SetLabels(kindLabels)
		}

		if kindAnnotations == nil {
			kindAnnotations = make(map[string]string)
			localKind.SetAnnotations(kindAnnotations)
		}

		remoteGeneration := fmt.Sprint(remoteKind.GetGeneration())
		lastRemoteGeneration := kindAnnotations[controllers.LastRemoteGenerationAnnotation]

		if lastRemoteGeneration != remoteGeneration {
			shouldUpdate = lastRemoteGeneration != ""

			controllers.UpdateLabels(localKind)
			kindAnnotations[controllers.LastRemoteGenerationAnnotation] = remoteGeneration
		}
	}

	if r.KindPreProcessor != nil {
		if err := r.KindPreProcessor(ctx, localKind); err != nil {
			return ctrl.Result{}, err
		}
	}

	if shouldCreate {
		if err := r.Create(ctx, localKind); err != nil {
			if apierrors.IsConflict(err) {
				return ctrl.Result{Requeue: true}, nil
			}

			return ctrl.Result{}, err
		}
	} else if shouldDelete {
		if err := r.Delete(ctx, localKind); err != nil {
			if apierrors.IsNotFound(err) {
				return ctrl.Result{}, nil
			}

			return ctrl.Result{}, err
		}
	} else if shouldUpdate {
		if err := r.Update(ctx, localKind); err != nil {
			if apierrors.IsConflict(err) {
				return ctrl.Result{Requeue: true}, nil
			}

			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *MirroringReconciler[T]) NewControllerManagedBy(mgr ctrl.Manager) *builder.Builder {
	return ctrl.NewControllerManagedBy(mgr).
		// local watch
		For(
			r.KindGenerator(),
			builder.WithPredicates(
				predicate.And(
					predicate.GenerationChangedPredicate{},
					controllers.NotChangedByEdgeControllers{},
				),
			),
		).
		// remote watch
		Watches(
			source.NewKindWithCache(r.KindGenerator(), r.RemoteCluster.GetCache()),
			&handler.EnqueueRequestForObject{},
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		)
}
