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

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	klog "sigs.k8s.io/controller-runtime/pkg/log"

	cloudv1 "edge.jevv.dev/pkg/apis/cloud/v1"
	ws "edge.jevv.dev/pkg/refractor/websockets"
)

//+kubebuilder:rbac:groups=edge.jevv.dev,resources=edgeclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=edge.jevv.dev,resources=edgeclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=edge.jevv.dev,resources=edgeclusters/finalizers,verbs=update

// EdgeClusterReconciler reconciles a EdgeCluster object
type EdgeClusterReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder

	clientManager *ws.ClientManager
}

func (r *EdgeClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := klog.FromContext(ctx)

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

func (r *EdgeClusterReconciler) CleanUp() {
	if r.clientManager != nil {
		r.clientManager.Stop()
	}
}

func (r *EdgeClusterReconciler) Setup(mgr ctrl.Manager, clientManager *ws.ClientManager) error {
	r.clientManager = clientManager

	return ctrl.NewControllerManagedBy(mgr).
		For(&cloudv1.EdgeCluster{}).
		Complete(r)
}
