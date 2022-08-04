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

package controllers

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	ws "edge.knative.dev/pkg/cloud/apiproxy/websockets"

	"sigs.k8s.io/controller-runtime/pkg/builder"   // Required for Watching
	"sigs.k8s.io/controller-runtime/pkg/handler"   // Required for Watching
	"sigs.k8s.io/controller-runtime/pkg/predicate" // Required for Watching

	"sigs.k8s.io/controller-runtime/pkg/source" // Required for Watching

	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

// EdgeClusterReconciler reconciles a EdgeCluster object
type KservicesReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder

	clientManager *ws.ClientManager
}

//+kubebuilder:rbac:groups=knative.dev,resources=kservices,verbs=get,list,watch
func (r *KservicesReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	var kservice servingv1.Service

	if err := r.Get(ctx, req.NamespacedName, &kservice); err != nil {
		log.Error(err, "unable to fetch service", "controller", "Services")

		if errors.IsNotFound(err) {
			r.clientManager.DeleteKService(req.NamespacedName.String())
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	err := r.clientManager.UpdateKService(&kservice)

	return ctrl.Result{}, err
}

func (r *KservicesReconciler) Setup(mgr ctrl.Manager, clientManager *ws.ClientManager) error {
	r.clientManager = clientManager

	return ctrl.NewControllerManagedBy(mgr).
		For(&servingv1.Service{}).
		Watches(
			&source.Kind{Type: &servingv1.Service{}},
			&handler.EnqueueRequestForObject{},
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Complete(r)
}
