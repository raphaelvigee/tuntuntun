package tuntunmux

import (
	"context"
	"io"
	"net"
	"testing"
	"tuntuntun"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

func TestSanity(t *testing.T) {
	l, err := net.Listen("tcp", ":0")
	require.NoError(t, err)

	go func() {
		srv := NewServer(tuntuntun.HandlerFunc(func(ctx context.Context, conn io.ReadWriteCloser) error {
			defer conn.Close()

			buf := make([]byte, 1024)
			n, err := conn.Read(buf)
			require.NoError(t, err)

			conn.Write([]byte("said: " + string(buf[:n])))

			return nil
		}))

		for {
			conn, err := l.Accept()
			require.NoError(t, err)

			go func() {
				defer conn.Close()

				err := srv.ServeConn(t.Context(), conn)
				require.NoError(t, err)
			}()
		}
	}()

	c := NewClient(tuntuntun.OpenerFunc(func(ctx context.Context) (net.Conn, error) {
		return net.Dial("tcp", l.Addr().String())
	}))
	require.NoError(t, err)
	defer c.Close()

	conn, err := c.Open(t.Context())
	require.NoError(t, err)

	go func() {
		conn.Write([]byte("hello"))
		conn.Close()
	}()

	b, err := io.ReadAll(conn)
	require.NoError(t, err)

	assert.Equal(t, "said: hello", string(b))
}

func TestStress(t *testing.T) {
	l, err := net.Listen("tcp", ":0")
	require.NoError(t, err)

	go func() {
		srv := NewServer(tuntuntun.HandlerFunc(func(ctx context.Context, conn io.ReadWriteCloser) error {
			defer conn.Close()

			buf := make([]byte, 1024)
			n, err := conn.Read(buf)
			require.NoError(t, err)

			conn.Write([]byte("said: " + string(buf[:n])))

			return nil
		}))

		for {
			conn, err := l.Accept()
			require.NoError(t, err)

			go func() {
				defer conn.Close()

				err := srv.ServeConn(t.Context(), conn)
				require.NoError(t, err)
			}()
		}
	}()

	c := NewClient(tuntuntun.OpenerFunc(func(ctx context.Context) (net.Conn, error) {
		return net.Dial("tcp", l.Addr().String())
	}))
	require.NoError(t, err)
	defer c.Close()

	var g errgroup.Group
	for range 1000 {
		g.Go(func() error {
			conn, err := c.Open(t.Context())
			require.NoError(t, err)

			go func() {
				conn.Write([]byte("hello"))
				conn.Close()
			}()

			b, err := io.ReadAll(conn)
			require.NoError(t, err)

			assert.Equal(t, "said: hello", string(b))

			return nil
		})
	}

	g.Wait()
}
