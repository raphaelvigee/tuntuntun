package tuntunfwd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"tuntuntun"
	"tuntuntun/tuntunopener"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testServer(t *testing.T) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello"))
	}))
	t.Cleanup(srv.Close)

	return srv
}

func TestServerOpenSanity(t *testing.T) {
	t.SkipNow()

	targetSrv := testServer(t)

	ctx := t.Context()

	cfg := Config{
		LocalDial: func(ctx context.Context, addr string) (net.Conn, error) {
			return net.Dial("tcp", addr)
		},
		LocalListen: func(ctx context.Context, addr string) (net.Listener, error) {
			return net.Listen("tcp", addr)
		},
	}

	receivedCh := make(chan string)

	srv := NewServer(func() (tuntunopener.PeerHandler, error) {
		return DefaultPeerHandler(cfg, []string{targetSrv.Listener.Addr().String()}, func(ctx context.Context, raddr, laddr string) {
			fmt.Println("Listening on ", laddr)

			res, err := http.Get("http://" + laddr)
			require.NoError(t, err)

			defer res.Body.Close()
			b, err := io.ReadAll(res.Body)
			require.NoError(t, err)

			receivedCh <- string(b)
		}), nil
	})

	l, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	defer l.Close()

	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					return
				}

				require.NoError(t, err)
			}

			go func() {
				defer c.Close()

				err := srv.ServeConn(ctx, c)
				require.NoError(t, err)
			}()
		}
	}()

	c := NewClient(
		cfg,
		tuntuntun.OpenerFunc(func(ctx context.Context) (net.Conn, error) {
			return net.Dial(l.Addr().Network(), l.Addr().String())
		}),
		DefaultPeerHandler(cfg, nil, func(ctx context.Context, raddr, laddr string) {
			panic("should not be called")
		}),
	)
	defer c.Close()

	_, err = c.Start(ctx)
	require.NoError(t, err)

	received := <-receivedCh

	assert.Equal(t, "hello", received)
}
