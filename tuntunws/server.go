package tuntunws

import (
	"fmt"
	"net/http"
	"tuntuntun"

	"github.com/coder/websocket"
)

const Protocol = "tuntun"

type Server struct {
	handler tuntuntun.Handler
}

func NewServer(handler tuntuntun.Handler) *Server {
	return &Server{handler: handler}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		Subprotocols: []string{Protocol},
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer c.CloseNow()

	if c.Subprotocol() != Protocol {
		c.Close(websocket.StatusPolicyViolation, "client must speak the echo subprotocol")
		return
	}

	err = s.handler.ServeConn(r.Context(), websocket.NetConn(r.Context(), c, websocket.MessageBinary))
	if err != nil {
		fmt.Println("ws serve:", err)
		return
	}
}
