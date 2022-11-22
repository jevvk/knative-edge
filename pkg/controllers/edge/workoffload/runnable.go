package workoffload

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"edge.jevv.dev/pkg/controllers"
	"edge.jevv.dev/pkg/controllers/edge/store"
	"github.com/go-logr/logr"
	klabels "k8s.io/apimachinery/pkg/labels"

	"sigs.k8s.io/controller-runtime/pkg/client"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

type KServiceListOptions struct {
	Selector        klabels.Selector
	ResourceVersion string
	Continue        string
}

func (ko *KServiceListOptions) ApplyToList(o *client.ListOptions) {
	o.LabelSelector = ko.Selector
	o.Continue = ko.Continue

	o.Raw = &metav1.ListOptions{
		ResourceVersion: ko.ResourceVersion,
	}
}

type EdgeWorkOffload struct {
	client.Client

	Log   logr.Logger
	Envs  []string
	Store *store.Store

	strategy WorkOffloadStrategy
}

func (t *EdgeWorkOffload) NeedLeaderElection() bool {
	// doesn't matter either way, this only controls the cleanup
	return true
}

func createListOptions(envs []string, services *servingv1.ServiceList) (*KServiceListOptions, error) {
	var labelSelector metav1.LabelSelector

	if len(envs) == 0 {
		labelSelector = metav1.LabelSelector{
			MatchExpressions: []metav1.LabelSelectorRequirement{
				{
					Key:      controllers.EnvironmentLabel,
					Operator: metav1.LabelSelectorOpExists,
				},
				{
					Key:      controllers.EdgeOffloadLabel,
					Operator: metav1.LabelSelectorOpIn,
					Values:   []string{"true"},
				},
			},
		}
	} else {
		labelSelector = metav1.LabelSelector{
			MatchExpressions: []metav1.LabelSelectorRequirement{
				{
					Key:      controllers.EnvironmentLabel,
					Operator: metav1.LabelSelectorOpIn,
					Values:   envs,
				},
				{
					Key:      controllers.EdgeOffloadLabel,
					Operator: metav1.LabelSelectorOpIn,
					Values:   []string{"true"},
				},
			},
		}
	}

	selector, err := metav1.LabelSelectorAsSelector(&labelSelector)

	if err != nil {
		return nil, fmt.Errorf("couldn't create label selector: %w", err)
	}

	if services == nil {
		return &KServiceListOptions{Selector: selector}, nil
	}

	return &KServiceListOptions{
		Selector:        selector,
		ResourceVersion: services.ResourceVersion,
		Continue:        services.Continue,
	}, nil
}

func (t *EdgeWorkOffload) listServices(ctx context.Context) ([]servingv1.Service, error) {
	var services []servingv1.Service

	errChan := make(chan error)
	doneChan := make(chan interface{})
	listChan := make(chan *servingv1.ServiceList)

	ctx, cancel := context.WithCancel(ctx)

	// producer: lists services, deals with continuation tokens
	go func() {
		var lastServices *servingv1.ServiceList

		var err error
		var services servingv1.ServiceList
		var options *KServiceListOptions

		// cancel when producer is finished
		defer cancel()

		for {
			if options, err = createListOptions(t.Envs, lastServices); err != nil {
				t.Log.Error(err, "Couldn't create list options for Knative Services.")
				errChan <- err
				return
			}

			if err = t.List(ctx, &services, options); err != nil {
				// maybe it's a temporary issue
				t.Log.Error(err, "Couldn't list the Knative Services in the cluster.")
				errChan <- err
				return
			}

			// push to consumer
			listChan <- &services

			// stop when there's no continuation token
			if services.Continue == "" {
				return
			}

			lastServices = &services
		}
	}()

	// consumer: consumes services, adds them to the final list
	go func() {
		sList := make([]*servingv1.ServiceList, 0)

		// at the end of the function, coppy to final buffer
		defer func() {
			var length int

			for _, s := range sList {
				length += len(s.Items)
			}

			services = make([]servingv1.Service, length)

			var i int
			for _, s := range sList {
				i += copy(services[i:], s.Items)
			}

			doneChan <- true
		}()

		// just loop until we are finished
		for {
			select {
			case list := <-listChan:
				sList = append(sList, list)
			case <-ctx.Done():
				return
			}
		}
	}()

	// wait for work to finish
	<-doneChan

	select {
	case err := <-errChan:
		return services, err

	// TODO: should I return ctx.Err ?
	case <-ctx.Done():
		return services, nil
	}
}

func (t *EdgeWorkOffload) run(ctx context.Context, services []servingv1.Service) error {
	if len(services) == 0 {
		t.Log.V(controllers.DebugLevel).Info("no services found, will skip this run")
		return nil
	}

	if err := t.strategy.Execute(ctx); err != nil {
		return fmt.Errorf("could not execute strategy: %w", err)
	}

	for _, result := range t.strategy.GetResults(services) {
		// TODO: check if service has traffic enabled (might not matter)

		traffic, exists := t.Store.Get(result.Name.String())

		// update traffic in store if it doesn't exist
		if !exists {
			// default is 0
			traffic = 0

			// parse it from annotations
			if result.Service != nil && result.Service.Annotations != nil {
				previousTrafficStr := result.Service.Annotations[controllers.EdgeProxyTrafficAnnotation]
				previousTraffic, err := strconv.ParseInt(previousTrafficStr, 10, 64)

				if err != nil {
					traffic = previousTraffic
				}
			}

			t.Store.Set(result.Name.String(), traffic)
		}

		switch result.Result {
		case IncreaseTraffic:
			traffic += 10

			if traffic > 100 {
				traffic = 100
			}

			t.Store.Set(result.Name.String(), traffic)
		case DecreaseTraffic:
			traffic -= 10

			if traffic < 0 {
				traffic = 0
			}

			t.Store.Set(result.Name.String(), traffic)
		}

		t.Log.V(controllers.DebugLevel).Info("debug results", "name", result.Name, "result", result.Result, "traffic", traffic)
	}

	return nil
}

func (t *EdgeWorkOffload) Start(ctx context.Context) error {
	var err error

	if t.strategy, err = NewPrometheusStrategy(t.Log.WithName("strategy")); err != nil {
		return err
	}

	if t.Store == nil {
		return fmt.Errorf("no traffic split store provided")
	}

	go func() {
		for {
			timeoutCtx, timeoutCancel := context.WithTimeout(context.Background(), time.Second*15)
			// timeoutCtx, timeoutCancel := context.WithTimeout(context.Background(), time.Minute*5)

			select {
			case <-timeoutCtx.Done():
				timeoutCancel() // not sure if necessary
			case <-ctx.Done():
				timeoutCancel()
				return
			}

			services, err := t.listServices(ctx)

			if err != nil {
				t.Log.Error(err, "Encountered an error while listing Knative services. Will skip this run.")
				continue
			}

			t.run(ctx, services)
		}
	}()

	return nil
}
