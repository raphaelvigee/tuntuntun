package tuntunfwd

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"tuntuntun"
)

type Server struct {
}

func (s *Server) ServeConn(ctx context.Context, conn io.ReadWriteCloser) error {
	defer conn.Close()

	var i Init
	err := json.NewDecoder(conn).Decode(&i)
	if err != nil {
		return err
	}

	l, err := net.Dial("tcp", i.Addr)
	if err != nil {
		return err
	}
	defer l.Close()

	tuntuntun.BidiCopy(conn, l)

	return nil
}

func NewServer() *Server {
	return &Server{}
}
