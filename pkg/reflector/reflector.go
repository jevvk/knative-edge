package reflector

import (
	"context"
	"errors"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	klog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"

	ws "edge.jevv.dev/pkg/reflector/websockets"
)

type config struct {
	url   string
	token string
}

type Reflector struct {
	manager.Runnable
	inject.Cache
	inject.Client
	inject.Stoppable

	SyncPeriod time.Duration
	edgeClient *ws.EdgeClient

	client client.Client
	cache  cache.Cache
	stop   <-chan struct{}
	err    chan error

	reload chan *config
	cfg    config
}

var log = klog.Log.WithName("reflector")

func (r *Reflector) InjectClient(cl client.Client) {
	r.client = cl
}

func (r *Reflector) InjectCache(ch cache.Cache) {
	r.cache = ch
}

func (r *Reflector) InjectStopChannel(stop <-chan struct{}) error {
	r.stop = stop
	return nil
}

func (r *Reflector) loop(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for {
		time.Sleep(1 * time.Second)
		if cfg, ok := <-r.reload; ok {
			r.cfg = *cfg
		}

		var err error
		r.edgeClient, err = ws.New(ctx, r.cfg.url, r.cfg.token)

		if err != nil {
			log.Error(err, "Edge client not set up yet.")
		} else {
			r.edgeClient.AddEventHandler(r.handleEdgeEvent)

			go func() {
				r.err <- r.edgeClient.Start(ctx)
			}()
		}

		select {
		case cfg := <-r.reload:
			r.edgeClient.Stop()
			r.cfg = *cfg
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

	if r.cache == nil {
		return errors.New("no object cache found")
	}

	if r.stop == nil {
		return errors.New("no stop channel found")
	}

	r.err = make(chan error)
	r.reload = make(chan *config)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go r.waitForStop()
	go r.watchConfig(ctx)
	go r.loop(ctx)

	select {
	case err := <-r.err:
		return err
	case <-ctx.Done():
		return nil
	}
}
