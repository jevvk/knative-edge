package websockets

import (
	"fmt"
	"net/http"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/gorilla/websocket"
)

type WebsocketHandler interface {
	HandleConnection(*websocket.Conn)
}

type handler struct {
	next WebsocketHandler
}

func NewServer(addr string, h WebsocketHandler) *http.Server {
	return &http.Server{
		Addr:    addr,
		Handler: h2c.NewHandler(&handler{next: h}, &http2.Server{}),
	}
}

var upgrader = websocket.Upgrader{}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		fmt.Printf("Couldn't upgrade HTTP connection to Websockets: %s", err)
		http.Error(w, "websocket support is required", http.StatusBadRequest)
		return
	}

	defer conn.Close()

	h.next.HandleConnection(conn)

	// for {
	// 	msgType, msg, err := conn.ReadMessage()

	// 	if err != nil {
	// 		fmt.Printf("Enountered an error when reading websocket message: %s", err)
	// 		return
	// 	}

	// 	// Print the message to the console
	// 	fmt.Printf("%s sent: %s\n", conn.RemoteAddr(), string(msg))

	// 	// Write message back to browser
	// 	if err = conn.WriteMessage(msgType, msg); err != nil {
	// 		return
	// 	}
	// }
}
