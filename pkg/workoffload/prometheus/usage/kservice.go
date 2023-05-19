package usage

import (
	"math"
	"strconv"
	"time"

	"edge.jevv.dev/pkg/controllers"
	prometheus "edge.jevv.dev/pkg/workoffload/prometheus/client"
	"edge.jevv.dev/pkg/workoffload/strategy"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

type KServiceUsage struct {
	Name      string
	Namespace string

	Pods []*PodUsage

	// cpu used by all pods, relative to all pods
	Cpu UsageMetric
	// memory used by all pods, relative to all pods
	Memory UsageMetric

	CpuPressure            Pressure
	MemoryPressure         Pressure
	RequestLatencyPressure Pressure
}

func (s *KServiceUsage) UpdateWithPercentileRatio(data []prometheus.PrometheusMatrixDataValue) {
	debug := log.Log.WithName("usage.kservice").V(controllers.DebugLevel)

	timestampCutoff := float64(time.Now().Add(-3 * strategy.EvaluationPeriodInSeconds * time.Second).Unix())

	allValues := make([]float32, 0)

	var halvesRatio float32 = 0
	var firstHalf, secondHalf float32 = 0, 0
	var firstHalfCount, secondHalfCount = 0, 0
	var firstHalfAvg, secondHalfAvg float32 = 0, 0

	var average, variance float32 = 0, 0
	var varianceRatio float32 = 0

	for _, dataPoint := range data {
		value64, err := strconv.ParseFloat(dataPoint.Value, 32)

		if err != nil || math.IsNaN(value64) {
			continue
		}

		value := float32(value64)
		allValues = append(allValues, value)

		if dataPoint.Timestamp > timestampCutoff {
			secondHalf += value
			secondHalfCount++
		} else {
			firstHalf += value
			firstHalfCount++
		}
	}

	if firstHalfCount > 0 {
		firstHalfAvg = firstHalf / float32(firstHalfCount)
	}

	if secondHalfCount > 0 {
		secondHalfAvg = secondHalf / float32(secondHalfCount)
	}

	if len(allValues) > 0 {
		average = (firstHalf + secondHalf) / float32(len(allValues))
	}

	if firstHalfAvg != 0 {
		halvesRatio = secondHalfAvg / firstHalfAvg
	}

	for _, value := range allValues {
		dev := value - average
		variance += dev * dev
	}

	if len(allValues) > 0 {
		variance /= float32(len(allValues))
	}

	debug.Info("debug service metrics", "values", len(allValues), "firstHalfAvg", firstHalfAvg, "secondHalfAvg", secondHalfAvg, "halvesRatio", halvesRatio, "average", average, "variance", variance)

	if average > 0 {
		varianceRatio = variance / average
	}

	if halvesRatio > 1.5 || varianceRatio > 0.5 {
		s.RequestLatencyPressure = HighPressure
	}

	debug.Info("debug service metrics", "condition", halvesRatio > 1.5 || varianceRatio > 0.5, "halvesRatio", halvesRatio, "varianceRatio", varianceRatio)
	debug.Info("debug service", "service", s.RequestLatencyPressure)
}

func (c *ClusterUsage) UpdateKServiceMetrics() {

}

func (c *ClusterUsage) AddKService(name, namespace string) *KServiceUsage {
	namespacedName := types.NamespacedName{Name: name, Namespace: namespace}
	usage, exists := c.Services[namespacedName.String()]

	if exists {
		return usage
	}

	usage = &KServiceUsage{
		Name:                   name,
		Namespace:              namespace,
		Pods:                   make([]*PodUsage, 0),
		CpuPressure:            LowPressure,
		MemoryPressure:         LowPressure,
		RequestLatencyPressure: LowPressure,
	}

	c.Services[namespacedName.String()] = usage

	return usage
}

func (c *ClusterUsage) FinalizeKServiceMetrics() {
	for _, service := range c.Services {
		var cpuUsage int64 = 0
		var memoryUsage int64 = 0

		for _, pod := range service.Pods {
			cpuUsage += pod.Cpu.Usage
			memoryUsage += pod.Memory.Usage
		}

		service.Cpu.Usage = cpuUsage
		service.Cpu.Capacity = c.Cpu.Capacity

		service.Memory.Usage = memoryUsage
		service.Memory.Capacity = c.Memory.Capacity

		if service.Cpu.Capacity > 0 {
			service.Cpu.Percentage = float32(service.Cpu.Usage) / float32(service.Cpu.Capacity) * 100.0
		}

		if service.Memory.Capacity > 0 {
			service.Memory.Percentage = float32(service.Memory.Usage) / float32(service.Memory.Capacity) * 100.0
		}
	}
}
