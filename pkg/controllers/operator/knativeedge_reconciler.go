package operator

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/record"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
	"sigs.k8s.io/controller-runtime/pkg/source"

	edgev1 "edge.jevv.dev/pkg/apis/edge/v1"
	operatorv1 "edge.jevv.dev/pkg/apis/operator/v1"
	"edge.jevv.dev/pkg/controllers"
	"edge.jevv.dev/pkg/reflector"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//+kubebuilder:rbac:groups=operator.edge.jevv.dev,resources=knativeedges,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.edge.jevv.dev,resources=knativeedges/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=operator.edge.jevv.dev,resources=knativeedges/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,namespace=knative-edge-system,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=secrets,namespace=knative-edge-system,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;update;patch
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get

type clusterWithExtras struct {
	cluster          cluster.Cluster
	ctx              context.Context
	stop             context.CancelFunc
	secretGeneration int64
}

type EdgeReconciler struct {
	client.Client
	inject.APIReader

	reader client.Reader

	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder

	// same cluster, different cache
	SystemCluster cluster.Cluster

	ProxyImage       string
	ControllerImage  string
	RemoteSyncPeriod time.Duration

	mgr            ctrl.Manager
	remoteClusters map[string]clusterWithExtras
}

func (r *EdgeReconciler) InjectAPIReader(reader client.Reader) error {
	r.reader = reader
	return nil
}

func (r *EdgeReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	var edge operatorv1.KnativeEdge

	if err := r.Get(ctx, request.NamespacedName, &edge); err != nil && !apierrors.IsNotFound(err) {
		return ctrl.Result{}, err
	}

	kubeconfigSecret, err := r.reconcileCluster(ctx, &edge)
	ctx = withKubeconfigInContext(ctx, kubeconfigSecret)

	if err != nil {
		return ctrl.Result{}, err
	}

	result, err := r.reconcileSecret(ctx, &edge)

	if err != nil || result.Requeue || result.RequeueAfter > 0 {
		return result, err
	}

	return r.reconcileDeployment(ctx, &edge)
}

func (r *EdgeReconciler) reconcileCluster(ctx context.Context, edge *operatorv1.KnativeEdge) (*corev1.Secret, error) {
	if edge == nil {
		return nil, nil
	}

	var kubeconfigSecret corev1.Secret

	r.Log.Info("Reconciling KnativeEdge remote cluster.", "KnativeEdge/Name", edge.Name, "KnativeEdge/Namespace", edge.Namespace)

	if edge.Spec.SecretRef != nil {
		r.Log.Info("Reconciling KnativeEdge remote cluster with referenced secret.", "KnativeEdge/Name", edge.Name, "KnativeEdge/Namespace", edge.Namespace, "KnativeEdge/Secret/Name", edge.Spec.SecretRef.Name, "KnativeEdge/Secret/Namespace", edge.Spec.SecretRef.Namespace)

		namespacedSecretName := types.NamespacedName{Name: edge.Spec.SecretRef.Name, Namespace: edge.Spec.SecretRef.Namespace}

		var remoteCluster cluster.Cluster

		r.Log.Info("Retrieving KnativeEdge referenced secret.", "secret", namespacedSecretName.String())

		if err := r.reader.Get(ctx, namespacedSecretName, &kubeconfigSecret); err != nil {
			if apierrors.IsNotFound(err) {
				r.Recorder.Event(edge, "Warning", "RemoteKubeconfigMissing", "Referenced secret doesn't exist.")
			} else {
				r.Recorder.Event(edge, "Warning", "RemoteKubeconfigError", fmt.Sprintf("Kubeconfig secret couldn't be retrieved: %s", err))
			}

			return nil, nil
		}

		remoteClusterKey := getRemoteClusterName(edge.Name, controllers.SystemNamespace).String()
		existingRemoteCluster, remoteClusterExists := r.remoteClusters[remoteClusterKey]

		// if connection exists, but the secret changed, then disconnect the cluster
		if remoteClusterExists && existingRemoteCluster.secretGeneration != kubeconfigSecret.Generation {
			existingRemoteCluster.stop()

			delete(r.remoteClusters, remoteClusterKey)
			remoteClusterExists = false

			r.Recorder.Event(edge, "Normal", "RemoteClusterDisconnected", "Remote cluster has been disconnected due to secret being changed.")
		}

		if !remoteClusterExists {
			// only create the remote cluster if we don't have it (or has been disconnected)
			kubeconfigData, exists := kubeconfigSecret.Data["kubeconfig"]

			if !exists {
				r.Recorder.Event(edge, "Warning", "KubeconfigMissing", "There is no kubeconfig available in the referenced secret.")
				return nil, nil
			}

			config, err := clientcmd.NewClientConfigFromBytes([]byte(kubeconfigData))

			if err != nil {
				r.Recorder.Event(edge, "Warning", "KubeconfigParsingError", fmt.Sprintf("Kubeconfig couldn't be parsed: %s", err))
				return nil, nil
			}

			kubeconfig, err := config.ClientConfig()

			if err != nil {
				r.Recorder.Event(edge, "Warning", "KubeconfigParsingError", fmt.Sprintf("Kubeconfig couldn't be retrieved: %s", err))
				return nil, nil
			}

			// resync cache once in a while
			// TODO: check later if disabling cache would be better
			remoteCluster, err = cluster.New(kubeconfig, func(o *cluster.Options) {
				o.SyncPeriod = &r.RemoteSyncPeriod
				o.Scheme = r.Scheme
				// // disable cache for reading from remote
				// o.ClientDisableCacheFor = []client.Object{&edgev1.EdgeCluster{}}
			})

			if err != nil {
				r.Recorder.Event(edge, "Warning", "RemoteClusterError", fmt.Sprintf("Remote cluster couldn't be created: %s", err))
				return nil, nil
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

			// inject dependencies
			if err := r.mgr.SetFields(remoteCluster); err != nil {
				r.Recorder.Event(edge, "Warning", "RemoteClusterInternalError", fmt.Sprintf("Remote setup failed: %s", err))
				return nil, nil
			}

			go remoteCluster.Start(remoteClusterCtx)

			r.Recorder.Event(edge, "Normal", "RemoteClusterConnected", "New remote cluster connection created.")
		}
	}

	r.Log.Info("KnativeEdge remote cluster reconciliation finished.", "KnativeEdge/Name", edge.Name, "KnativeEdge/Namespace", edge.Namespace)

	return &kubeconfigSecret, nil
}

func (r *EdgeReconciler) reconcileSecret(ctx context.Context, edge *operatorv1.KnativeEdge) (ctrl.Result, error) {
	if edge != nil {
		r.Log.Info("Reconciling KnativeEdge secret.", "KnativeEdge/Name", edge.Name, "KnativeEdge/Namespace", edge.Namespace)
	} else {
		r.Log.Info("Reconciling nil KnativeEdge secret.")
	}

	systemClient := r.SystemCluster.GetClient()

	shouldCreate := false
	shouldUpdate := false
	shouldDelete := false

	var refSecret corev1.Secret

	if edge == nil || edge.DeletionTimestamp != nil {
		shouldDelete = true
	}

	// retrieve kubeconfig from context
	withKubeconfigFromContext(ctx, &refSecret)

	namespacedSecretName := getSecretName(edge.Name, controllers.SystemNamespace)
	var secret corev1.Secret

	// if the name and namespace match, just skip copying
	if refSecret.Name == namespacedSecretName.Name && refSecret.Namespace == namespacedSecretName.Namespace {
		return ctrl.Result{}, nil
	}

	if err := systemClient.Get(ctx, namespacedSecretName, &secret); err != nil {
		r.Log.Info("Retrieving KnativeEdge system secret.", "secret", namespacedSecretName.String())

		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		if edge.Spec.SecretRef != nil {
			shouldCreate = !shouldDelete
		} else {
			shouldDelete = false
		}
	}

	if !shouldCreate && !shouldUpdate {
		shouldUpdate =
			secret.Annotations == nil ||
				secret.Annotations[controllers.ObserverGenerationAnnotation] != fmt.Sprint(refSecret.Generation)
	}

	if shouldCreate {
		r.Log.Info("Creating KnativeEdge system secret.", "secret", namespacedSecretName.String())

		r.buildSecret(namespacedSecretName, edge, &refSecret, &secret)
		if err := systemClient.Create(ctx, &secret); err != nil {
			if apierrors.IsAlreadyExists(err) {
				return ctrl.Result{Requeue: true}, nil
			} else {
				return ctrl.Result{}, err
			}
		}

		r.Recorder.Event(edge, "Normal", "SecretCreated", "Knative Edge config has been created.")
	} else if shouldUpdate {
		r.Log.Info("Updating KnativeEdge system secret.", "secret", namespacedSecretName.String())

		r.buildSecret(namespacedSecretName, edge, &refSecret, &secret)
		if err := systemClient.Update(ctx, &secret); err != nil {
			if apierrors.IsConflict(err) || apierrors.IsNotFound(err) {
				return ctrl.Result{Requeue: true}, nil
			} else {
				return ctrl.Result{}, err
			}
		}

		r.Recorder.Event(edge, "Normal", "SecretUpdated", "Knative Edge config has been updated.")
	} else if shouldDelete {
		r.Log.Info("Deleting KnativeEdge system secret.", "secret", namespacedSecretName.String())

		if err := systemClient.Delete(ctx, &secret); err != nil {
			if apierrors.IsNotFound(err) {
				return ctrl.Result{}, nil
			} else {
				return ctrl.Result{}, err
			}
		}
	}

	r.Log.Info("KnativeEdge system secret reconciliation finished.", "deployment", namespacedSecretName.String())

	return ctrl.Result{}, nil
}

func (r *EdgeReconciler) reconcileDeployment(ctx context.Context, edge *operatorv1.KnativeEdge) (ctrl.Result, error) {
	if edge != nil {
		r.Log.Info("Reconciling KnativeEdge deployment.", "KnativeEdge/Name", edge.Name, "KnativeEdge/Namespace", edge.Namespace)
	} else {
		r.Log.Info("Reconciling nil KnativeEdge deployment.")
	}

	systemClient := r.SystemCluster.GetClient()

	shouldCreate := false
	shouldUpdate := false
	shouldDelete := false

	var edgeCluster edgev1.EdgeCluster

	if edge == nil || edge.DeletionTimestamp != nil {
		shouldDelete = true
	}

	if !shouldDelete && edge.Spec.SecretRef != nil {
		namespacedEdgeClusterName := types.NamespacedName{Name: edge.Spec.ClusterName, Namespace: ""}

		remoteClusterKey := getRemoteClusterName(edge.Name, controllers.SystemNamespace).String()
		existingRemoteCluster, remoteClusterExists := r.remoteClusters[remoteClusterKey]

		if !remoteClusterExists {
			// event was already logged, retry after a while and check if the kubeconfig is valid
			return ctrl.Result{RequeueAfter: time.Second * 30}, nil
		}

		remoteCluster := existingRemoteCluster.cluster

		if err := remoteCluster.GetClient().Get(ctx, namespacedEdgeClusterName, &edgeCluster); err != nil {
			if apierrors.IsNotFound(err) {
				r.Recorder.Event(edge, "Warning", "EdgeClusterError", fmt.Sprintf("EdgeCluster %s hasn't been found in remote", edge.Spec.ClusterName))
				return ctrl.Result{}, nil
			}

			r.Recorder.Event(edge, "Warning", "EdgeClusterError", fmt.Sprintf("EdgeCluster %s couldn't be retrieved: %s", edge.Spec.ClusterName, err))
			return ctrl.Result{}, err
		}
	}

	namespacedDeploymentName := getDeploymentName(edge.Name, controllers.SystemNamespace)
	namespacedSecretName := getSecretName(edge.Name, controllers.SystemNamespace)

	var deployment appsv1.Deployment

	if err := systemClient.Get(ctx, namespacedDeploymentName, &deployment); err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		shouldCreate = !shouldDelete
	}

	if !shouldCreate && !shouldUpdate {
		shouldUpdate =
			deployment.Generation != edge.Status.DeploymentObservedGeneration ||
				edge.Generation != edge.Status.EdgeObservedGeneration ||
				edgeCluster.Generation != edge.Status.EdgeClusterObservedGeneration
	}

	if shouldCreate {
		r.Log.Info("Creating KnativeEdge system deployment.", "deployment", namespacedSecretName.String())

		r.buildDeployment(namespacedDeploymentName, namespacedSecretName, edge, &edgeCluster, &deployment)
		if err := systemClient.Create(ctx, &deployment); err != nil {
			if apierrors.IsAlreadyExists(err) {
				return ctrl.Result{Requeue: true}, nil
			} else {
				return ctrl.Result{}, err
			}
		}

		r.Recorder.Event(edge, "Normal", "DeploymentCreated", "Knative Edge deployment has been created.")

		if err := r.updateEdgeStatus(ctx, edge, &edgeCluster, &deployment); err != nil {
			// TODO: log instead
			return ctrl.Result{}, err
		}
	} else if shouldUpdate {
		r.Log.Info("Updating KnativeEdge system deployment.", "deployment", namespacedSecretName.String())

		r.buildDeployment(namespacedDeploymentName, namespacedSecretName, edge, &edgeCluster, &deployment)
		if err := systemClient.Update(ctx, &deployment); err != nil {
			if apierrors.IsConflict(err) || apierrors.IsNotFound(err) {
				return ctrl.Result{Requeue: true}, nil
			} else {
				return ctrl.Result{}, err
			}
		}

		r.Recorder.Event(edge, "Normal", "DeploymentUpdated", "Knative Edge deployment has been updated.")

		if err := r.updateEdgeStatus(ctx, edge, &edgeCluster, &deployment); err != nil {
			// TODO: log instead
			return ctrl.Result{}, err
		}
	} else if shouldDelete {
		r.Log.Info("Deleting KnativeEdge system deployment.", "deployment", namespacedSecretName.String())

		if err := systemClient.Delete(ctx, &deployment); err != nil {
			if apierrors.IsNotFound(err) {
				return ctrl.Result{}, nil
			} else {
				return ctrl.Result{}, err
			}
		}
	}

	r.Log.Info("KnativeEdge system deployment reconciliation finished.", "deployment", namespacedSecretName.String())

	return ctrl.Result{}, nil
}

func getRemoteClusterName(name, namespace string) types.NamespacedName {
	return types.NamespacedName{Name: fmt.Sprintf("%s-remote-cluster", name), Namespace: namespace}
}

func getSecretName(name, namespace string) types.NamespacedName {
	return types.NamespacedName{Name: fmt.Sprintf("%s-edgeconfig", name), Namespace: namespace}
}

func getDeploymentName(name, namespace string) types.NamespacedName {
	return types.NamespacedName{Name: fmt.Sprintf("%s-controller", name), Namespace: namespace}
}

func (r *EdgeReconciler) buildSecret(namespacedName types.NamespacedName, edge *operatorv1.KnativeEdge, src, dst *corev1.Secret) {
	if src == nil {
		return
	}

	if dst == nil {
		*dst = corev1.Secret{}
	}

	if dst.Annotations == nil {
		dst.Annotations = make(map[string]string)
	}

	dst.Name = namespacedName.Name
	dst.Namespace = namespacedName.Namespace

	dst.Labels = getLabels(namespacedName)
	dst.Annotations[controllers.ObserverGenerationAnnotation] = fmt.Sprint(src.GetGeneration())

	dst.Data = src.Data

	controllerutil.SetControllerReference(edge, dst, r.Scheme)
}

func (r *EdgeReconciler) buildDeployment(namespacedName, namespacedSecretName types.NamespacedName, edge *operatorv1.KnativeEdge, edgeCluster *edgev1.EdgeCluster, deployment *appsv1.Deployment) {
	replicas := int32(1)
	labels := getLabels(namespacedName)

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
								SecretName: namespacedSecretName.Name,
							},
						},
					},
				},
			},
		},
	}

	controllerutil.SetControllerReference(edge, deployment, r.Scheme)
}

func getLabels(namespacedName types.NamespacedName) map[string]string {
	return map[string]string{
		controllers.AppLabel:        "controller",
		controllers.ServiceLabel:    "knative-edge",
		controllers.ControllerLabel: namespacedName.Name,
	}
}

func (r *EdgeReconciler) updateEdgeStatus(ctx context.Context, edge *operatorv1.KnativeEdge, edgeCluster *edgev1.EdgeCluster, deployment *appsv1.Deployment) error {
	edge.Status = operatorv1.KnativeEdgeStatus{
		Zone:                          edgeCluster.Spec.Zone,
		Region:                        edgeCluster.Spec.Region,
		Environments:                  edgeCluster.Spec.Environments,
		DeploymentObservedGeneration:  deployment.Generation,
		EdgeObservedGeneration:        edge.Generation,
		EdgeClusterObservedGeneration: edgeCluster.Generation,
		Conditions:                    make([]metav1.Condition, 0),
	}

	return r.Status().Update(ctx, edge)
}

func (r *EdgeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	var err error

	r.Log = mgr.GetLogger().WithName("operator-knativeedge")
	r.SystemCluster, err = cluster.New(mgr.GetConfig(), func(o *cluster.Options) {
		o.Namespace = controllers.SystemNamespace
	})

	if err != nil {
		return fmt.Errorf("could not create knative-edge-system cluster/cache: %w", err)
	}

	if err := mgr.Add(r.SystemCluster); err != nil {
		return fmt.Errorf("could not start knative-edge-system cluster/cache: %w", err)
	}

	r.mgr = mgr
	r.remoteClusters = make(map[string]clusterWithExtras)

	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1.KnativeEdge{}).
		Watches(
			source.NewKindWithCache(&appsv1.Deployment{}, r.SystemCluster.GetCache()),
			&handler.EnqueueRequestForOwner{OwnerType: &operatorv1.KnativeEdge{}, IsController: true},
		).
		Watches(
			source.NewKindWithCache(&corev1.Secret{}, r.SystemCluster.GetCache()),
			&handler.EnqueueRequestForOwner{OwnerType: &operatorv1.KnativeEdge{}, IsController: true},
		).
		Complete(r)
}
