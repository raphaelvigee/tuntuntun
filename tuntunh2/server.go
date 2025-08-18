package tuntunh2

import (
	"context"
	"fmt"
	"net/http"
	"tuntuntun"
)

type Server struct {
	handler tuntuntun.Handler
}

func NewServer(handler tuntuntun.Handler) *Server {
	return &Server{
		handler: handler,
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !r.ProtoAtLeast(2, 0) {
		http.Error(w, "unsupported proto", http.StatusBadRequest)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "unsupported writer", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	conn := newConn(r.Body, &responseWriterCloser{ResponseWriter: w, close: cancel, f: flusher})
	defer conn.Close()

	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	err := s.handler.ServeConn(ctx, conn)
	if err != nil {
		fmt.Println("h2 serve:", err)
		return
	}
}
