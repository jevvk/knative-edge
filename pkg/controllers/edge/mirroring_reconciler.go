package edge

import (
	"context"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"edge.jevv.dev/pkg/controllers"
)

type refGenerator[T client.Object] func() (T, T)
type refMerger[T client.Object] func(src, dst T) error

type mirroringReconciler[T client.Object] struct {
	client.Client
	controllers.EdgeReconciler

	Scheme        *runtime.Scheme
	Recorder      record.EventRecorder
	RemoteCluster cluster.Cluster

	Name         string
	HealthzName  string
	RefGenerator refGenerator[T]
	RefMerger    refMerger[T]
}

func (r *mirroringReconciler[T]) GetName() string {
	return r.Name
}

func (r *mirroringReconciler[T]) GetHealthz() healthz.Checker {
	return nil
}

func (r *mirroringReconciler[T]) GetHealthzName() string {
	return r.HealthzName
}

func (r *mirroringReconciler[T]) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	localKind, remoteKind := r.RefGenerator()

	shouldCreate := false
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

	r.RefMerger(remoteKind, localKind)

	if !shouldDelete {
		kindLabels := localKind.GetLabels()

		if kindLabels == nil {
			kindLabels = make(map[string]string)
			localKind.SetLabels(kindLabels)
		}

		kindLabels[controllers.ManagedLabel] = "true"
		kindLabels[controllers.ManagedByLabel] = "knative-edge-controller"
		kindLabels[controllers.CreatedByLabel] = "knative-edge-controller"
	}

	if shouldCreate {
		return ctrl.Result{}, r.Create(ctx, localKind)
	} else if shouldDelete {
		return ctrl.Result{}, r.Delete(ctx, localKind)
	} else {
		return ctrl.Result{}, r.Update(ctx, localKind)
	}
}

func (r *mirroringReconciler[T]) Setup(mgr ctrl.Manager) error {
	kind1, kind2 := r.RefGenerator()

	return ctrl.NewControllerManagedBy(mgr).
		// local watch
		Watches(
			&source.Kind{Type: kind1},
			&handler.EnqueueRequestForObject{},
		).
		// remote watch
		Watches(
			source.NewKindWithCache(kind2, r.RemoteCluster.GetCache()),
			&handler.EnqueueRequestForObject{},
		).
		Complete(r)
}
