package controllers

import (
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
)

type EdgeOuterReconciler interface {
	GetInnerReconcilers() []EdgeReconciler
}

type EdgeReconciler interface {
	Setup(mgr ctrl.Manager) error
	GetName() string
	GetHealthz() healthz.Checker
	GetHealthzName() string
}
