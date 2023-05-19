package strategy

const (
	EvaluationPeriodInSeconds = 60
	LookbackMultiplier        = 5

	TrafficInertiaAnnotation           = "strategy.edge.jevv.dev/traffic-inertia"
	TrafficInertiaDefaultValue float32 = 0.75
)
