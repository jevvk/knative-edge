package apiproxy

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"edge.knative.dev/pkg/cloud/apiproxy/authentication"
	"edge.knative.dev/pkg/cloud/apiproxy/websockets"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func New(ctx context.Context, addr string, clientManager *websockets.ClientManager) *http.Server {
	authenticator, err := authentication.NewFromLocalFiles()

	if err != nil {
		panic(fmt.Errorf("could not create authenticator: %s", err))
	}

	websocketHandler := websockets.NewHandler(clientManager)
	log.Printf("Created websocket handler.")

	authenticatorHandler := authenticator.NewHandler(websocketHandler)
	log.Printf("Created authenticator.")

	server := &http.Server{
		Addr:    addr,
		Handler: h2c.NewHandler(authenticatorHandler, &http2.Server{}),
	}

	return server
}
