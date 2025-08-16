package tuntuntun

import (
	"context"
	"errors"
	"net"
	"sync/atomic"
)

type Opener interface {
	Open(ctx context.Context) (net.Conn, error)
}

func NewOpenerFuncOnce(f func(ctx context.Context) (net.Conn, error)) Opener {
	var b atomic.Bool
	return OpenerFunc(func(ctx context.Context) (net.Conn, error) {
		if b.Swap(true) {
			return nil, errors.New("already open")
		}

		return f(ctx)
	})
}

type OpenerFunc func(ctx context.Context) (net.Conn, error)

func (f OpenerFunc) Open(ctx context.Context) (net.Conn, error) {
	return f(ctx)
}
