package usage

import (
	"k8s.io/apimachinery/pkg/api/resource"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

type NodeUsage struct {
	Name   string
	Cpu    UsageMetric
	Memory UsageMetric

	CpuPressure    Pressure
	MemoryPressure Pressure
}

func (n *NodeUsage) UpdateCpuCapacity(q *resource.Quantity) {
	if q == nil {
		return
	}

	n.Cpu.Capacity = q.MilliValue()

	if n.Cpu.Capacity <= 0 {
		n.Cpu.Percentage = 0
	} else {
		n.Cpu.Percentage = float32(n.Cpu.Usage) / float32(n.Cpu.Capacity) * 100.0
	}
}

func (n *NodeUsage) UpdateCpuUsage(q *resource.Quantity) {
	if q == nil {
		return
	}

	n.Cpu.Usage = q.MilliValue()

	if n.Cpu.Capacity <= 0 {
		n.Cpu.Percentage = 0
	} else {
		n.Cpu.Percentage = float32(n.Cpu.Usage) / float32(n.Cpu.Capacity) * 100.0
	}
}

func (n *NodeUsage) UpdateMemoryCapacity(q *resource.Quantity) {
	if q == nil {
		return
	}

	n.Memory.Capacity = q.Value()

	if n.Memory.Capacity <= 0 {
		n.Memory.Percentage = 0
	} else {
		n.Memory.Percentage = float32(n.Memory.Usage) / float32(n.Memory.Capacity) * 100.0
	}
}

func (n *NodeUsage) UpdateMemoryUsage(q *resource.Quantity) {
	if q == nil {
		return
	}

	n.Memory.Usage = q.Value()

	if n.Memory.Capacity <= 0 {
		n.Memory.Percentage = 0
	} else {
		n.Memory.Percentage = float32(n.Memory.Usage) / float32(n.Memory.Capacity) * 100.0
	}
}

func (c *ClusterUsage) AddNode(node corev1.Node) *NodeUsage {
	usage, exists := c.Nodes[node.Name]

	if exists {
		return usage
	}

	usage = &NodeUsage{Name: node.Name}
	c.Nodes[node.Name] = usage

	usage.UpdateCpuCapacity(node.Status.Capacity.Cpu())
	usage.UpdateMemoryCapacity(node.Status.Capacity.Memory())

	return usage
}

func (c *ClusterUsage) UpdateNodeMetrics(metric metricsv1beta1.NodeMetrics) {
	node := c.AddNode(corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: metric.Name}})

	node.UpdateCpuUsage(metric.Usage.Cpu())
	node.UpdateMemoryUsage(metric.Usage.Memory())
}

func (c *ClusterUsage) FinalizeNodeMetrics() {
	for _, node := range c.Nodes {
		if node.Cpu.Percentage >= CPU_PRESSURE_THRESHOLD {
			node.CpuPressure = HighPressure
		} else {
			node.CpuPressure = LowPressure
		}

		if node.Memory.Percentage >= MEM_PRESSURE_THRESHOLD {
			node.MemoryPressure = HighPressure
		} else {
			node.MemoryPressure = LowPressure
		}
	}
}
