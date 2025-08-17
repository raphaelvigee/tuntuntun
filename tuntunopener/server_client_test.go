package tuntunopener

import (
	"context"
	"io"
	"net"
	"testing"
	"tuntuntun"

	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

func TestServerOpenSanity(t *testing.T) {
	ctx := t.Context()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	clientConnected := make(chan struct{})
	serverConnected := make(chan struct{})

	srv := NewServer(
		func() Handler {
			return HandlerFunc(func(ctx context.Context, rw io.ReadWriteCloser) error {
				panic("should not be called")
			})
		},
		func(ctx context.Context, h *PeerHandle) {
			err := h.Open(ctx, HandlerFunc(func(ctx context.Context, rw io.ReadWriteCloser) error {
				serverConnected <- struct{}{}

				return nil
			}))
			require.NoError(t, err)

			err = h.Open(ctx, HandlerFunc(func(ctx context.Context, rw io.ReadWriteCloser) error {
				serverConnected <- struct{}{}

				return nil
			}))
			require.NoError(t, err)
		},
	)

	l, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	defer l.Close()

	go func() {
		for {
			c, err := l.Accept()
			require.NoError(t, err)

			go func() {
				defer c.Close()

				err := srv.ServeConn(ctx, c)
				require.NoError(t, err)
			}()
		}
	}()

	c := NewClient(
		tuntuntun.OpenerFunc(func(ctx context.Context) (net.Conn, error) {
			return net.Dial(l.Addr().Network(), l.Addr().String())
		}),
		HandlerFunc(func(ctx context.Context, rw io.ReadWriteCloser) error {
			clientConnected <- struct{}{}

			return nil
		}),
	)

	go func() {
		err := c.Run(ctx)
		require.NoError(t, err)
	}()

	var g errgroup.Group
	g.Go(func() error {
		<-clientConnected

		return nil
	})
	g.Go(func() error {
		<-clientConnected

		return nil
	})
	g.Go(func() error {
		<-serverConnected

		return nil
	})
	g.Go(func() error {
		<-serverConnected

		return nil
	})

	g.Wait()
}

func TestClientOpenSanity(t *testing.T) {
	ctx := t.Context()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	clientConnected := make(chan struct{})
	serverConnected := make(chan struct{})

	srv := NewServer(
		func() Handler {
			return HandlerFunc(func(ctx context.Context, rw io.ReadWriteCloser) error {
				serverConnected <- struct{}{}

				return nil
			})
		},
		nil,
	)

	l, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	defer l.Close()

	go func() {
		for {
			c, err := l.Accept()
			require.NoError(t, err)

			go func() {
				defer c.Close()

				err := srv.ServeConn(ctx, c)
				require.NoError(t, err)
			}()
		}
	}()

	c := NewClient(
		tuntuntun.OpenerFunc(func(ctx context.Context) (net.Conn, error) {
			return net.Dial(l.Addr().Network(), l.Addr().String())
		}),
		HandlerFunc(func(ctx context.Context, rw io.ReadWriteCloser) error {
			clientConnected <- struct{}{}

			return nil
		}),
	)

	go func() {
		err := c.Run(ctx)
		require.NoError(t, err)
	}()

	go func() {
		err := c.Open(ctx, HandlerFunc(func(ctx context.Context, rw io.ReadWriteCloser) error {
			clientConnected <- struct{}{}

			return nil
		}))
		require.NoError(t, err)
	}()

	go func() {
		err := c.Open(ctx, HandlerFunc(func(ctx context.Context, rw io.ReadWriteCloser) error {
			clientConnected <- struct{}{}

			return nil
		}))
		require.NoError(t, err)
	}()

	var g errgroup.Group
	g.Go(func() error {
		<-clientConnected

		return nil
	})
	g.Go(func() error {
		<-clientConnected

		return nil
	})
	g.Go(func() error {
		<-serverConnected

		return nil
	})
	g.Go(func() error {
		<-serverConnected

		return nil
	})

	g.Wait()
}
