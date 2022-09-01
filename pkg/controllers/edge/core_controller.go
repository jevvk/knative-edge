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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"edge.jevv.dev/pkg/controllers"
)

//+kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete

type CoreV1Reconciler struct {
	controllers.EdgeOuterReconciler

	RemoteCluster cluster.Cluster

	NamespaceReconciler mirroringReconciler[*corev1.Namespace]
	ConfigMapReconciler mirroringReconciler[*corev1.ConfigMap]
	SecretReconciler    mirroringReconciler[*corev1.Secret]
}

func (r *CoreV1Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.NamespaceReconciler = mirroringReconciler[*corev1.Namespace]{
		Name:          "CoreV1/namespace",
		HealthzName:   "healthz-corev1-namespace",
		Recorder:      mgr.GetEventRecorderFor("controller-corev1-namespace"),
		Scheme:        mgr.GetScheme(),
		RemoteCluster: r.RemoteCluster,
		KindGenerator: func() *corev1.Namespace {
			return &corev1.Namespace{}
		},
		KindMerger: func(src, dst *corev1.Namespace) error {
			if src == nil {
				return nil
			}

			if dst == nil {
				*dst = corev1.Namespace{}
			}

			dst.ObjectMeta = metav1.ObjectMeta{
				Name:        src.ObjectMeta.Name,
				Namespace:   src.ObjectMeta.Namespace,
				Annotations: src.ObjectMeta.Annotations,
				Labels:      src.ObjectMeta.Labels,
			}

			return nil
		},
	}

	if err := r.NamespaceReconciler.SetupWithManager(mgr); err != nil {
		return err
	}

	r.ConfigMapReconciler = mirroringReconciler[*corev1.ConfigMap]{
		Name:          "CoreV1/configMap",
		HealthzName:   "healthz-corev1-configmap",
		Recorder:      mgr.GetEventRecorderFor("controller-corev1-configmap"),
		Scheme:        mgr.GetScheme(),
		RemoteCluster: r.RemoteCluster,
		KindGenerator: func() *corev1.ConfigMap {
			return &corev1.ConfigMap{}
		},
		KindMerger: func(src, dst *corev1.ConfigMap) error {
			if src == nil {
				return nil
			}

			if dst == nil {
				*dst = corev1.ConfigMap{}
			}

			dst.ObjectMeta = metav1.ObjectMeta{
				Name:        src.ObjectMeta.Name,
				Namespace:   src.ObjectMeta.Namespace,
				Annotations: src.ObjectMeta.Annotations,
				Labels:      src.ObjectMeta.Labels,
			}

			dst.Data = src.Data

			return nil
		},
	}

	if err := r.ConfigMapReconciler.SetupWithManager(mgr); err != nil {
		return err
	}

	r.SecretReconciler = mirroringReconciler[*corev1.Secret]{
		Name:          "CoreV1/secret",
		HealthzName:   "healthz-corev1-secret",
		Recorder:      mgr.GetEventRecorderFor("controller-corev1-secret"),
		Scheme:        mgr.GetScheme(),
		RemoteCluster: r.RemoteCluster,
		KindGenerator: func() *corev1.Secret {
			return &corev1.Secret{}
		},
		KindMerger: func(src, dst *corev1.Secret) error {
			if src == nil {
				return nil
			}

			if dst == nil {
				*dst = corev1.Secret{}
			}

			dst.ObjectMeta = metav1.ObjectMeta{
				Name:        src.ObjectMeta.Name,
				Namespace:   src.ObjectMeta.Namespace,
				Annotations: src.ObjectMeta.Annotations,
				Labels:      src.ObjectMeta.Labels,
			}

			dst.Data = src.Data

			return nil
		},
	}

	if err := r.SecretReconciler.SetupWithManager(mgr); err != nil {
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
