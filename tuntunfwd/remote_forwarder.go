package tuntunfwd

import (
	"context"
	"errors"
	"fmt"
	"net"
	"tuntuntun"
)

type Conn struct {
	conn net.Conn
}

func (c *Conn) Close() error {
	return c.conn.Close()
}

func (c *Conn) Read(b []byte) (int, error) {
	return c.conn.Read(b)
}

func (c *Conn) Write(b []byte) (int, error) {
	return c.conn.Write(b)
}

func RemoteForward(opener tuntuntun.Opener, remoteAddr, localAddr string) error {
	f := NewRemoteForwarder(opener)

	err := f.Start(remoteAddr, localAddr)
	if err != nil {
		return err
	}

	<-f.Wait()

	return nil
}

type RemoteForwarder struct {
	opener tuntuntun.Opener

	listener net.Listener
	doneCh   chan struct{}
}

func NewRemoteForwarder(opener tuntuntun.Opener) *RemoteForwarder {
	return &RemoteForwarder{
		opener: opener,
	}
}

func (f *RemoteForwarder) Start(remoteAddr, localAddr string) error {
	l, err := net.Listen("tcp", localAddr)
	if err != nil {
		return err
	}

	f.listener = l
	f.doneCh = make(chan struct{})

	go func() {
		defer close(f.doneCh)
		defer f.Close()

		for {
			lconn, err := l.Accept()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					return
				}

				fmt.Println("fwd accept:", err)
				continue
			}

			go f.handleConn(context.Background(), lconn, remoteAddr)
		}
	}()

	return err
}

func (f *RemoteForwarder) handleConn(ctx context.Context, lconn net.Conn, remoteAddr string) {
	defer lconn.Close()

	rconn, err := RemoteDialContext(ctx, f.opener, remoteAddr)
	if err != nil {
		fmt.Println("fwd dial:", err)
		return
	}
	defer rconn.Close()

	tuntuntun.BidiCopy(rconn, lconn)
}

func (f *RemoteForwarder) LocalAddr() net.Addr {
	if f.listener == nil {
		return nil
	}

	return f.listener.Addr()
}

func (f *RemoteForwarder) Close() error {
	if f.listener == nil {
		return nil
	}

	return f.listener.Close()
}

func (f *RemoteForwarder) Wait() <-chan struct{} {
	return f.doneCh
}
