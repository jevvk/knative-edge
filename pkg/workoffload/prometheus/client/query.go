package client

type PrometheusQuery struct {
	// raw prometheus query; only queries returning a series (i.e.return type is matrix)
	Query string
	// resolution of the data (e.g. 1m, 5m, 1h)
	Step string
	// how many seconds in the past to look back
	Lookback int
	// max attempts to query prometheus
	Attempts *int
}
