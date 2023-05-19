package usage

import (
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"

	"edge.jevv.dev/pkg/controllers"
)

type PodUsage struct {
	Name      string
	Namespace string
	Node      string

	Cpu     UsageMetric
	Memory  UsageMetric
	Storage UsageMetric
}

func (p *PodUsage) UpdateCpuUsage(q *resource.Quantity) {
	if q == nil {
		return
	}

	p.Cpu.Usage = p.Cpu.Usage + q.MilliValue()
}

func (p *PodUsage) UpdateMemoryUsage(q *resource.Quantity) {
	if q == nil {
		return
	}

	p.Memory.Usage = p.Memory.Usage + q.Value()
}

func (p *PodUsage) UpdateStorageUsage(q *resource.Quantity) {
	if q == nil {
		return
	}

	p.Storage.Usage = p.Storage.Usage + q.Value()
}

func (p *PodUsage) UpdateCpuCapacity(capacity int64) {
	p.Cpu.Capacity = capacity

	if p.Cpu.Capacity > 0 {
		p.Cpu.Percentage = float32(p.Cpu.Usage) / float32(p.Cpu.Capacity) * 100.0
	}
}

func (p *PodUsage) UpdateMemoryCapacity(capacity int64) {
	p.Memory.Capacity = capacity

	if p.Memory.Capacity > 0 {
		p.Memory.Percentage = float32(p.Memory.Usage) / float32(p.Memory.Capacity) * 100.0
	}
}

func (p *PodUsage) UpdateStorageCapacity(capacity int64) {
	p.Storage.Capacity = capacity

	if p.Storage.Capacity > 0 {
		p.Storage.Percentage = float32(p.Storage.Usage) / float32(p.Storage.Capacity) * 100.0
	}
}

func (c *ClusterUsage) AddPod(pod corev1.Pod) *PodUsage {
	namespacedName := types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}
	usage, exists := c.Pods[namespacedName.String()]

	if exists {
		return usage
	}

	usage = &PodUsage{
		Name:      pod.Name,
		Namespace: pod.Namespace,
		Node:      pod.Status.NominatedNodeName,
	}

	if usage.Node == "" {
		usage.Node = pod.Spec.NodeName
	}

	c.Pods[namespacedName.String()] = usage

	if pod.Labels == nil {
		return usage
	}

	serviceName := pod.Labels[controllers.KServiceLabel]

	if serviceName == "" {
		return usage
	}

	service := c.AddKService(serviceName, pod.Namespace)
	service.Pods = append(service.Pods, usage)

	return usage
}

func (c *ClusterUsage) UpdatePodMetrics(metric metricsv1beta1.PodMetrics) {
	pod := c.AddPod(corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: metric.Name, Namespace: metric.Namespace}})

	for _, container := range metric.Containers {
		pod.UpdateCpuUsage(container.Usage.Cpu())
		pod.UpdateMemoryUsage(container.Usage.Memory())
		pod.UpdateStorageUsage(container.Usage.Storage())
	}
}

func (c *ClusterUsage) FinalizePodMetrics() {
	for _, pod := range c.Pods {
		node, exists := c.Nodes[pod.Node]

		if !exists {
			// TODO: log?
			continue
		}

		pod.UpdateCpuCapacity(node.Cpu.Capacity)
		pod.UpdateMemoryCapacity(node.Memory.Capacity)
	}
}
