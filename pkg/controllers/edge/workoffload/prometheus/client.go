package prometheus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/go-logr/logr"

	"edge.jevv.dev/pkg/controllers"
)

var ErrPrometheusBadRequest = errors.New("prometheus rejected the request")
var ErrPrometheusInvalidQuery = errors.New("prometheus rejected the query")
var ErrPrometheusServiceUnavailable = errors.New("prometheus is timed out")
var ErrPrometheusUnsuccessfulQuery = errors.New("prometheus query has failed")
var ErrPrometheusInvalidResultType = errors.New("prometheus query has an invalid result type")

type PrometheusClient struct {
	Log logr.Logger

	Url url.URL
}

type prometheusQuery struct {
	// raw prometheus query; only queries returning a series (i.e.return type is matrix)
	Query string
	// resolution of the data (e.g. 1m, 5m, 1h)
	Step string
	// how many seconds in the past to look back
	Lookback int
	// max attempts to query prometheus
	Attempts *int
}

func (p *PrometheusClient) Query(ctx context.Context, q prometheusQuery) (*PrometheusMatrixResult, error) {
	url := p.Url
	url.Path = "/api/v1/query_range"

	endTimestamp := time.Now().Unix()
	startTimestamp := endTimestamp - int64(q.Lookback)

	query := url.Query()

	query.Add("query", q.Query)
	query.Add("start", fmt.Sprint(startTimestamp))
	query.Add("end", fmt.Sprint(endTimestamp))
	query.Add("step", q.Step)

	url.RawQuery = query.Encode()

	p.Log.V(controllers.DebugLevel).Info("debug prometheus query", "path", "/api/v1/query_range", "query", q.Query, "start", startTimestamp, "end", endTimestamp, "step", q.Step)

	req, err := http.NewRequestWithContext(ctx, "GET", url.String(), nil)

	if err != nil {
		return nil, fmt.Errorf("couldn't create prometheus api request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return nil, fmt.Errorf("couldn't query prometheus: %w", err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	jsonStr := string(body)

	// missing parameters
	if resp.StatusCode == 400 {
		p.Log.Error(err, "Prometheus request is invalid.", "json", jsonStr)
		// TODO: remove panic later on
		panic(ErrPrometheusBadRequest)
	}

	// invalid query
	if resp.StatusCode == 422 {
		p.Log.Error(err, "Prometheus query is invalid.", "json", jsonStr)
		// TODO: remove panic later on
		panic(ErrPrometheusInvalidQuery)
	}

	// timeout, try again
	if resp.StatusCode == 503 {
		err := ErrPrometheusServiceUnavailable
		p.Log.V(controllers.DebugLevel).Error(err, "Prometheus returned service unavailable.", "json", jsonStr)
		return nil, err
	}

	var pResponse PrometheusResponse
	if err := json.Unmarshal(body, &pResponse); err != nil {
		p.Log.V(controllers.DebugLevel).Error(err, "Prometheus response cannot be parsed.", "json", jsonStr)
		return nil, fmt.Errorf("couldn't parse prometheus response: %w", err)
	}

	if pResponse.Status != "success" {
		err := fmt.Errorf("%w: status is %s", ErrPrometheusUnsuccessfulQuery, pResponse.Status)
		p.Log.Error(err, "Prometheus query returned an error.", "json", jsonStr)
		return nil, err
	}

	if pResponse.Data == nil {
		err := fmt.Errorf("%w: no result", ErrPrometheusUnsuccessfulQuery)
		p.Log.Error(err, "Prometheus query returned no result.", "json", jsonStr)
		return nil, err
	}

	if pResponse.Data.ResultType != PrometheusResultTypeMatrix {
		err := fmt.Errorf("%w: result type is %s", ErrPrometheusInvalidResultType, pResponse.Data.ResultType)
		p.Log.Error(err, "Prometheus query returned invalid result type.", "json", jsonStr)
		return nil, err
	}

	if result, ok := pResponse.Data.Result.(*PrometheusMatrixResult); ok {
		return result, nil
	} else {
		err := fmt.Errorf("cannot cast prometheus result")
		p.Log.Error(err, "Prometheus response cannot be casted.")
		return nil, err
	}
}

func (t *PrometheusClient) QueryWithRetry(ctx context.Context, q prometheusQuery) (*PrometheusMatrixResult, error) {
	var attempts = 1

	if q.Attempts != nil && *q.Attempts > 0 {
		attempts = *q.Attempts
	}

	var result *PrometheusMatrixResult
	var err error

	t.Log.V(controllers.DebugLevel).Info("debug query prometheus with retry", "attempts", attempts)

	for i := 0; i < attempts; i++ {
		result, err = t.Query(ctx, q)

		if err == nil {
			return result, nil
		}

		if errors.Is(err, ErrPrometheusServiceUnavailable) {
			// give it some time to recover
			time.Sleep(time.Second / 5)
			continue
		}

		break
	}

	return nil, err
}
