package tuntunmux

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"testing"
	"tuntuntun"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

func roundtrip(t *testing.T, conn net.Conn, sent string) {
	expected := "said: " + sent

	go func() {
		n, err := conn.Write([]byte(sent))
		t.Log("CLIENT SENT:", n, len(sent), err)
	}()

	buf := make([]byte, len(expected))
	n, err := io.ReadFull(conn, buf)
	require.NoError(t, err)
	buf = buf[:n]

	t.Log("CLIENT RECEIVED:", n, len(buf), err)

	assert.Equal(t, expected, string(buf))
}

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

func TestSanity(t *testing.T) {
	toWrite := "hello"

	l, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	defer l.Close()

	go func() {
		srv := NewServer(echo(t, toWrite))

		for {
			conn, err := l.Accept()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					return
				}

				require.NoError(t, err)
			}

			t.Log("SERVER ACCEPTED:", conn.RemoteAddr())

			go func() {
				defer conn.Close()

				err := srv.ServeConn(t.Context(), conn)
				if err != nil {
					t.Log(err)
				}
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
	defer conn.Close()

	roundtrip(t, conn, toWrite)
}

func TestStress(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	toWrite := "hello"

	l, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	defer l.Close()

	var wg sync.WaitGroup

	go func() {
		srv := NewServer(echo(t, toWrite))

		for {
			conn, err := l.Accept()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					return
				}

				require.NoError(t, err)
			}

			t.Log("SERVER ACCEPTED:", conn.RemoteAddr())

			wg.Add(1)

			go func() {
				defer wg.Done()
				defer conn.Close()

				err := srv.ServeConn(ctx, conn)
				if err != nil {
					t.Log(err)
				}
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
			defer conn.Close()

			roundtrip(t, conn, toWrite)

			return nil
		})
	}

	require.NoError(t, g.Wait())

	c.Close()

	wg.Wait()
}
