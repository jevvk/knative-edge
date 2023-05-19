package usage

import (
	"encoding/json"
	"math"
	"strconv"

	"edge.jevv.dev/pkg/controllers"
	prometheus "edge.jevv.dev/pkg/workoffload/prometheus/client"
	"k8s.io/apimachinery/pkg/types"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
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

	CpuPressure    Pressure
	MemoryPressure Pressure

	RequestLatencyPressure  Pressure
	RequestLatency          float32
	RequestLatencySoftLimit float32
	RequestLatencyHardLimit float32

	requestLatencies     []float32
	requestLatencyDecay  float32
	requestLatencyTarget float32
}

func (s *KServiceUsage) MarshalJSON() ([]byte, error) {
	obj := struct {
		Name                    string
		Namespace               string
		Cpu                     UsageMetric
		CpuPressure             Pressure
		Memory                  UsageMetric
		MemoryPressure          Pressure
		RequestLatency          *float32
		RequestLatencies        []float32
		RequestLatencySoftLimit *float32
		RequestLatencyHardLimit *float32
		RequestLatencyDecay     *float32
		RequestLatencyTarget    *float32
	}{
		s.Name, s.Namespace, s.Cpu, s.CpuPressure, s.Memory, s.MemoryPressure,
		&s.RequestLatency, nil,
		&s.RequestLatencySoftLimit, &s.RequestLatencyHardLimit,
		&s.requestLatencyDecay, &s.requestLatencyTarget,
	}

	if s.requestLatencies != nil {
		latencies := make([]float32, 0, len(s.requestLatencies))

		for _, latency := range s.requestLatencies {
			if math.IsInf(float64(latency), -1) {
				latency = math.MaxFloat32 * -1
			} else if math.IsInf(float64(latency), 1) {
				latency = math.MaxFloat32
			}

			if math.IsNaN(float64(latency)) {
				latencies = append(latencies, 85428)
			} else {
				latencies = append(latencies, latency)
			}
		}

		obj.RequestLatencies = latencies
	}

	if math.IsNaN(float64(*obj.RequestLatency)) {
		obj.RequestLatency = nil
	}
	if math.IsNaN(float64(*obj.RequestLatencySoftLimit)) {
		obj.RequestLatencySoftLimit = nil
	}
	if math.IsNaN(float64(*obj.RequestLatencyHardLimit)) {
		obj.RequestLatencyHardLimit = nil
	}
	if math.IsNaN(float64(*obj.RequestLatencyDecay)) {
		obj.RequestLatencyDecay = nil
	}
	if math.IsNaN(float64(*obj.RequestLatencyTarget)) {
		obj.RequestLatencyTarget = nil
	}

	return json.Marshal(obj)
}

func (s *KServiceUsage) UpdateWithKService(service servingv1.Service) {
	annotations := service.Annotations

	if annotations == nil {
		annotations = make(map[string]string)
	}

	s.requestLatencyDecay = LatencyRatioDecayAnnotationDefaultValue
	s.requestLatencyTarget = LatencyRatioTargetAnnotationDefaultValue
	s.RequestLatencySoftLimit = LatencyRatioSoftLimitAnnotationDefaultValue
	s.RequestLatencyHardLimit = LatencyRatioHardLimitAnnotationDefaultValue

	if value, err := strconv.ParseFloat(annotations[LatencyRatioDecayAnnotation], 32); err == nil {
		s.requestLatencyDecay = float32(value)
	}

	if value, err := strconv.ParseFloat(annotations[LatencyRatioTargetAnnotation], 32); err == nil {
		s.requestLatencyTarget = float32(value)
	}

	if value, err := strconv.ParseFloat(annotations[LatencyRatioSoftLimitAnnotation], 32); err == nil {
		s.RequestLatencySoftLimit = float32(value)
	}

	if value, err := strconv.ParseFloat(annotations[LatencyRatioHardLimitAnnotation], 32); err == nil {
		s.RequestLatencyHardLimit = float32(value)
	}
}

func (s *KServiceUsage) UpdateWithPercentileRatio(data []prometheus.PrometheusMatrixDataValue) {
	debug := log.Log.WithName("usage.kservice").V(controllers.DebugLevel)

	// default to NaN
	var lastValue *float32

	s.requestLatencies = make([]float32, 0, len(data))

	// we only get NaNs for missing data after the first value, however
	// we decay starting from last index, so we don't care if we don't
	// have NaNs filled in before the first value

	for _, dataPoint := range data {
		var value float32
		value64, err := strconv.ParseFloat(dataPoint.Value, 32)

		if err != nil || math.IsNaN(value64) || math.IsInf(value64, 0) {
			if lastValue == nil {
				continue
			}

			value = *lastValue
		} else {
			value = float32(value64)
		}

		s.requestLatencies = append(s.requestLatencies, value)
		lastValue = &value
	}

	debug.Info("debug req latency", "namespace", s.Namespace, "service", s.Name, "data", data, "latencies", s.requestLatencies)
}

func (s *KServiceUsage) FinalizeKServiceMetrics() {
	var requestLatency float32 = 0
	var requestLatencyWeights float32 = 0
	var weight float32 = 1

	if s.requestLatencies == nil {
		return
	}

	// use last value to replace NaNs
	var lastLatency *float32

	for i := len(s.requestLatencies) - 1; i >= 0; i-- {
		latency := s.requestLatencies[i]

		if math.IsInf(float64(latency), 0) {
			continue
		}

		if math.IsNaN(float64(latency)) {
			// skip NaNs at the end
			if lastLatency == nil {
				continue
			}

			latency = *lastLatency
		}

		requestLatency += weight * latency
		requestLatencyWeights += weight
		weight *= s.requestLatencyDecay

		lastLatency = &latency
	}

	if requestLatencyWeights > 0 {
		requestLatency /= requestLatencyWeights
	}

	s.RequestLatency = requestLatency
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
		// cpu and memory usage

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
