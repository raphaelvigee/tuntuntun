package tuntunfwd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"tuntuntun"
)

type ServerConfig struct {
	// Allows the client to request a port forwarding from the server
	AllowServerForward func(ctx context.Context, addr string) error
	// Allows the client to forward a port to the server
	OnClientForward func(ctx context.Context, conn io.ReadWriteCloser) error
}

type Server struct {
	ServerConfig
}

func (s *Server) ServeConn(ctx context.Context, conn io.ReadWriteCloser) error {
	defer conn.Close()

	var i Init
	err := json.NewDecoder(conn).Decode(&i)
	if err != nil {
		return err
	}

	if i.Version != V1 {
		return errors.New("unsupported version")
	}

	switch i.Mode {
	case ServerForward:
		if i.Addr == "" {
			return errors.New("addr is empty")
		}

		if s.AllowServerForward == nil {
			return errors.New("server denied forwarding request")
		}

		err := s.AllowServerForward(ctx, i.Addr)
		if err != nil {
			return fmt.Errorf("server denied forwarding request: %w", err)
		}

		l, err := net.Dial("tcp", i.Addr)
		if err != nil {
			return err
		}
		defer l.Close()

		tuntuntun.BidiCopy(conn, l)

		return nil
	case ClientForward:
		if s.OnClientForward == nil {
			return errors.New("server doesnt accept forwarding")
		}

		return s.OnClientForward(ctx, conn)
	default:
		return fmt.Errorf("unknown server forward mode: %q", i.Mode)
	}
}

func NewServer(cfg ServerConfig) *Server {
	return &Server{
		ServerConfig: cfg,
	}
}
