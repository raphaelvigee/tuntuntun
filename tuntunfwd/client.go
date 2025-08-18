package tuntunfwd

import (
	"context"
	"log/slog"
	"net"
	"tuntuntun"
	"tuntuntun/tuntunopener"
)

type Config struct {
	LocalDial   func(ctx context.Context, addr string) (net.Conn, error)
	LocalListen func(ctx context.Context, addr string) (net.Listener, error)
	Logger      *slog.Logger
}

type Client struct {
	cfg Config

	client *tuntunopener.Client
	onPeer func(ctx context.Context, h *tuntunopener.PeerDescriptor)
}

func NewClient(cfg Config, opener tuntuntun.Opener, handler tuntunopener.PeerHandler) *Client {
	return &Client{
		cfg:    cfg,
		onPeer: handler.OnPeer,
		client: tuntunopener.NewClient(opener, handler, tuntunopener.WithLogger(cfg.Logger)),
	}
}

func (c *Client) Start(ctx context.Context) (chan error, error) {
	doneCh, err := c.client.Start(ctx)
	if err != nil {
		return doneCh, err
	}

	if c.onPeer != nil {
		c.onPeer(ctx, c.client.GetPeerDescriptor())
	}

	return doneCh, nil
}

func (c *Client) Close() error {
	return c.client.Close()
}
