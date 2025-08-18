package tuntunh2

import (
	"errors"
	"io"
	"net"
	"time"
)

type h2Addr struct {
}

func (a h2Addr) Network() string {
	return "h2"
}

func (a h2Addr) String() string {
	return "h2/unknown-addr"
}

type Conn struct {
	r io.ReadCloser
	w io.WriteCloser
}

var _ net.Conn = (*Conn)(nil)

func newConn(r io.ReadCloser, wc io.WriteCloser) *Conn {
	return &Conn{
		r: r,
		w: wc,
	}
}

func (c *Conn) LocalAddr() net.Addr {
	return h2Addr{}
}

func (c *Conn) RemoteAddr() net.Addr {
	return h2Addr{}
}

func (c *Conn) SetDeadline(t time.Time) error {
	if c, ok := c.w.(interface {
		SetDeadline(deadline time.Time) error
	}); ok {
		return c.SetDeadline(t)
	}

	err1 := c.SetReadDeadline(t)
	err2 := c.SetWriteDeadline(t)

	return errors.Join(err1, err2)
}

func (c *Conn) SetReadDeadline(t time.Time) error {
	if c, ok := c.w.(interface {
		SetReadDeadline(deadline time.Time) error
	}); ok {
		return c.SetReadDeadline(t)
	}

	return nil
}

func (c *Conn) SetWriteDeadline(t time.Time) error {
	if c, ok := c.w.(interface {
		SetWriteDeadline(deadline time.Time) error
	}); ok {
		return c.SetWriteDeadline(t)
	}

	return nil
}

func (c *Conn) Write(data []byte) (int, error) {
	return c.w.Write(data)
}

func (c *Conn) Read(data []byte) (int, error) {
	return c.r.Read(data)
}

func (c *Conn) Close() error {
	err1 := c.r.Close()
	err2 := c.w.Close()

	return errors.Join(err1, err2)
}
