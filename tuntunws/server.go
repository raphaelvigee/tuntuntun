package tuntunws

import (
	"fmt"
	"log"
	"net/http"
	"time"
	"tuntuntun"

	"github.com/gorilla/websocket"
)

type Server struct {
	handler tuntuntun.Handler
}

func NewServer(handler tuntuntun.Handler) *Server {
	return &Server{handler: handler}
}

var upgrader = websocket.Upgrader{}

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10
)

func (s *Server) ping(ws *websocket.Conn, done <-chan struct{}) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := ws.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(writeWait)); err != nil {
				log.Println("ping:", err)
			}
		case <-done:
			return
		}
	}
}

func closeWs(c *websocket.Conn) error {
	defer c.Close()

	err := c.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
	if err != nil {
		return err
	}

	return c.Close()
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer closeWs(ws)

	//go s.ping(ws, r.Context().Done())

	err = s.handler.ServeConn(r.Context(), newConn(ws))
	if err != nil {
		fmt.Println("ws serve:", err)
		return
	}
}
