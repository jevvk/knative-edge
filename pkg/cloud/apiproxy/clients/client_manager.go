package clients

import (
	"context"
	"fmt"
	"log"

	"edge.knative.dev/pkg/cloud/apiproxy/authentication"
	"github.com/gorilla/websocket"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/rest"

	cloudv1 "edge.knative.dev/pkg/apis/cloud/v1"
)

type ClientManager struct {
	auth       *authentication.Authenticator
	ctx        *context.Context
	edgeClient *clientset.Clientset
	clients    map[string]*Client
}

func New() *ClientManager {
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

func (cm *ClientManager) AddCluster(cluster *cloudv1.EdgeCluster) *cloudv1.EdgeCluster {
	// deb, err := yaml.Marshal(cluster)

	// if err == nil {
	// 	log.Printf("debug:\n%s", deb)
	// } else {
	// 	log.Printf("debug err: %s", err)
	// }

	if len(cluster.Status.AuthenticationToken) != 0 {
		log.Printf("debug: edge cluster %s already configured with token: '%s' %d", cluster.Name, cluster.Status.AuthenticationToken, len(cluster.Status.AuthenticationToken))
		return nil
	}

	log.Printf("Edge cluster '%s' not yet set up.", cluster.Name)

	token, err := cm.auth.CreateToken()

	if err != nil {
		log.Printf("Error creating token: %s", err)
		return nil
	}

	cluster.Status.AuthenticationToken = *token
	cluster.Status.ConnectionStatus = cloudv1.Disconnected

	cm.clients[cluster.Name] = &Client{
		name:       cluster.Name,
		namespaces: cluster.Spec.Namespaces,
	}

	log.Printf("Edge cluster added: %s", cluster.Name)

	return cluster
}

func (cm *ClientManager) UpdateCluster(cluster *cloudv1.EdgeCluster) *cloudv1.EdgeCluster {
	client, exists := cm.clients[cluster.Name]

	if !exists {
		return cm.AddCluster(cluster)
	}

	client.UpdateSpec(cluster)

	log.Printf("Edge cluster updated: %s", cluster.Name)

	return nil
}

func (cm *ClientManager) DeleteCluster(clusterName string) {
	client, exists := cm.clients[clusterName]

	if !exists {
		return
	}

	err := client.Disconnect()

	if err != nil {
		log.Printf("Error disconnecting cluster edge: %s", err)
	}

	delete(cm.clients, clusterName)

	log.Printf("Edge cluster removed: %s", clusterName)
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
