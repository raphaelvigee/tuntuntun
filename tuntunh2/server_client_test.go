package tuntunh2

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"tuntuntun"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"golang.org/x/sync/errgroup"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func echo(t *testing.T, expected string) tuntuntun.HandlerFunc {
	return func(ctx context.Context, conn io.ReadWriteCloser) error {
		defer conn.Close()

		buf := make([]byte, len(expected))
		_, err := io.ReadFull(conn, buf)
		if err != nil {
			return err
		}

		t.Log("SERVER RECEIVED:", len(buf))

		sent := []byte("said: " + string(buf))

		n, err := conn.Write(sent)
		t.Log("SERVER WRITTEN:", n, len(sent))

		return err
	}
}

func roundtrip(t *testing.T, conn net.Conn, sent string) {
	expected := "said: " + sent

	go func() {
		n, err := conn.Write([]byte(sent))
		t.Log("SERVER SENT:", n, len(sent), err)
	}()

	buf := make([]byte, len(expected))
	_, err := io.ReadFull(conn, buf)
	require.NoError(t, err)

	assert.Equal(t, expected, string(buf))
}

func newServer(h tuntuntun.Handler) http.Handler {
	h2s := &http2.Server{
		MaxConcurrentStreams: 250,
	}

	hh := NewServer(h)

	return h2c.NewHandler(hh, h2s)
}

func TestSanity(t *testing.T) {
	toWrite := "hello"

	h := newServer(echo(t, toWrite))

	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	t.Log("URL", srv.URL)

	c := NewClient(srv.URL)

	conn, _, err := c.Connect(t.Context())
	require.NoError(t, err)
	defer conn.Close()

	roundtrip(t, conn, toWrite)
}

func TestSanityLarge(t *testing.T) {
	sent := ""
	for i := range 1024 {
		if sent != "" {
			sent += "_"
		}
		sent += fmt.Sprint(i)
	}

	h := newServer(echo(t, sent))

	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	t.Log("URL", srv.URL)

	c := NewClient(srv.URL)

	conn, _, err := c.Connect(t.Context())
	require.NoError(t, err)
	defer conn.Close()

	roundtrip(t, conn, sent)
}

func TestStress(t *testing.T) {
	toWrite := "hello"

	h := newServer(echo(t, toWrite))

	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	var g errgroup.Group
	for range 1000 {
		g.Go(func() error {
			c := NewClient(srv.URL)

			conn, _, err := c.Connect(t.Context())
			require.NoError(t, err)
			defer conn.Close()

			roundtrip(t, conn, toWrite)

			return nil
		})
	}

	g.Wait()
}
