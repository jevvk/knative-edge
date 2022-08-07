package websockets

import (
	"fmt"

	"edge.jevv.dev/pkg/refractor/authentication"
	"github.com/gorilla/websocket"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/rest"
)

type ClientManager struct {
	auth       *authentication.Authenticator
	edgeClient *clientset.Clientset
	clients    map[string]*Client
}

func NewManager() *ClientManager {
	config, err := rest.InClusterConfig()

	if err != nil {
		panic(fmt.Errorf("couldn't retrieve kubernetes client: %s", err))
	}

	auth, err := authentication.NewFromLocalFiles()

	if err != nil {
		panic(fmt.Errorf("couldn't create authenticator: %s", err))
	}

	return &ClientManager{
		auth:       auth,
		edgeClient: clientset.NewForConfigOrDie(config),
		clients:    make(map[string]*Client),
	}
}

func (cm *ClientManager) Stop() {
	for _, client := range cm.clients {
		client.Disconnect()
	}
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
