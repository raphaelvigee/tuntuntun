package tuntunmux

import (
	"context"
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

	for {
		conn, err := sess.AcceptStreamWithContext(ctx)
		if err != nil {
			return err
		}

		go func() {
			defer conn.Close()

			err := s.handler.ServeConn(ctx, conn)
			if err != nil {
				fmt.Println("mux handle:", err)
			}
		}()
	}
}
