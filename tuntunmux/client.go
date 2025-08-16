package tuntunmux

import (
	"context"
	"net"
	"sync"
	"tuntuntun"

	"github.com/hashicorp/yamux"
)

type Client struct {
	opener tuntuntun.Opener

	mu   sync.Mutex
	sess *yamux.Session
	err  error

	once sync.Once
}

func (c *Client) yamuxSession(ctx context.Context) (*yamux.Session, error) {
	if c.sess != nil || c.err != nil {
		return c.sess, c.err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.sess != nil || c.err != nil {
		return c.sess, c.err
	}

	c.sess, c.err = c.openSession(ctx)

	return c.sess, c.err
}

func (c *Client) openSession(ctx context.Context) (*yamux.Session, error) {
	conn, err := c.opener.Open(ctx)
	if err != nil {
		return nil, err
	}

	sess, err := yamux.Client(conn, nil)
	if err != nil {
		return nil, err
	}

	return sess, nil
}

func (c *Client) Open(ctx context.Context) (net.Conn, error) {
	sess, err := c.openSession(ctx)
	if err != nil {
		return nil, err
	}

	return sess.Open()
}

func (c *Client) Close() error {
	if c.sess == nil {
		return nil
	}

	return c.sess.Close()
}

func NewClient(opener tuntuntun.Opener) *Client {
	return &Client{
		opener: opener,
	}
}
