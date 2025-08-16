package tuntunfwd

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"tuntuntun"

	"github.com/stretchr/testify/require"
)

func startServer(t *testing.T, f func(net.Conn)) net.Listener {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { listener.Close() })

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					return
				}

				t.Log("error accepting connection", err)
				return
			}

			go func() {
				defer conn.Close()

				f(conn)
			}()
		}
	}()

	return listener
}

func TestSanity(t *testing.T) {
	targetListener := startServer(t, func(conn net.Conn) {
		_, _ = conn.Write([]byte("hello"))
	})

	srv := NewServer()
	fwdListener := startServer(t, func(conn net.Conn) {
		err := srv.ServeConn(context.Background(), conn)
		require.NoError(t, err)
	})

	conn, err := DialContext(t.Context(), tuntuntun.NewOpenerFuncOnce(func(ctx context.Context) (net.Conn, error) {
		return net.Dial(fwdListener.Addr().Network(), fwdListener.Addr().String())
	}), targetListener.Addr().String())
	require.NoError(t, err)

	b, err := io.ReadAll(conn)
	require.NoError(t, err)

	require.Equal(t, "hello", string(b))
}
