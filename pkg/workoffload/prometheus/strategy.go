package prometheus

import (
	"context"
	"fmt"
	"net/url"

	"github.com/go-logr/logr"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"edge.jevv.dev/pkg/controllers"
	prometheus "edge.jevv.dev/pkg/workoffload/prometheus/client"
	"edge.jevv.dev/pkg/workoffload/prometheus/usage"
	"edge.jevv.dev/pkg/workoffload/strategy"
)

//+kubebuilder:rbac:groups="",resources=nodes,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
//+kubebuilder:rbac:groups=metrics.k8s.io,resources=nodes,verbs=get;list;watch
//+kubebuilder:rbac:groups=metrics.k8s.io,resources=pods,verbs=get;list;watch

type PrometheusStrategy struct {
	client.Client

	Log logr.Logger

	PrometheusClient prometheus.PrometheusClient
	MetricsClient    *metricsv.Clientset

	cluster *usage.ClusterUsage
}

func NewStrategy(log logr.Logger, prometheusUrl string, client client.Client, metricsClient *metricsv.Clientset) (*PrometheusStrategy, error) {
	var prometheusURL *url.URL
	var err error

	if prometheusUrl == "" {
		return nil, fmt.Errorf("prometheus url is empty")
	}

	// TODO: add basic auth
	// prometheusUser := os.Getenv(controllers.PrometheusUserEnv)
	// prometheusPassword := os.Getenv(controllers.PrometheusPasswordEnv)

	if prometheusURL, err = url.Parse(prometheusUrl); err != nil {
		return nil, fmt.Errorf("prometheus url is invalid: %w", err)
	}

	return &PrometheusStrategy{
		Client:        client,
		Log:           log.WithName("prometheus"),
		MetricsClient: metricsClient,
		PrometheusClient: prometheus.PrometheusClient{
			Log: log.WithName("prometheus"),
			Url: *prometheusURL,
		},
	}, nil
}

func (s *PrometheusStrategy) Execute(ctx context.Context) error {
	debug := s.Log.V(controllers.DebugLevel)

	var err error
	cluster := usage.NewClusterUsage()

	if err = s.updateNodesUsage(ctx, cluster); err != nil {
		return err
	}

	if err := s.updatePodsUsage(ctx, cluster); err != nil {
		return err
	}

	if err := s.updateKServiceUsage(ctx, cluster); err != nil {
		return err
	}

	cluster.FinalizeClusterMetrics()
	cluster.UpdateFromPreviousState(s.cluster)
	s.cluster = cluster

	// debug.Info("debug cluster", "cluster", cluster)
	debug.Info("debug cluster pressure", "nodes", len(cluster.Nodes), "cpu", cluster.Cpu, "mem", cluster.Memory)
	debug.Info("debug cluster pressure", "nodes", len(cluster.Nodes), "cpu", cluster.CpuPressure, "mem", cluster.MemoryPressure)

	return nil
}

func (s *PrometheusStrategy) updateNodesUsage(ctx context.Context, cluster *usage.ClusterUsage) error {
	debug := s.Log.V(controllers.DebugLevel + 1)

	var nodeList corev1.NodeList

	if err := s.Client.List(ctx, &nodeList); err != nil {
		return fmt.Errorf("cannot list nodes metrics: %w", err)
	}

	nodeMetricsList, err := s.MetricsClient.MetricsV1beta1().NodeMetricses().List(ctx, metav1.ListOptions{})

	if err != nil {
		return fmt.Errorf("cannot list node metrics: %w", err)
	}

	defer cluster.FinalizeNodeMetrics()

	for _, node := range nodeList.Items {
		debug.Info("debug node", "node", node.Name, "cpu", node.Status.Capacity.Cpu(), "mem", node.Status.Capacity.Memory())
		cluster.AddNode(node)
	}

	for _, nodeMetrics := range nodeMetricsList.Items {
		debug.Info("debug metrics", "node", nodeMetrics.Name, "cpu", nodeMetrics.Usage.Cpu(), "mem", nodeMetrics.Usage.Memory())
		cluster.UpdateNodeMetrics(nodeMetrics)
	}

	return nil
}

func (s *PrometheusStrategy) updatePodsUsage(ctx context.Context, cluster *usage.ClusterUsage) error {
	debug := s.Log.V(controllers.DebugLevel + 1)

	var namespaceList corev1.NamespaceList

	if err := s.Client.List(ctx, &namespaceList); err != nil {
		return fmt.Errorf("cannot list namespaces: %w", err)
	}

	defer cluster.FinalizePodMetrics()

	for _, namespace := range namespaceList.Items {
		debug.Info("debug update pods", "namespace", namespace.Name)

		err := s.updatePodsUsageInNamespace(ctx, cluster, namespace.Name)

		if err != nil {
			return err
		}
	}

	return nil
}

func (s *PrometheusStrategy) updatePodsUsageInNamespace(ctx context.Context, cluster *usage.ClusterUsage, namespace string) error {
	debug := s.Log.V(controllers.DebugLevel + 1)

	podMetricsList, err := s.MetricsClient.MetricsV1beta1().PodMetricses(namespace).List(ctx, metav1.ListOptions{})

	if err != nil {
		return fmt.Errorf("cannot list pod metrics: %w", err)
	}

	for _, podMetrics := range podMetricsList.Items {
		var pod corev1.Pod
		podName := types.NamespacedName{Name: podMetrics.Name, Namespace: podMetrics.Namespace}

		debug.Info("debug update pods", "pod", podName.String())

		if err := s.Client.Get(ctx, podName, &pod); err != nil {
			// not in the cache?
			if apierrors.IsNotFound(err) {
				debug.Info("debug update pods (not found)", "pod", podName.String())
				continue
			}

			return fmt.Errorf("cannot retrieve pod %s: %w", podName.String(), err)
		}

		debug.Info("debug metrics", "namespace", namespace, "pod", podMetrics.Name, "containers", podMetrics.Containers)

		cluster.AddPod(pod)
		cluster.UpdatePodMetrics(podMetrics)
	}

	return nil
}

func (s *PrometheusStrategy) updateKServiceUsage(ctx context.Context, cluster *usage.ClusterUsage) error {
	debug := s.Log.V(controllers.DebugLevel)

	var err error
	var result *prometheus.PrometheusMatrixResult

	defer cluster.FinalizeKServiceMetrics()

	if result, err = s.PrometheusClient.QueryWithRetry(ctx, ServiceRequestLatencyPercentileRatio); err != nil {
		return err
	}

	for _, data := range result.Data {
		namespace := data.Metric["namespace_name"]
		serviceName := data.Metric["service_name"]

		if serviceName == "" {
			debug.Info("debug empty service name", "metric", data.Metric, "size", len(data.Data))
			continue
		}

		service := cluster.AddKService(serviceName, namespace)

		service.UpdateWithPercentileRatio(data.Data)
		debug.Info("debug revision", "namespace", namespace, "service", serviceName, "metric", data.Metric, "data", data.Data)
	}

	return nil
}

func (s *PrometheusStrategy) GetResults(services []servingv1.Service) []strategy.WorkOffloadServiceResult {
	debug := s.Log.V(controllers.DebugLevel)

	// now := time.Now()
	ret := make([]strategy.WorkOffloadServiceResult, 0, len(services))

	for _, service := range services {
		var action strategy.TrafficAction = strategy.PreserveTraffic
		var desiredTraffic int64 = -1
		serviceName := types.NamespacedName{Name: service.Name, Namespace: service.Namespace}

		serviceUsage, exists := s.cluster.Services[serviceName.String()]

		// don't update traffic if service not in usage
		if exists {
			// FIXME: this is ugly
			serviceUsage.UpdateWithKService(service)
			serviceUsage.FinalizeKServiceMetrics()

			action = strategy.SetTraffic

			if serviceUsage.RequestLatency < serviceUsage.RequestLatencySoftLimit {
				desiredTraffic = 0
			} else if serviceUsage.RequestLatency >= serviceUsage.RequestLatencyHardLimit {
				desiredTraffic = 100
			} else {
				limitDiff := serviceUsage.RequestLatencyHardLimit - serviceUsage.RequestLatencySoftLimit
				overSoftLimit := (serviceUsage.RequestLatency - serviceUsage.RequestLatencySoftLimit) / limitDiff

				desiredTraffic = int64(100.0 * overSoftLimit)
			}
		}

		ret = append(ret, strategy.WorkOffloadServiceResult{
			Name:           serviceName,
			Service:        &service,
			Action:         action,
			DesiredTraffic: desiredTraffic,
		})

		if serviceUsage != nil {
			debug.Info("debug results cluster", "service", serviceName, "clusterCpuPressure", s.cluster.CpuPressure, "clusterMemoryPressure", s.cluster.MemoryPressure)
			debug.Info("debug results service", "service", serviceName, "serviceUsage", serviceUsage)
			debug.Info("debug results service", "service", serviceName, "annotations", service.Annotations)
		}

	}

	return ret
}
