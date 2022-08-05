package websockets

import (
	"log"

	cloudv1 "edge.knative.dev/pkg/apis/cloud/v1"
)

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
