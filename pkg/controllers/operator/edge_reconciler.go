package operator

import (
	"context"
	"fmt"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/record"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	edgev1 "edge.jevv.dev/pkg/apis/edge/v1"
	operatorv1 "edge.jevv.dev/pkg/apis/operator/v1"
	"edge.jevv.dev/pkg/reflector"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type clusterWithExtras struct {
	cluster          cluster.Cluster
	ctx              context.Context
	stop             context.CancelFunc
	secretGeneration int64
}

type EdgeReconciler struct {
	client.Client

	Scheme   *runtime.Scheme
	Recorder record.EventRecorder

	ProxyImage       string
	ControllerImage  string
	RemoteSyncPeriod time.Duration

	remoteClusters map[string]clusterWithExtras
}

func (r *EdgeReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	shouldCreate := false
	shouldUpdate := false
	shouldDelete := false

	var edge operatorv1.Edge
	var edgeCluster edgev1.EdgeCluster

	if err := r.Get(ctx, request.NamespacedName, &edge); err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		shouldDelete = true
	}

	if !shouldDelete {
		namespacedSecretName := types.NamespacedName{Name: edge.Spec.SecretRef.Name, Namespace: edge.Spec.SecretRef.Namespace}
		var kubeconfigSecret corev1.Secret
		var remoteCluster cluster.Cluster

		if err := r.Get(ctx, namespacedSecretName, &kubeconfigSecret); err != nil {
			if apierrors.IsNotFound(err) {
				r.Recorder.Event(&edge, "RemoteClusterError", "no kubeconfig", "Referenced secret doesn't exist.")
			} else {
				r.Recorder.Event(&edge, "RemoteClusterError", "no kubeconfig", fmt.Sprintf("Kubeconfig secret couldn't be retrieved: %s", err))
			}

			return ctrl.Result{}, nil
		}

		remoteClusterKey := request.NamespacedName.String()
		existingRemoteCluster, remoteClusterExists := r.remoteClusters[remoteClusterKey]

		// if connection exists, but the secret changed, then disconnect the cluster
		if remoteClusterExists && existingRemoteCluster.secretGeneration != kubeconfigSecret.Generation {
			existingRemoteCluster.stop()

			delete(r.remoteClusters, remoteClusterKey)
			remoteClusterExists = false

			r.Recorder.Event(&edge, "RemoteClusterDisconnected", "remote cluster disconnected", "Remote cluster has been disconnected due to secret being changed.")
		}

		if remoteClusterExists {
			remoteCluster = existingRemoteCluster.cluster
		} else {
			// only create the remote cluster if we don't have it (or has been disconnected)
			kubeconfigData, exists := kubeconfigSecret.Data["kubeconfig"]

			if !exists {
				r.Recorder.Event(&edge, "RemoteClusterError", "kubeconfig missing", "There is no kubeconfig available in the referenced secret.")
				return ctrl.Result{}, nil
			}

			config, err := clientcmd.NewClientConfigFromBytes([]byte(kubeconfigData))

			if err != nil {
				r.Recorder.Event(&edge, "RemoteClusterError", "kubeconfig parsing", fmt.Sprintf("Kubeconfig couldn't be parsed: %s", err))
				return ctrl.Result{}, nil
			}

			kubeconfig, err := config.ClientConfig()

			if err != nil {
				r.Recorder.Event(&edge, "RemoteClusterError", "kubeconfig parsing", fmt.Sprintf("Kubeconfig couldn't be retrieved: %s", err))
				return ctrl.Result{}, nil
			}

			// resync cache once in a while
			remoteCluster, err = cluster.New(kubeconfig, func(o *cluster.Options) {
				o.SyncPeriod = &r.RemoteSyncPeriod
			})

			if err != nil {
				r.Recorder.Event(&edge, "RemoteClusterError", "remote cluster", fmt.Sprintf("Remote cluster couldn't be created: %s", err))
				return ctrl.Result{}, nil
			}

			// now create a new cluster connection
			// not sure if adding this to the manager while this is running is right
			// so this just uses the context to cancel

			remoteClusterCtx, remoteClusterStop := context.WithCancel(ctx)

			r.remoteClusters[remoteClusterKey] = clusterWithExtras{
				cluster:          remoteCluster,
				ctx:              remoteClusterCtx,
				stop:             remoteClusterStop,
				secretGeneration: kubeconfigSecret.Generation,
			}

			go remoteCluster.Start(remoteClusterCtx)

			r.Recorder.Event(&edge, "RemoteClusterConnected", "remote cluster created", "New remote cluster connection created.")
		}

		namespacedEdgeClusterName := types.NamespacedName{Name: edge.Spec.ClusterName, Namespace: ""}

		if err := remoteCluster.GetClient().Get(ctx, namespacedEdgeClusterName, &edgeCluster); err != nil {
			if apierrors.IsNotFound(err) {
				r.Recorder.Event(&edge, "EdgeClusterError", "no edgecluster", fmt.Sprintf("EdgeCluster %s hasn't been found in remote", edge.Spec.ClusterName))
				return ctrl.Result{}, nil
			}

			r.Recorder.Event(&edge, "EdgeClusterError", "remote get error", fmt.Sprintf("EdgeCluster %s couldn't be retrieved: %s", edge.Spec.ClusterName, err))
			return ctrl.Result{}, err
		}
	}

	namespacedDeploymentName := types.NamespacedName{Name: fmt.Sprintf("%s-controller", edge.Name), Namespace: edge.Namespace}
	var deployment appsv1.Deployment

	if err := r.Get(ctx, namespacedDeploymentName, &deployment); err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		shouldCreate = !shouldDelete
	}

	if !shouldCreate && !shouldUpdate {
		shouldUpdate = edge.Generation != edge.Status.EdgeObservedGeneration || edgeCluster.Generation != edge.Status.EdgeClusterObservedGeneration
	}

	if shouldCreate {
		r.buildDeployment(namespacedDeploymentName, &edge, &edgeCluster, &deployment)
		if err := r.Create(ctx, &deployment); err != nil {
			if apierrors.IsAlreadyExists(err) {
				return ctrl.Result{Requeue: true}, nil
			} else {
				return ctrl.Result{}, err
			}
		}

		r.Recorder.Event(&edge, "DeploymentCreated", "deployment created", "Knative Edge deployment has been created.")

		if err := r.updateEdgeStatus(ctx, &edge, &edgeCluster); err != nil {
			// TODO: log instead
			return ctrl.Result{}, err
		}
	} else if shouldUpdate {
		r.buildDeployment(namespacedDeploymentName, &edge, &edgeCluster, &deployment)

		if err := r.Update(ctx, &deployment); err != nil {
			if apierrors.IsConflict(err) || apierrors.IsNotFound(err) {
				return ctrl.Result{Requeue: true}, nil
			} else {
				return ctrl.Result{}, err
			}
		}

		r.Recorder.Event(&edge, "DeploymentUpdated", "deployment updated", "Knative Edge deployment has been updated.")

		if err := r.updateEdgeStatus(ctx, &edge, &edgeCluster); err != nil {
			// TODO: log instead
			return ctrl.Result{}, err
		}
	} else if shouldDelete {
		if err := r.Delete(ctx, &deployment); err != nil {
			if apierrors.IsNotFound(err) {
				return ctrl.Result{}, nil
			} else {
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

func (r *EdgeReconciler) buildDeployment(namespacedName types.NamespacedName, edge *operatorv1.Edge, edgeCluster *edgev1.EdgeCluster, deployment *appsv1.Deployment) {
	replicas := int32(1)
	labels := map[string]string{
		"app":                     "knative-edge-controller",
		"knative-edge-controller": namespacedName.Name,
	}

	proxyImage := r.ProxyImage

	if edge.Spec.OverrideProxyImage != "" {
		proxyImage = edge.Spec.OverrideProxyImage
	}

	deployment.Name = namespacedName.Name
	deployment.Namespace = namespacedName.Namespace

	deployment.Spec = appsv1.DeploymentSpec{
		Replicas: &replicas,
		Selector: &metav1.LabelSelector{
			MatchLabels: labels,
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: labels,
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Image: r.ControllerImage,
						Name:  "knative-edge-controller",
						Args: []string{
							"--envs", strings.Join(edgeCluster.Spec.Environments, ","),
							"--proxy-image", proxyImage,
						},
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "edgeconfig",
								MountPath: reflector.ConfigPath,
								ReadOnly:  true,
							},
						},
					},
				},
				Volumes: []corev1.Volume{
					{
						Name: "edgeconfig",
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName: edge.Spec.SecretRef.Name,
							},
						},
					},
				},
			},
		},
	}

	controllerutil.SetControllerReference(edge, deployment, r.Scheme)
}

func (r *EdgeReconciler) updateEdgeStatus(ctx context.Context, edge *operatorv1.Edge, edgeCluster *edgev1.EdgeCluster) error {
	edge.Status = operatorv1.EdgeStatus{
		Zone:                          edgeCluster.Spec.Zone,
		Region:                        edgeCluster.Spec.Region,
		Environments:                  edgeCluster.Spec.Environments,
		EdgeObservedGeneration:        edge.Generation,
		EdgeClusterObservedGeneration: edgeCluster.Generation,
	}

	return r.Status().Update(ctx, edge)
}

func (r *EdgeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1.Edge{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}
