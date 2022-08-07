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
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ws "edge.knative.dev/pkg/apiproxy/websockets"

	edgev1 "edge.knative.dev/pkg/apis/edge/v1"
	corev1 "k8s.io/api/core/v1"
)

//+kubebuilder:rbac:groups=serving.knative.dev,resources=services,verbs=get;list;watch

// EdgeClusterReconciler reconciles a EdgeCluster object
type EdgeNamespacesReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder

	clientManager *ws.ClientManager
}

func (r *EdgeNamespacesReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// log := log.FromContext(ctx)

	var edgeNamespace edgev1.EdgeResource

	if err := r.Get(ctx, req.NamespacedName, &edgeNamespace); err != nil {
		// TODO
	}

	var namespace corev1.Namespace

	if err := r.Get(ctx, req.NamespacedName, &namespace); err != nil {
		// TODO
	}

	return ctrl.Result{}, nil
}

func (r *EdgeNamespacesReconciler) Setup(mgr ctrl.Manager, clientManager *ws.ClientManager) error {
	r.clientManager = clientManager

	return ctrl.NewControllerManagedBy(mgr).
		For(&edgev1.EdgeResource{}).
		Complete(r)
}
