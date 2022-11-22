package prometheus

// const PrometheusQuery = "count by(service_name, revision_name) (activator_request_count)"
// const PrometheusQueryResolution = "1m"
// const PrometheusQueryLookback = 5 * 60

const ()

type PrometheusQuery interface{}

type NodesUtilizationQuery struct{}

type ServiceUtilizationQuery struct{}
