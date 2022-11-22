package prometheus

import (
	"encoding/json"
	"errors"
	"fmt"
)

const (
	PrometheusResultTypeMatrix = "matrix"
	PrometheusResultTypeVector = "vector"
	PrometheusResultTypeScalar = "scalar"
	PrometheusResultTypeString = "string"
)

type PrometheusResponse struct {
	Status    string
	Data      *PrometheusData
	ErrorType *string
	Error     *string
	Warnings  *[]string
}

type prometheusResponse struct {
	Status    string          `json:"status"`
	Data      *prometheusData `json:"data,omitempty"`
	ErrorType *string         `json:"errorType,omitempty"`
	Error     *string         `json:"error,omitempty"`
	Warnings  *[]string       `json:"warnings,omitempty"`
}

type PrometheusData struct {
	ResultType string
	Result     PrometheusResult
}

type prometheusData struct {
	ResultType string          `json:"resultType"`
	Result     json.RawMessage `json:"result"`
}

type PrometheusResult interface{}

type PrometheusMatrixResult struct {
	PrometheusResult

	Data []PrometheusMatrixData
}

type PrometheusMatrixData struct {
	Metric map[string]string
	Data   []PrometheusMatrixDataValue
}

type prometheusMatrixData struct {
	Metric map[string]string `json:"metric"`
	Data   [][]interface{}   `json:"values"`
}

type PrometheusMatrixDataValue struct {
	Timestamp float64
	Value     string
}

var ErrPrometheusMatrixDataValueType = errors.New("prometheus result data of type matrix has unexpected type")

func (r *PrometheusResponse) UnmarshalJSON(b []byte) error {
	response := prometheusResponse{}

	if err := json.Unmarshal(b, &response); err != nil {
		return err
	}

	if response.Data != nil {
		switch response.Data.ResultType {
		case PrometheusResultTypeMatrix:
			var result []prometheusMatrixData

			if err := json.Unmarshal(response.Data.Result, &result); err != nil {
				return err
			}

			marshaledData := make([]PrometheusMatrixData, 0, len(result))

			for _, singleResult := range result {
				var data = make([]PrometheusMatrixDataValue, 0, len(singleResult.Data))

				for _, value := range singleResult.Data {
					timestamp, ok := value[0].(float64)

					if !ok {
						return fmt.Errorf("%w: timestamp is not float64", ErrPrometheusMatrixDataValueType)
					}

					value, ok := value[1].(string)

					if !ok {
						return fmt.Errorf("%w: value is not string", ErrPrometheusMatrixDataValueType)
					}

					data = append(data, PrometheusMatrixDataValue{
						Timestamp: timestamp,
						Value:     value,
					})
				}

				marshaledData = append(marshaledData, PrometheusMatrixData{
					Metric: singleResult.Metric,
					Data:   data,
				})
			}

			r.Data = &PrometheusData{
				ResultType: response.Data.ResultType,
				Result: &PrometheusMatrixResult{
					Data: marshaledData,
				},
			}
		default:
			return fmt.Errorf("unsupported prometheus result type: %s", response.Data.ResultType)
		}
	}

	r.Status = response.Status
	r.Error = response.Error
	r.ErrorType = response.ErrorType
	r.Warnings = response.Warnings

	return nil
}
