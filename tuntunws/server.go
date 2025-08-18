package tuntunws

import (
	"log/slog"
	"net/http"
	"tuntuntun"

	"github.com/coder/websocket"
)

const SubProtocol = "tuntun"

type Option func(s *Server)

func WithLogger(l *slog.Logger) Option {
	return func(s *Server) {
		s.logger = l
	}
}

type Server struct {
	handler tuntuntun.Handler
	logger  *slog.Logger
}

func NewServer(handler tuntuntun.Handler, opts ...Option) *Server {
	s := &Server{
		handler: handler,
	}
	for _, opt := range opts {
		opt(s)
	}

	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		Subprotocols: []string{SubProtocol},
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer c.CloseNow()

	if c.Subprotocol() != SubProtocol {
		c.Close(websocket.StatusPolicyViolation, "client must speak the "+SubProtocol+" subprotocol")
		return
	}

	err = s.handler.ServeConn(r.Context(), websocket.NetConn(r.Context(), c, websocket.MessageBinary))
	if err != nil {
		if s.logger != nil {
			s.logger.Log(r.Context(), slog.LevelError, "ws: failed to serve", slog.String("err", err.Error()))
		}
		return
	}
}
