package tuntunws

import (
	"context"
	"net"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

func NewClient(url string) *Client {
	if rest, ok := strings.CutPrefix(url, "http:"); ok {
		url = "ws:" + rest
	} else if rest, ok := strings.CutPrefix(url, "https:"); ok {
		url = "wss:" + rest
	}

	return &Client{
		url: url,
	}
}

type Client struct {
	url string
}

func (c *Client) Connect(ctx context.Context) (net.Conn, *http.Response, error) {
	conn, res, err := websocket.DefaultDialer.DialContext(ctx, c.url, nil)
	if err != nil {
		return nil, nil, err
	}

	return newConn(conn), res, nil
}

func (c *Client) Open(ctx context.Context) (net.Conn, error) {
	conn, _, err := c.Connect(ctx)

	return conn, err
}
