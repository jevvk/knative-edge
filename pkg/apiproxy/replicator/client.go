package replicator

import (
	"github.com/gorilla/websocket"
)

type Client struct {
	conn                  websocket.Conn
	localResourceVersion  string
	remoteResourceVersion string
}

func (c *Client) PushUpdate() {

}
