package strategy

import (
	"context"

	"k8s.io/apimachinery/pkg/types"

	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

type TrafficAction int

func (r TrafficAction) String() string {
	switch r {
	case IncreaseTraffic:
		return "IncreaseTraffic"
	case DecreaseTraffic:
		return "DecreaseTraffic"
	case PreserveTraffic:
		return "PreserveTraffic"
	case SetTraffic:
		return "SetTraffic"
	default:
		return "unknown"
	}
}

const (
	PreserveTraffic = iota
	IncreaseTraffic
	DecreaseTraffic
	SetTraffic
)

type WorkOffloadServiceResult struct {
	Name    types.NamespacedName
	Service *servingv1.Service

	Action         TrafficAction
	DesiredTraffic int64
}

type WorkOffloadStrategy interface {
	Execute(ctx context.Context) error
	GetResults(services []servingv1.Service) []WorkOffloadServiceResult
}
