package refractor

import (
	"context"
	"fmt"
	"net/http"

	"edge.jevv.dev/pkg/refractor/authentication"
	"edge.jevv.dev/pkg/refractor/websockets"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func New(ctx context.Context, addr string, clientManager *websockets.ClientManager) *http.Server {
	authenticator, err := authentication.NewFromLocalFiles()
	log := log.FromContext(ctx).WithName("refractor")

	if err != nil {
		panic(fmt.Errorf("could not create authenticator: %s", err))
	}

	websocketHandler := websockets.NewHandler(clientManager)
	log.Info("Created websocket handler.")

	authenticatorHandler := authenticator.NewHandler(websocketHandler)
	log.Info("Created authenticator.")

	server := &http.Server{
		Addr:    addr,
		Handler: h2c.NewHandler(authenticatorHandler, &http2.Server{}),
	}

	return server
}
