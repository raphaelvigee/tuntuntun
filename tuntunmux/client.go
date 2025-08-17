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
}

func (c *Client) getSession(ctx context.Context) (*yamux.Session, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.err != nil {
		return nil, c.err
	}

	if c.sess != nil && !c.sess.IsClosed() {
		return c.sess, nil
	}

	if c.sess != nil {
		_ = c.sess.Close()
	}

	c.sess, c.err = c.openSession(ctx)

	return c.sess, c.err
}

func (c *Client) openSession(ctx context.Context) (*yamux.Session, error) {
	conn, err := c.opener.Open(ctx)
	if err != nil {
		return nil, err
	}

	cfg := yamux.DefaultConfig()
	//cfg.Logger = something

	sess, err := yamux.Client(conn, cfg)
	if err != nil {
		return nil, err
	}

	return sess, nil
}

func (c *Client) Open(ctx context.Context) (net.Conn, error) {
	sess, err := c.getSession(ctx)
	if err != nil {
		return nil, err
	}

	return sess.Open()
}

func (c *Client) Close() error {
	if c.sess == nil {
		return nil
	}

	defer func() {
		c.mu.Lock()
		defer c.mu.Unlock()

		c.sess = nil
		c.err = nil
	}()

	return c.sess.Close()
}

func NewClient(opener tuntuntun.Opener) *Client {
	return &Client{
		opener: opener,
	}
}
