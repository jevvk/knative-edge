/*
Copyright 2022 jevv k.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package edge

import (
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cluster"

	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	"edge.jevv.dev/pkg/controllers"
)

type KnativeServiceV1Reconciler struct {
	controllers.EdgeOuterReconciler

	RemoteCluster cluster.Cluster

	ServiceReconciler       mirroringReconciler[*servingv1.Service]
	RevisionReconciler      mirroringReconciler[*servingv1.Revision]
	RouteReconciler         mirroringReconciler[*servingv1.Route]
	ConfigurationReconciler mirroringReconciler[*servingv1.Configuration]
}

func (r *KnativeServiceV1Reconciler) Setup(mgr ctrl.Manager) error {
	r.ServiceReconciler = mirroringReconciler[*servingv1.Service]{
		Name:          "KnativeServingV1/Service",
		HealthzName:   "healthz-knative-servingv1-service",
		Recorder:      mgr.GetEventRecorderFor("controller-knative-servingv1-service"),
		Scheme:        mgr.GetScheme(),
		RemoteCluster: r.RemoteCluster,
		RefGenerator: func() (*servingv1.Service, *servingv1.Service) {
			return &servingv1.Service{}, &servingv1.Service{}
		},
		RefMerger: func(src, dst *servingv1.Service) error {
			if src == nil {
				return nil
			}

			if dst == nil {
				*dst = servingv1.Service{}
			}

			dst.Spec = src.Spec

			return nil
		},
	}

	if err := r.ServiceReconciler.Setup(mgr); err != nil {
		return err
	}

	r.RevisionReconciler = mirroringReconciler[*servingv1.Revision]{
		Name:          "KnativeServingV1/Revision",
		HealthzName:   "healthz-knative-servingv1-revision",
		Recorder:      mgr.GetEventRecorderFor("controller-knative-servingv1-revision"),
		Scheme:        mgr.GetScheme(),
		RemoteCluster: r.RemoteCluster,
		RefGenerator: func() (*servingv1.Revision, *servingv1.Revision) {
			return &servingv1.Revision{}, &servingv1.Revision{}
		},
		RefMerger: func(src, dst *servingv1.Revision) error {
			if src == nil {
				return nil
			}

			if dst == nil {
				*dst = servingv1.Revision{}
			}

			dst.Spec = src.Spec

			return nil
		},
	}

	if err := r.RevisionReconciler.Setup(mgr); err != nil {
		return err
	}

	r.RouteReconciler = mirroringReconciler[*servingv1.Route]{
		Name:          "KnativeServingV1/Route",
		HealthzName:   "healthz-knative-servingv1-route",
		Recorder:      mgr.GetEventRecorderFor("controller-knative-servingv1-route"),
		Scheme:        mgr.GetScheme(),
		RemoteCluster: r.RemoteCluster,
		RefGenerator: func() (*servingv1.Route, *servingv1.Route) {
			return &servingv1.Route{}, &servingv1.Route{}
		},
		RefMerger: func(src, dst *servingv1.Route) error {
			if src == nil {
				return nil
			}

			if dst == nil {
				*dst = servingv1.Route{}
			}

			dst.Spec = src.Spec

			return nil
		},
	}

	if err := r.RouteReconciler.Setup(mgr); err != nil {
		return err
	}

	r.ConfigurationReconciler = mirroringReconciler[*servingv1.Configuration]{
		Name:          "KnativeServingV1/Configuration",
		HealthzName:   "healthz-knative-servingv1-configuration",
		Recorder:      mgr.GetEventRecorderFor("controller-knative-servingv1-configuration"),
		Scheme:        mgr.GetScheme(),
		RemoteCluster: r.RemoteCluster,
		RefGenerator: func() (*servingv1.Configuration, *servingv1.Configuration) {
			return &servingv1.Configuration{}, &servingv1.Configuration{}
		},
		RefMerger: func(src, dst *servingv1.Configuration) error {
			if src == nil {
				return nil
			}

			if dst == nil {
				*dst = servingv1.Configuration{}
			}

			dst.Spec = src.Spec

			return nil
		},
	}

	if err := r.ConfigurationReconciler.Setup(mgr); err != nil {
		return err
	}

	return nil
}

func (r *KnativeServiceV1Reconciler) GetInnerReconcilers() []controllers.EdgeReconciler {
	return []controllers.EdgeReconciler{
		&r.ServiceReconciler,
		&r.RevisionReconciler,
		&r.RouteReconciler,
		&r.ConfigurationReconciler,
	}
}
