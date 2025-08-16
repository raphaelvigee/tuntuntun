package tuntunh2

import (
	"context"
	"fmt"
	"io"
	"net/http/httptest"
	"testing"
	"tuntuntun"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"golang.org/x/sync/errgroup"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func echo(t *testing.T) tuntuntun.HandlerFunc {
	return func(ctx context.Context, rw io.ReadWriteCloser) error {
		defer rw.Close()

		b, err := io.ReadAll(rw)
		if err != nil {
			return err
		}

		t.Log("SERVER RECEIVED:", len(b))

		sent := []byte("said: " + string(b))

		n, err := rw.Write(sent)
		t.Log("SERVER WRITEN:", n, len(sent))

		return err
	}
}

func TestSanity(t *testing.T) {
	h := NewServer(echo(t))

	h2s := &http2.Server{
		MaxConcurrentStreams: 250,
	}
	srv := httptest.NewServer(h2c.NewHandler(h, h2s))
	t.Cleanup(srv.Close)

	t.Log("URL", srv.URL)

	c := NewClient(srv.URL)

	conn, _, err := c.Connect(t.Context())
	require.NoError(t, err)

	go func() {
		_, _ = conn.Write([]byte("hello"))
		_ = conn.Close()
	}()

	b, err := io.ReadAll(conn)
	require.NoError(t, err)

	assert.Equal(t, "said: hello", string(b))
}

func TestSanityLarge(t *testing.T) {
	h := NewServer(echo(t))

	h2s := &http2.Server{
		MaxConcurrentStreams: 250,
	}
	srv := httptest.NewServer(h2c.NewHandler(h, h2s))
	t.Cleanup(srv.Close)

	t.Log("URL", srv.URL)

	c := NewClient(srv.URL)

	conn, _, err := c.Connect(t.Context())
	require.NoError(t, err)

	sent := ""
	for i := range 1024 {
		if sent != "" {
			sent += "_"
		}
		sent += fmt.Sprint(i)
	}

	go func() {
		n, err := conn.Write([]byte(sent))
		t.Log("CLIENT WRITTEN:", n, err, len(sent))
		_ = conn.Close()
	}()

	b, err := io.ReadAll(conn)
	require.NoError(t, err)

	assert.Equal(t, "said: "+sent, string(b))
}

func TestStress(t *testing.T) {
	h := NewServer(echo(t))

	h2s := &http2.Server{
		MaxConcurrentStreams: 250,
	}
	srv := httptest.NewServer(h2c.NewHandler(h, h2s))
	t.Cleanup(srv.Close)

	var g errgroup.Group
	for range 1000 {
		g.Go(func() error {
			c := NewClient(srv.URL)

			conn, _, err := c.Connect(t.Context())
			require.NoError(t, err)

			go func() {
				_, _ = conn.Write([]byte("hello"))
				_ = conn.Close()
			}()

			b, err := io.ReadAll(conn)
			require.NoError(t, err)

			assert.Equal(t, "said: hello", string(b))

			return nil
		})
	}

	g.Wait()
}
