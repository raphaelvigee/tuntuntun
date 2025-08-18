package tuntunws

import (
	"context"
	"net"
	"net/http"

	"github.com/coder/websocket"
)

func NewClient(url string) *Client {
	return &Client{
		url: url,
	}
}

type Client struct {
	url string
}

func (c *Client) Connect(ctx context.Context) (net.Conn, *http.Response, error) {
	conn, res, err := websocket.Dial(ctx, c.url, &websocket.DialOptions{
		Subprotocols: []string{SubProtocol},
	})
	if err != nil {
		return nil, nil, err
	}

	return websocket.NetConn(ctx, conn, websocket.MessageBinary), res, nil
}

func (c *Client) Open(ctx context.Context) (net.Conn, error) {
	conn, _, err := c.Connect(ctx)

	return conn, err
}
