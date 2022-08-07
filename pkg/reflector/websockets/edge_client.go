package websockets

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	klog "sigs.k8s.io/controller-runtime/pkg/log"

	cloudauthentication "edge.jevv.dev/pkg/apiproxy/authentication"
	edgeevent "edge.jevv.dev/pkg/apiproxy/event"
	"edge.jevv.dev/pkg/reflector/authentication"
)

var log = klog.Log.WithName("reflector").WithName("websockets")

type handlerfn func(ctx context.Context, eventWrapper *edgeevent.Event)

type EdgeClient struct {
	conn     *websocket.Conn
	handlers []handlerfn

	stop chan error
}

func New(ctx context.Context, url string, token string) (*EdgeClient, error) {
	if url == "" {
		return nil, errors.New("empty websocket url provided")
	}

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
		handlers: make([]handlerfn, 0),
		stop:     make(chan error),
	}, nil
}

func (ec *EdgeClient) SendEvent(ctx context.Context, event *edgeevent.Event) error {
	if ec.conn == nil {
		return errors.New("no websocket connection found")
	}

	return ec.conn.WriteJSON(event)
}

func (ec *EdgeClient) AddEventHandler(handler handlerfn) {
	ec.handlers = append(ec.handlers, handler)
}

func (ec *EdgeClient) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)

	go func() {
		for {
			var event edgeevent.Event
			err := ec.conn.ReadJSON(&event)

			if err != nil {
				if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, &websocket.CloseError{}) {
					log.Error(err, "Edge client closed unexpectedly.")

					cancel()
					break
				} else if neterr, ok := err.(interface{ Temporary() bool }); ok && neterr.Temporary() {
					continue
				} else {
					log.Error(err, "Edge client received an unknown error.")
					cancel()
					break
				}
			}

			for _, handler := range ec.handlers {
				handler(ctx, &event)
			}
		}
	}()

	<-ctx.Done()
	return nil
}

func (ec *EdgeClient) Stop() {
	if ec.conn != nil {
		ec.conn.Close()
	}

	ec.stop <- nil
}
