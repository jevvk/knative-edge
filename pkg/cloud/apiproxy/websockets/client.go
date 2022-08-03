package websockets

import (
	"github.com/gorilla/websocket"

	cloudv1 "edge.knative.dev/pkg/apis/cloud/v1"
)

type Client struct {
	conn                  *websocket.Conn
	localResourceVersion  *string
	remoteResourceVersion *string

	name       string
	namespaces []string
}

func (c *Client) UpdateSpec(cluster *cloudv1.EdgeCluster) {
	c.name = cluster.Name
	c.namespaces = cluster.Spec.Namespaces
}

func (c *Client) Disconnect() error {
	if c.conn == nil {
		return nil
	}

	return c.conn.Close()
}
