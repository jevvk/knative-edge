package reflector

import (
	"context"
	"errors"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"
	klog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"

	ws "edge.knative.dev/pkg/reflector/websockets"
)

type Reflector struct {
	manager.Runnable
	inject.Client
	inject.Stoppable

	SyncPeriod time.Duration
	edgeClient *ws.EdgeClient

	client client.Client
	stop   <-chan struct{}
	err    chan error
}

var log = klog.Log.WithName("reflector")

func (r *Reflector) InjectClient(cl client.Client) {
	r.client = cl
}

func (r *Reflector) InjectStopChannel(stop <-chan struct{}) error {
	r.stop = stop
	return nil
}

func (r *Reflector) loop(ctx context.Context) {
	// TODO: remake edge connection on config change
	// ctx, cancel = context.WithCancel(ctx)

	for {
		go func() {
			r.err <- r.edgeClient.Start(ctx)
		}()

		select {
		case <-r.err:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (r *Reflector) waitForStop() {
	if r.stop == nil {
		return
	}

	<-r.stop

	if r.edgeClient != nil {
		r.edgeClient.Stop()
	}

	r.err <- nil
}

func (r *Reflector) Start(ctx context.Context) error {
	if r.client == nil {
		return errors.New("no kube client found")
	}

	if r.stop == nil {
		return errors.New("no stop channel found")
	}

	r.err = make(chan error)

	go r.waitForStop()
	go r.loop(ctx)

	select {
	case err := <-r.err:
		return err
	case <-ctx.Done():
		return nil
	}
}
