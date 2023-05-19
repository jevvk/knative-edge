package workoffload

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"

	"sigs.k8s.io/controller-runtime/pkg/client"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	"edge.jevv.dev/pkg/controllers"
	"edge.jevv.dev/pkg/workoffload/prometheus"
	"edge.jevv.dev/pkg/workoffload/store"
	"edge.jevv.dev/pkg/workoffload/strategy"
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

	MetricsClient *metricsv.Clientset

	Log           logr.Logger
	Envs          []string
	Store         *store.Store
	PrometheusUrl string

	strategy strategy.WorkOffloadStrategy
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
	debug := t.Log.V(controllers.DebugLevel)

	if len(services) == 0 {
		debug.Info("no services found, will skip this run")
		return nil
	}

	if err := t.strategy.Execute(ctx); err != nil {
		return fmt.Errorf("could not execute strategy: %w", err)
	}

	// recover what last traffic was set to
	for _, service := range services {
		serviceName := types.NamespacedName{Name: service.Name, Namespace: service.Namespace}
		_, exists := t.Store.Get(serviceName.String())

		if exists {
			continue
		}

		// update traffic in store if it doesn't exist
		// default is 0
		var traffic int64 = 0

		// parse it from annotations
		if service.Annotations != nil {
			previousTrafficStr := service.Annotations[controllers.EdgeProxyTrafficAnnotation]
			previousTraffic, err := strconv.ParseInt(previousTrafficStr, 10, 64)

			if err != nil {
				traffic = previousTraffic
			}
		}

		t.Store.Set(serviceName.String(), traffic)
	}

	for _, result := range t.strategy.GetResults(services) {
		// TODO: check if service has traffic enabled (might not matter)

		traffic, exists := t.Store.Get(result.Name.String())

		if !exists {
			// default is 0
			traffic = 0
		}

		switch result.Action {
		case strategy.PreserveTraffic:
			continue
		case strategy.SetTraffic:
			inertia := strategy.TrafficInertiaDefaultValue
			annotations := result.Service.Annotations

			if annotations == nil {
				annotations = make(map[string]string)
			}

			if inertiaAnnotation := annotations[strategy.TrafficInertiaAnnotation]; inertiaAnnotation != "" {
				if inertiaValue, err := strconv.ParseFloat(inertiaAnnotation, 32); err == nil {
					inertia = float32(inertiaValue)
				}
			}

			traffic = int64(float32(traffic)*inertia + float32(result.DesiredTraffic)*(1-inertia))
		case strategy.IncreaseTraffic:
			traffic += 10
		case strategy.DecreaseTraffic:
			traffic -= 2
		}

		if traffic > 100 {
			traffic = 100
		} else if traffic < 0 {
			traffic = 0
		}

		t.Store.Set(result.Name.String(), traffic)
		debug.Info("debug results", "name", result.Name, "action", result.Action, "traffic", traffic)
	}

	return nil
}

func (t *EdgeWorkOffload) Start(ctx context.Context) error {
	debug := t.Log.V(controllers.DebugLevel)
	log := t.Log.V(controllers.InfoLevel)

	var err error

	log.Info("Starting edge traffic runnable.")

	if t.strategy, err = prometheus.NewStrategy(t.Log, t.PrometheusUrl, t.Client, t.MetricsClient); err != nil {
		return err
	}

	if t.Store == nil {
		return fmt.Errorf("no traffic split store provided")
	}

	go func() {
		for {
			timeoutCtx, timeoutCancel := context.WithTimeout(context.Background(), time.Second*strategy.EvaluationPeriodInSeconds)
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
				debug.Error(err, "Encountered an error while listing Knative services. Will skip this run.")
				continue
			}

			startTime := time.Now()
			err = t.run(ctx, services)
			duration := time.Since(startTime)

			if err != nil {
				debug.Error(err, "Encountered an error while executing edge traffic strategy. Will skip this run.")
				continue
			}

			debug.Info("debug run time", "duration", duration)
		}
	}()

	return nil
}
