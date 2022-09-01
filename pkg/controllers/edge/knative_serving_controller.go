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
	"strings"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cluster"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	"edge.jevv.dev/pkg/controllers"
)

//+kubebuilder:rbac:groups=serving.knative.dev,resources=services,verbs=get;list;watch;create;update;patch;delete

type KnativeServiceV1Reconciler struct {
	controllers.EdgeOuterReconciler

	RemoteCluster cluster.Cluster

	ServiceReconciler mirroringReconciler[*servingv1.Service]

	// NOTE:
	// disabled because knative recommends only changing the service
	// since revisions, routes, and configurations are managed by
	// the service
	// RevisionReconciler      mirroringReconciler[*servingv1.Revision]
	// RouteReconciler         mirroringReconciler[*servingv1.Route]
	// ConfigurationReconciler mirroringReconciler[*servingv1.Configuration]
}

func (r *KnativeServiceV1Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.ServiceReconciler = mirroringReconciler[*servingv1.Service]{
		Name:          "KnativeServingV1/Service",
		HealthzName:   "healthz-knative-servingv1-service",
		Recorder:      mgr.GetEventRecorderFor("controller-knative-servingv1-service"),
		Scheme:        mgr.GetScheme(),
		RemoteCluster: r.RemoteCluster,
		KindGenerator: func() *servingv1.Service {
			return &servingv1.Service{}
		},
		KindMerger: func(src, dst *servingv1.Service) error {
			if src == nil {
				return nil
			}

			if dst == nil {
				*dst = servingv1.Service{}
			}

			dst.ObjectMeta = metav1.ObjectMeta{
				Name:        src.ObjectMeta.Name,
				Namespace:   src.ObjectMeta.Namespace,
				Annotations: src.ObjectMeta.Annotations,
				Labels:      src.ObjectMeta.Labels,
			}

			src.Spec.DeepCopyInto(&dst.Spec)

			annotations := dst.GetAnnotations()

			if annotations == nil {
				annotations = make(map[string]string)
				dst.SetAnnotations(annotations)
			}

			if src.Status.URL != nil {
				url := src.Status.URL.String()

				if !strings.HasSuffix(url, "/") {
					url += "/"
				}

				annotations[controllers.RemoteUrlAnnotation] = url
			}

			return nil
		},
	}

	if err := r.ServiceReconciler.SetupWithManager(mgr); err != nil {
		return err
	}

	// r.RevisionReconciler = mirroringReconciler[*servingv1.Revision]{
	// 	Name:          "KnativeServingV1/Revision",
	// 	HealthzName:   "healthz-knative-servingv1-revision",
	// 	Recorder:      mgr.GetEventRecorderFor("controller-knative-servingv1-revision"),
	// 	Scheme:        mgr.GetScheme(),
	// 	RemoteCluster: r.RemoteCluster,
	// 	KindGenerator: func() (*servingv1.Revision) {
	// 		return &servingv1.Revision{}
	// 	},
	// 	KindMerger: func(src, dst *servingv1.Revision) error {
	// 		if src == nil {
	// 			return nil
	// 		}

	// 		if dst == nil {
	// 			*dst = servingv1.Revision{}
	// 		}

	// 		dst.Spec = src.Spec

	// 		return nil
	// 	},
	// }

	// if err := r.RevisionReconciler.SetupWithManager(mgr); err != nil {
	// 	return err
	// }

	// r.RouteReconciler = mirroringReconciler[*servingv1.Route]{
	// 	Name:          "KnativeServingV1/Route",
	// 	HealthzName:   "healthz-knative-servingv1-route",
	// 	Recorder:      mgr.GetEventRecorderFor("controller-knative-servingv1-route"),
	// 	Scheme:        mgr.GetScheme(),
	// 	RemoteCluster: r.RemoteCluster,
	// 	KindGenerator: func() (*servingv1.Route) {
	// 		return &servingv1.Route{}
	// 	},
	// 	KindMerger: func(src, dst *servingv1.Route) error {
	// 		if src == nil {
	// 			return nil
	// 		}

	// 		if dst == nil {
	// 			*dst = servingv1.Route{}
	// 		}

	// 		dst.Spec = src.Spec

	// 		return nil
	// 	},
	// }

	// if err := r.RouteReconciler.SetupWithManager(mgr); err != nil {
	// 	return err
	// }

	// r.ConfigurationReconciler = mirroringReconciler[*servingv1.Configuration]{
	// 	Name:          "KnativeServingV1/Configuration",
	// 	HealthzName:   "healthz-knative-servingv1-configuration",
	// 	Recorder:      mgr.GetEventRecorderFor("controller-knative-servingv1-configuration"),
	// 	Scheme:        mgr.GetScheme(),
	// 	RemoteCluster: r.RemoteCluster,
	// 	KindGenerator: func() (*servingv1.Configuration) {
	// 		return &servingv1.Configuration{}
	// 	},
	// 	KindMerger: func(src, dst *servingv1.Configuration) error {
	// 		if src == nil {
	// 			return nil
	// 		}

	// 		if dst == nil {
	// 			*dst = servingv1.Configuration{}
	// 		}

	// 		dst.Spec = src.Spec

	// 		return nil
	// 	},
	// }

	// if err := r.ConfigurationReconciler.SetupWithManager(mgr); err != nil {
	// 	return err
	// }

	return nil
}

func (r *KnativeServiceV1Reconciler) GetInnerReconcilers() []controllers.EdgeReconciler {
	return []controllers.EdgeReconciler{
		&r.ServiceReconciler,
		// &r.RevisionReconciler,
		// &r.RouteReconciler,
		// &r.ConfigurationReconciler,
	}
}
