package clients

import (
	"github.com/gorilla/websocket"

	edgeV1 "knative.dev/edge/pkg/apis/edge/v1"
)

type Client struct {
	conn                  *websocket.Conn
	localResourceVersion  *string
	remoteResourceVersion *string

	name       string
	namespaces []string
}

func (c *Client) UpdateSpec(cluster *edgeV1.EdgeCluster) {
	c.name = cluster.Name
	c.namespaces = cluster.Spec.Namespaces
}

func (c *Client) Disconnect() error {
	if c.conn == nil {
		return nil
	}

	return c.conn.Close()
}
