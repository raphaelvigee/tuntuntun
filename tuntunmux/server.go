package tuntunmux

import (
	"context"
	"errors"
	"fmt"
	"io"
	"tuntuntun"

	"github.com/hashicorp/yamux"
)

type Server struct {
	handler tuntuntun.Handler
}

func NewServer(h tuntuntun.Handler) *Server {
	return &Server{
		handler: h,
	}
}

func (s *Server) ServeConn(ctx context.Context, conn io.ReadWriteCloser) error {
	defer conn.Close()

	cfg := yamux.DefaultConfig()
	//cfg.Logger = something

	sess, err := yamux.Server(conn, cfg)
	if err != nil {
		return err
	}
	defer sess.Close()

	go func() {
		select {
		case <-ctx.Done():
			sess.Close()
		case <-sess.CloseChan():
		}
	}()

	for {
		conn, err := sess.AcceptStreamWithContext(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, io.EOF) {
				return nil
			}

			return err
		}

		go func() {
			defer conn.Close()

			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			err := s.handler.ServeConn(ctx, conn)
			if err != nil {
				fmt.Println("mux handle:", err)
			}
		}()
	}
}
