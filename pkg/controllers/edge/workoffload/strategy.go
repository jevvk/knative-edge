package workoffload

import (
	"context"

	"k8s.io/apimachinery/pkg/types"

	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

type workOffloadResult int

func (r workOffloadResult) String() string {
	switch r {
	case IncreaseTraffic:
		return "IncreaseTraffic"
	case DecreaseTraffic:
		return "DecreaseTraffic"
	case PreserveTraffic:
		return "PreserveTraffic"
	default:
		return "unknown"
	}
}

const (
	IncreaseTraffic = iota
	DecreaseTraffic
	PreserveTraffic
)

type WorkOffloadServiceResult struct {
	Name    types.NamespacedName
	Service *servingv1.Service
	Result  workOffloadResult
}

type WorkOffloadStrategy interface {
	Execute(ctx context.Context) error
	GetResults(services []servingv1.Service) []WorkOffloadServiceResult
}
