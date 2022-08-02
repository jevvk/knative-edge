package apiproxy

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"knative.dev/edge/pkg/apiproxy/authentication"
	cm "knative.dev/edge/pkg/apiproxy/clients"
	"knative.dev/edge/pkg/apiproxy/websockets"
)

func New(ctx context.Context, addr string) (*http.Server, *healthProvider) {
	authenticator, err := authentication.NewFromLocalFiles()

	if err != nil {
		panic(fmt.Errorf("could not create authenticator: %s", err))
	}

	clientManager := cm.New(authenticator)
	log.Printf("Created client manager.")

	websocketHandler := websockets.NewHandler(&clientManager)
	log.Printf("Created websocket handler.")

	authenticatorHandler := authenticator.NewHandler(websocketHandler)
	log.Printf("Created authenticator.")

	errChan := make(chan *error)

	go func() {
		err := clientManager.Listen(ctx, errChan)

		if !errors.Is(err, context.Canceled) {
			log.Printf("Client manager stopped with an error: %s", err)
		}
	}()

	server := &http.Server{
		Addr:    addr,
		Handler: h2c.NewHandler(authenticatorHandler, &http2.Server{}),
	}

	err = fmt.Errorf("not started")

	healthz := &healthProvider{
		err:     &err,
		errChan: errChan,
	}

	go healthz.Start(ctx)

	return server, healthz
}
