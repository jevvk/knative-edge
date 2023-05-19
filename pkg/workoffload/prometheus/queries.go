package prometheus

import (
	"fmt"

	"edge.jevv.dev/pkg/controllers/utils"
	"edge.jevv.dev/pkg/workoffload/prometheus/client"
	"edge.jevv.dev/pkg/workoffload/strategy"
)

func withEdgeProxy(format string) string {
	return fmt.Sprintf(format, utils.EdgeProxySuffix)
}

var (
	ServiceRequestRate = client.PrometheusQuery{
		Query:    withEdgeProxy("sum by(service_name, namespace_name) (rate(activator_request_count{revision_name!~\".+%s-[0-9]+\"}[1m]))"),
		Step:     "1m",
		Lookback: 3 * strategy.EvaluationPeriodInSeconds,
	}

	ServiceRequestLatencySum = client.PrometheusQuery{
		Query:    withEdgeProxy("sum by(service_name, namespace_name) (rate(activator_request_latencies_sum{revision_name!~\".+%s-[0-9]+\"}[1m]))"),
		Step:     "1m",
		Lookback: 3 * strategy.EvaluationPeriodInSeconds,
	}

	ServicePodCount = client.PrometheusQuery{
		Query:    withEdgeProxy("count by(service_name, namespace_name) (revision_app_request_count{revision_name!~\".+%s-[0-9]+\"})"),
		Step:     "1m",
		Lookback: 3 * strategy.EvaluationPeriodInSeconds,
	}

	ServiceRequestLatency95thPercentile = client.PrometheusQuery{
		Query:    withEdgeProxy("histogram_quantile(0.95, sum(rate(activator_request_latencies_bucket{response_code!=\"502\",revision_name!~\".+%s-[0-9]+\"}[1m])) by(le, service_name, namespace_name))"),
		Step:     "1m",
		Lookback: 3 * 2 * strategy.EvaluationPeriodInSeconds,
	}

	ServiceRequestLatency50thPercentile = client.PrometheusQuery{
		Query:    withEdgeProxy("histogram_quantile(0.50, sum(rate(activator_request_latencies_bucket{response_code!=\"502\",revision_name!~\".+%s-[0-9]+\"}[1m])) by(le, service_name, namespace_name))"),
		Step:     "1m",
		Lookback: 3 * 2 * strategy.EvaluationPeriodInSeconds,
	}
)

var (
	ServiceAverageRequestLatency = client.PrometheusQuery{
		Query:    fmt.Sprintf("(%s) / (%s)", ServiceRequestLatencySum.Query, ServiceRequestRate.Query),
		Step:     "1m",
		Lookback: 3 * strategy.EvaluationPeriodInSeconds,
	}

	ServiceRequestLatencyPercentileRatio = client.PrometheusQuery{
		Query:    fmt.Sprintf("(%s) / (%s)", ServiceRequestLatency95thPercentile.Query, ServiceRequestLatency50thPercentile.Query),
		Step:     "1m",
		Lookback: 3 * 2 * strategy.EvaluationPeriodInSeconds,
	}
)
