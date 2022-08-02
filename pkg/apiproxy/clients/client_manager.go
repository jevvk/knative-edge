package clients

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"knative.dev/edge/pkg/apiproxy/authentication"

	edgeV1 "knative.dev/edge/pkg/apis/edge/v1"
	clientset "knative.dev/edge/pkg/client/clientset/versioned"
	informers "knative.dev/edge/pkg/client/informers/externalversions"
)

type ClientManager struct {
	auth       *authentication.Authenticator
	ctx        *context.Context
	edgeClient *clientset.Clientset
	clients    map[string]*Client
}

func New(auth *authentication.Authenticator) ClientManager {
	config, err := rest.InClusterConfig()

	if err != nil {
		panic(fmt.Errorf("couldn't retrieve kubernetes client: %s", err))
	}

	return ClientManager{
		auth:       auth,
		edgeClient: clientset.NewForConfigOrDie(config),
		clients:    make(map[string]*Client),
	}
}

func (cm *ClientManager) Listen(ctx context.Context, errChan chan<- *error) error {
	if cm == nil {
		return fmt.Errorf("no client manager provided")
	}

	cm.ctx = &ctx
	config, err := rest.InClusterConfig()

	if err != nil {
		return fmt.Errorf("couldn't retrieve kubernetes client: %s", err)
	}

	client := clientset.NewForConfigOrDie(config)
	// clusters, err := client.EdgeV1().EdgeClusters().List(ctx, metav1.ListOptions{})

	// if err != nil {
	// 	return fmt.Errorf("could no list edge clusters: %s", err)
	// }

	// log.Printf("Found %d existing edge cluster(s).", len(clusters.Items))

	// for _, cluster := range clusters.Items {
	// 	go cm.AddCluster(cluster)
	// }

	informer := informers.NewSharedInformerFactory(client, time.Minute*5).Edge().V1().EdgeClusters().Informer()

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    cm.AddCluster,
		UpdateFunc: cm.UpdateCluster,
		DeleteFunc: cm.DeleteCluster,
	})

	errChan <- nil
	log.Printf("Initialized client manager.")

	go informer.Run(ctx.Done())
	log.Printf("Started client manager watcher.")

	<-ctx.Done()

	log.Printf("Closing %d edge clients connection(s).", len(cm.clients))

	for _, client := range cm.clients {
		client.Disconnect()
	}

	log.Printf("Stopped client manager.")

	return ctx.Err()
}

func (cm *ClientManager) AddCluster(obj interface{}) {
	cluster := obj.(*edgeV1.EdgeCluster)

	if cluster.Status.AuthenticationToken == "" {
		return
	}

	log.Printf("Edge cluster '%s' not yet set up.", cluster.Name)

	token, err := cm.auth.CreateToken()

	if err != nil {
		log.Printf("Error creating token: %s", err)
		return
	}

	cluster.Status.AuthenticationToken = *token
	cluster.Status.ConnectionStatus = edgeV1.Disconnected

	var ctx context.Context

	if cm.ctx != nil {
		ctx = *cm.ctx
	} else {
		ctx = context.TODO()
	}

	cluster, err = cm.edgeClient.EdgeV1().EdgeClusters().UpdateStatus(ctx, cluster, metav1.UpdateOptions{})

	// if we get 410, that means another instance of apiproxy updated the status
	// even if it was updated by something else, a resync will end up setting
	// the status at some point
	if err != nil && !errors.IsGone(err) {
		log.Printf("Error updating edge cluster status: %s", err)
		return
	}

	cm.clients[cluster.Name] = &Client{
		name:       cluster.Name,
		namespaces: cluster.Spec.Namespaces,
	}

	log.Printf("Edge cluster added: %s", cluster.Name)
}

func (cm *ClientManager) UpdateCluster(old, new interface{}) {
	cluster := new.(*edgeV1.EdgeCluster)
	client, exists := cm.clients[cluster.Name]

	if !exists {
		return
	}

	client.UpdateSpec(cluster)

	log.Printf("Edge cluster updated: %s", cluster.Name)
}

func (cm *ClientManager) DeleteCluster(obj interface{}) {
	cluster := obj.(*edgeV1.EdgeCluster)
	client, exists := cm.clients[cluster.Name]

	if !exists {
		return
	}

	err := client.Disconnect()

	if err != nil {
		log.Printf("Error disconnecting cluster edge: %s", err)
	}

	delete(cm.clients, cluster.Name)

	log.Printf("Edge cluster removed: %s", cluster.Name)
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
