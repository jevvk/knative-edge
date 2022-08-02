package apiproxy

import "context"

type healthProvider struct {
	err     *error
	errChan <-chan *error
}

func (h *healthProvider) Start(ctx context.Context) {
	for {
		select {
		case err := <-h.errChan:
			h.err = err
		case <-ctx.Done():
			return
		}
	}
}

func (h healthProvider) Healthz() error {
	if h.err == nil {
		return nil
	}

	return *h.err
}
