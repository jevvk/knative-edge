package replicator

import (
	"context"
	"fmt"

	"github.com/gorilla/websocket"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"knative.dev/edge/pkg/apiproxy/authentication"
)

type ClientManager struct {
	kube    kubernetes.Clientset
	clients map[string]Client
}

func NewClientManager(kube kubernetes.Clientset) ClientManager {
	return ClientManager{
		kube:    kube,
		clients: make(map[string]Client),
	}
}

func (cm *ClientManager) WithStore(ctx context.Context, store *authentication.Store) error {
	if store == nil {
		panic(fmt.Errorf("no authentication store provided to client manager"))
	}

	watcher, err := cm.kube.RESTClient().Get().Namespace(authentication.Namespace).Resource("edgeclusters").Watch(ctx)

	if err != nil {
		return fmt.Errorf("couldn't watch for new clusters: %s", err)
	}

	go func() {
		for {
			event := <-watcher.ResultChan()

			if event.Type == watch.Error {
				break
			}

			if event.Type == watch.Added {

			} else if event.Type == watch.Deleted {
				event.Object.GetObjectKind()
			} else {
				// nothing to do
			}
		}
	}()

	return nil
}

func (cm *ClientManager) HandleConnection(conn *websocket.Conn) {
	if conn == nil {
		panic(fmt.Errorf("no websocket connection provided to client manager"))
	}

	for {
		msgType, msg, err := conn.ReadMessage()

		if err != nil {
			fmt.Printf("Enountered an error when reading websocket message: %s", err)
			return
		}

		// Print the message to the console
		fmt.Printf("%s sent: %s\n", conn.RemoteAddr(), string(msg))

		// Write message back to browser
		if err = conn.WriteMessage(msgType, msg); err != nil {
			return
		}
	}
}
