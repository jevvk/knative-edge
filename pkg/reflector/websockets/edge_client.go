package websockets

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	klog "sigs.k8s.io/controller-runtime/pkg/log"

	cloudauthentication "edge.knative.dev/pkg/apiproxy/authentication"
	edgeevent "edge.knative.dev/pkg/apiproxy/event"
	"edge.knative.dev/pkg/reflector/authentication"
)

var log = klog.Log.WithName("reflector").WithName("websockets")

type handlerfn func(ctx context.Context, eventWrapper *edgeevent.Event)

type EdgeClient struct {
	conn     *websocket.Conn
	handlers []handlerfn

	stop chan error
}

func New(ctx context.Context, url string, token string) (*EdgeClient, error) {
	authenticator, err := authentication.New(token)

	if err != nil {
		return nil, err
	}

	dialer := &websocket.Dialer{HandshakeTimeout: 60 * time.Second}
	dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true, VerifyConnection: authenticator.Authorize}

	headers := http.Header{}
	headers.Add(cloudauthentication.AuthHeader, token)

	conn, resp, err := dialer.DialContext(ctx, url, headers)

	if err != nil {
		if resp != nil {
			log.Error(err, fmt.Sprintf("Couldn't connect to remote server. (status code: %d)", resp.StatusCode))
		} else {
			log.Error(err, "Couldn't connect to remote server.")
		}

		return nil, err
	}

	return &EdgeClient{
		conn:     conn,
		handlers: make([]handlerfn, 1),
		stop:     make(chan error),
	}, nil
}

func (ec *EdgeClient) AddEventHandler(handler handlerfn) {
	ec.handlers = append(ec.handlers, handler)
}

func (ec *EdgeClient) Start(ctx context.Context) error {

	<-ctx.Done()
	return nil
}

func (ec *EdgeClient) Stop() {
	if ec.conn != nil {
		ec.conn.Close()
	}

	ec.stop <- nil
}
