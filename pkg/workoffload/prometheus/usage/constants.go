package usage

const (
	LatencyRatioTargetAnnotation    = "strategy.edge.jevv.dev/latency-ratio-target"
	LatencyRatioSoftLimitAnnotation = "strategy.edge.jevv.dev/latency-ratio-soft-limit"
	LatencyRatioHardLimitAnnotation = "strategy.edge.jevv.dev/latency-ratio-hard-limit"
	LatencyRatioDecayAnnotation     = "strategy.edge.jevv.dev/latency-ratio-decay"

	LatencyRatioTargetAnnotationDefaultValue    = 1.0
	LatencyRatioSoftLimitAnnotationDefaultValue = 1.75
	LatencyRatioHardLimitAnnotationDefaultValue = 4.0
	LatencyRatioDecayAnnotationDefaultValue     = 0.75
)
