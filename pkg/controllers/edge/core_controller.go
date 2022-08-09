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
	corev1 "k8s.io/api/core/v1"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cluster"

	"edge.jevv.dev/pkg/controllers"
)

//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update

type CoreV1Reconciler struct {
	controllers.EdgeOuterReconciler

	RemoteCluster cluster.Cluster

	NamespaceReconciler mirroringReconciler[*corev1.Namespace]
	ConfigMapReconciler mirroringReconciler[*corev1.ConfigMap]
	SecretReconciler    mirroringReconciler[*corev1.Secret]
}

func (r *CoreV1Reconciler) Setup(mgr ctrl.Manager) error {
	r.NamespaceReconciler = mirroringReconciler[*corev1.Namespace]{
		Name:          "CoreV1/namespace",
		HealthzName:   "healthz-corev1-namespace",
		Recorder:      mgr.GetEventRecorderFor("controller-corev1-namespace"),
		Scheme:        mgr.GetScheme(),
		RemoteCluster: r.RemoteCluster,
		RefGenerator: func() (*corev1.Namespace, *corev1.Namespace) {
			return &corev1.Namespace{}, &corev1.Namespace{}
		},
		RefMerger: func(src, dst *corev1.Namespace) error {
			if src == nil {
				return nil
			}

			if dst == nil {
				*dst = corev1.Namespace{}
			}

			dst.Spec = src.Spec

			return nil
		},
	}

	if err := r.NamespaceReconciler.Setup(mgr); err != nil {
		return err
	}

	r.ConfigMapReconciler = mirroringReconciler[*corev1.ConfigMap]{
		Name:          "CoreV1/configMap",
		HealthzName:   "healthz-corev1-configmap",
		Recorder:      mgr.GetEventRecorderFor("controller-corev1-configmap"),
		Scheme:        mgr.GetScheme(),
		RemoteCluster: r.RemoteCluster,
		RefGenerator: func() (*corev1.ConfigMap, *corev1.ConfigMap) {
			return &corev1.ConfigMap{}, &corev1.ConfigMap{}
		},
		RefMerger: func(src, dst *corev1.ConfigMap) error {
			if src == nil {
				return nil
			}

			if dst == nil {
				*dst = corev1.ConfigMap{}
			}

			dst.Data = src.Data

			return nil
		},
	}

	if err := r.ConfigMapReconciler.Setup(mgr); err != nil {
		return err
	}

	r.SecretReconciler = mirroringReconciler[*corev1.Secret]{
		Name:          "CoreV1/secret",
		HealthzName:   "healthz-corev1-secret",
		Recorder:      mgr.GetEventRecorderFor("controller-corev1-secret"),
		Scheme:        mgr.GetScheme(),
		RemoteCluster: r.RemoteCluster,
		RefGenerator: func() (*corev1.Secret, *corev1.Secret) {
			return &corev1.Secret{}, &corev1.Secret{}
		},
		RefMerger: func(src, dst *corev1.Secret) error {
			if src == nil {
				return nil
			}

			if dst == nil {
				*dst = corev1.Secret{}
			}

			dst.Data = src.Data

			return nil
		},
	}

	if err := r.SecretReconciler.Setup(mgr); err != nil {
		return err
	}

	return nil
}

func (r *CoreV1Reconciler) GetInnerReconcilers() []controllers.EdgeReconciler {
	return []controllers.EdgeReconciler{
		&r.NamespaceReconciler,
		&r.ConfigMapReconciler,
		&r.SecretReconciler,
	}
}