package tuntuntun

import (
	"context"
	"io"
)

type Handler interface {
	ServeConn(ctx context.Context, conn io.ReadWriteCloser) error
}

type HandlerFunc func(context.Context, io.ReadWriteCloser) error

func (h HandlerFunc) ServeConn(ctx context.Context, conn io.ReadWriteCloser) error {
	return h(ctx, conn)
}
