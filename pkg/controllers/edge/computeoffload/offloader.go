package computeoffload

import (
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
