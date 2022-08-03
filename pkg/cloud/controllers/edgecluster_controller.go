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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	cloudv1 "edge.knative.dev/pkg/apis/cloud/v1"
	"edge.knative.dev/pkg/cloud/apiproxy/clients"
)

// EdgeClusterReconciler reconciles a EdgeCluster object
type EdgeClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	clientManager *clients.ClientManager
}

//+kubebuilder:rbac:groups=cloud.edge.knative.dev,resources=edgeclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cloud.edge.knative.dev,resources=edgeclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cloud.edge.knative.dev,resources=edgeclusters/finalizers,verbs=update
func (r *EdgeClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	var cluster cloudv1.EdgeCluster

	if err := r.Get(ctx, req.NamespacedName, &cluster); err != nil {
		log.Error(err, "unable to fetch edgecluster", "controller", "EdgeCluster")

		if errors.IsNotFound(err) {
			r.clientManager.DeleteCluster(req.NamespacedName.String())
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	statusUpdate := r.clientManager.UpdateCluster(&cluster)

	if statusUpdate != nil {
		if err := r.Status().Update(ctx, statusUpdate); err != nil {
			log.Error(err, "unable to update edgecluster status", "controller", "EdgeCluster")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *EdgeClusterReconciler) Stop() {
	if r.clientManager != nil {
		r.clientManager.Stop()
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *EdgeClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.clientManager = clients.New()

	return ctrl.NewControllerManagedBy(mgr).
		For(&cloudv1.EdgeCluster{}).
		Complete(r)
}
