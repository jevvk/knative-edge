package workoffload

import (
	"context"
	"fmt"
	"net/url"
	"os"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	"edge.jevv.dev/pkg/controllers"
	"edge.jevv.dev/pkg/controllers/edge/workoffload/prometheus"
)

type PrometheusStrategy struct {
	WorkOffloadStrategy

	Client prometheus.PrometheusClient
}

func NewPrometheusStrategy(log logr.Logger) (*PrometheusStrategy, error) {
	var prometheusUrl *url.URL
	var err error

	prometheusUrlString := os.Getenv(controllers.PrometheusUrlEnv)

	if prometheusUrlString == "" {
		return nil, fmt.Errorf("prometheus url is empty")
	}

	// TODO: add basic auth
	// prometheusUser := os.Getenv(controllers.PrometheusUserEnv)
	// prometheusPassword := os.Getenv(controllers.PrometheusPasswordEnv)

	if prometheusUrl, err = url.Parse(prometheusUrlString); err != nil {
		return nil, fmt.Errorf("prometheus url is invalid: %w", err)
	}

	return &PrometheusStrategy{
		Client: prometheus.PrometheusClient{
			Log: log.WithName("prometheus"),
			Url: *prometheusUrl,
		},
	}, nil
}

func (s *PrometheusStrategy) Execute(ctx context.Context) error {
	// TODO
	return nil
}

func (s *PrometheusStrategy) GetResults(services []servingv1.Service) []WorkOffloadServiceResult {
	ret := make([]WorkOffloadServiceResult, 0, len(services))

	for _, service := range services {
		// TODO
		ret = append(ret, WorkOffloadServiceResult{
			Name:    types.NamespacedName{Name: service.Name, Namespace: service.Namespace},
			Service: &service,
			Result:  PreserveTraffic,
		})
	}

	return ret
}
