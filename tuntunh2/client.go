package tuntunh2

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"

	"golang.org/x/net/http2"
)

func NewClient(url string) *Client {
	return &Client{
		url: url,
		Client: &http.Client{Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLSContext: func(ctx context.Context, network, addr string, cfg *tls.Config) (net.Conn, error) {
				return net.Dial(network, addr)
			},
		}},
	}
}

type Client struct {
	// Client must have an http2.Transport as its transport.
	Client *http.Client
	url    string
}

func (c *Client) Connect(ctx context.Context) (net.Conn, *http.Response, error) {
	reader, writer := io.Pipe()

	// Create a request object to send to the server
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, reader)
	if err != nil {
		return nil, nil, err
	}

	// Apply given context to the sent request
	req = req.WithContext(ctx)

	// Perform the request
	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, nil, err
	}

	// Create a connection
	conn, ctx := newConn(req.Context(), resp.Body, writer)

	// Apply the connection context on the request context
	resp.Request = req.WithContext(ctx)

	return conn, resp, nil
}

func (c *Client) Open(ctx context.Context) (net.Conn, error) {
	conn, _, err := c.Connect(ctx)

	return conn, err
}
