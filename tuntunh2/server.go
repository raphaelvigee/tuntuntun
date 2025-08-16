package tuntunh2

import (
	"net/http"
	"tuntuntun"
)

type Server struct {
	Upgrader Upgrader
	handler  tuntuntun.Handler
}

func NewServer(handler tuntuntun.Handler) *Server {
	return &Server{
		handler: handler,
		Upgrader: Upgrader{
			StatusCode: http.StatusOK,
		},
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := s.Upgrader.Accept(w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer conn.Close()

	err = s.handler.ServeConn(r.Context(), conn)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
