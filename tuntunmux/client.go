package tuntunmux

import (
	"context"
	"log/slog"
	"net"
	"sync"
	"tuntuntun"

	"github.com/hashicorp/yamux"
)

type ClientOption func(s *Client)

func WithClientLogger(l *slog.Logger) ClientOption {
	return func(s *Client) {
		s.logger = l
	}
}

func NewClient(opener tuntuntun.Opener, opts ...ClientOption) *Client {
	s := &Client{
		opener: opener,
	}
	for _, opt := range opts {
		opt(s)
	}

	return s
}

type Client struct {
	opener tuntuntun.Opener
	logger *slog.Logger

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
	cfg.Logger = logger{logger: c.logger, ctx: ctx}
	cfg.LogOutput = nil

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
