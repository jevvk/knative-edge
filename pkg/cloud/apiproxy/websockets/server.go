package websockets

// import (
// 	"fmt"
// 	"net/http"

// 	"github.com/gorilla/websocket"
// )

// type WebsocketHandler interface {
// 	HandleConnection(*websocket.Conn)
// }

// type handler struct {
// 	next WebsocketHandler
// }

// func NewHandler(h WebsocketHandler) *handler {
// 	return &handler{next: h}
// }

// var upgrader = websocket.Upgrader{}

// func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
// 	conn, err := upgrader.Upgrade(w, r, nil)

// 	if err != nil {
// 		fmt.Printf("Couldn't upgrade HTTP connection to Websockets: %s", err)
// 		http.Error(w, "websocket support is required", http.StatusBadRequest)
// 		return
// 	}

// 	defer conn.Close()

// 	h.next.HandleConnection(conn)

// 	// for {
// 	// 	msgType, msg, err := conn.ReadMessage()

// 	// 	if err != nil {
// 	// 		fmt.Printf("Enountered an error when reading websocket message: %s", err)
// 	// 		return
// 	// 	}

// 	// 	// Print the message to the console
// 	// 	fmt.Printf("%s sent: %s\n", conn.RemoteAddr(), string(msg))

// 	// 	// Write message back to browser
// 	// 	if err = conn.WriteMessage(msgType, msg); err != nil {
// 	// 		return
// 	// 	}
// 	// }
// }
