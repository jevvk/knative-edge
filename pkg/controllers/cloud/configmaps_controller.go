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

package cloud

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	ws "edge.jevv.dev/pkg/refractor/websockets"
)

//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch

// EdgeClusterReconciler reconciles a EdgeCluster object
type ConfigMapsReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder

	clientManager *ws.ClientManager
}

func (r *ConfigMapsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	var secret corev1.ConfigMap

	if err := r.Get(ctx, req.NamespacedName, &secret); err != nil {
		log.Error(err, "unable to fetch secret", "controller", "ConfigMaps")

		if errors.IsNotFound(err) {
			r.clientManager.DeleteConfigMap(req.NamespacedName.String())
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	err := r.clientManager.UpdateConfigMap(&secret)

	return ctrl.Result{}, err
}

func (r *ConfigMapsReconciler) Setup(mgr ctrl.Manager, clientManager *ws.ClientManager) error {
	r.clientManager = clientManager

	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.ConfigMap{}).
		Complete(r)
}
