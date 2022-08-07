package refractor

import (
	"context"
	"net/http"
)

type healthz struct {
	err     error
	errChan <-chan error
}

func (h *healthz) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case h.err = <-h.errChan:
			continue
		}
	}
}

func (h *healthz) Probe(_ http.Request) error {
	return h.err
}
