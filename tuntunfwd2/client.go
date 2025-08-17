package tuntunfwd2

import (
	"context"
	"errors"
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

func (c *Client) ClientToServer(ctx context.Context, laddr, raddr string) error {
	err := c.client.Open(ctx, tuntuntun.HandlerFunc(func(ctx context.Context, rconn io.ReadWriteCloser) error {
		defer rconn.Close()

		err := WriteInit(rconn, raddr)
		if err != nil {
			return err
		}

		lconn, err := c.cfg.LocalDial(ctx, laddr)
		if err != nil {
			return err
		}
		defer lconn.Close()

		tuntuntun.BidiCopy(rconn, lconn)

		return nil
	}))
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) ServerToClient(ctx context.Context, laddr, raddr string) (net.Listener, error) {
	l, err := c.cfg.LocalListen(ctx, laddr)
	if err != nil {
		return nil, err
	}

	go func() {
		defer l.Close()

		for {
			lconn, err := l.Accept()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					return
				}

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
