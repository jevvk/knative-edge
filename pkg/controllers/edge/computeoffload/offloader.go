package computeoffload

import (
	ctrl "sigs.k8s.io/controller-runtime"

	"edge.jevv.dev/pkg/controllers"
)

type Offloader struct {
	kServiceReconciler  KServiceReconciler
	kRevisionReconciler KRevisionReconciler
}

func New(proxyImage string) Offloader {
	return Offloader{
		kServiceReconciler:  KServiceReconciler{},
		kRevisionReconciler: KRevisionReconciler{ProxyImage: proxyImage},
	}
}

func (r Offloader) GetReconcilers() []controllers.EdgeReconciler {
	return []controllers.EdgeReconciler{&r.kServiceReconciler, &r.kRevisionReconciler}
}

func (r *Offloader) SetupWithManager(mgr ctrl.Manager) error {
	if err := r.kServiceReconciler.SetupWithManager(mgr); err != nil {
		return err
	}

	if err := r.kRevisionReconciler.SetupWithManager(mgr); err != nil {
		return err
	}

	return nil
}
