package tuntunfwd2

import (
	"context"
	"fmt"
	"io"
	"net"
	"tuntuntun"
	"tuntuntun/tuntunopener"
)

type Config struct {
	LocalDial   func(ctx context.Context, addr string) (net.Conn, error)
	LocalListen func(ctx context.Context, addr string) (net.Listener, error)
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
		client: tuntunopener.NewClient(opener, handler),
	}
}

func (c *Client) Start(ctx context.Context) error {
	err := c.client.Start(ctx)
	if err != nil {
		return err
	}

	if c.onPeer != nil {
		c.onPeer(ctx, c.client.GetPeerDescriptor())
	}

	return nil
}

func (c *Client) Open(ctx context.Context, laddr, raddr string) (net.Listener, error) {
	l, err := c.cfg.LocalListen(ctx, laddr)
	if err != nil {
		return nil, err
	}

	go func() {
		for {
			lconn, err := l.Accept()
			if err != nil {
				fmt.Println(err)
				return
			}

			go func() {
				err := c.client.Open(ctx, tuntuntun.HandlerFunc(func(ctx context.Context, rconn io.ReadWriteCloser) error {
					defer lconn.Close()
					defer rconn.Close()

					err := WriteInit(rconn, raddr)
					if err != nil {
						return err
					}

					tuntuntun.BidiCopy(rconn, lconn)

					return nil
				}))
				if err != nil {
					fmt.Println(err)
				}
			}()
		}
	}()

	return l, nil
}
