package tuntunh2

import (
	"context"
	"log/slog"
	"net/http"
	"tuntuntun"
)

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
		if s.logger != nil {
			s.logger.Log(ctx, slog.LevelError, "h2: failed to serve", slog.String("err", err.Error()))
		}
		return
	}
}
