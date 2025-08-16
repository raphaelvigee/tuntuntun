package tuntunws

import (
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/gorilla/websocket"
)

type Conn struct {
	*websocket.Conn

	wsReader          io.Reader
	wsReaderReadCount uint64

	readDone  bool
	writeDone bool
}

var _ net.Conn = (*Conn)(nil)

func newConn(conn *websocket.Conn) *Conn {
	return &Conn{
		Conn: conn,
	}
}

func (c *Conn) Read(b []byte) (int, error) {
	for {
		for c.wsReader == nil {
			messageType, wsReader, err := c.Conn.NextReader()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					return 0, io.EOF
				}

				return 0, err
			}
			if messageType != websocket.BinaryMessage {
				continue
			}

			c.wsReader = wsReader
			c.wsReaderReadCount = 0
		}

		n, err := c.wsReader.Read(b)
		if err != nil {
			if errors.Is(err, io.EOF) {
				c.wsReader = nil
			} else {
				return 0, err
			}
		}

		if n <= 0 {
			continue
		}

		c.wsReaderReadCount++

		if c.wsReaderReadCount > 1 {
			return n, nil
		}

		switch MsgType(b[0]) {
		case MsgTypeData:
			return copy(b, b[1:n]), nil
		case MsgTypeDone:
			c.readDone = true
			c.maybeCloseConn()

			return 0, io.EOF
		default:
			return 0, fmt.Errorf("unknown message type: %d", b[0])
		}
	}
}

func (c *Conn) Write(b []byte) (n int, err error) {
	err = c.writeMsg(MsgTypeData, b)

	return len(b), err
}

type MsgType int8

const (
	MsgTypeData MsgType = 2 // start of text
	MsgTypeDone MsgType = 4 // end of transmission
)

func (c *Conn) writeMsg(typ MsgType, b []byte) error {
	w, err := c.Conn.NextWriter(websocket.BinaryMessage)
	if err != nil {
		return err
	}
	defer w.Close()

	_, err = w.Write([]byte{byte(typ)})
	if err != nil {
		return err
	}
	if len(b) > 0 {
		_, err = w.Write(b)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Conn) SetDeadline(t time.Time) error {
	err1 := c.Conn.SetWriteDeadline(t)
	err2 := c.Conn.SetReadDeadline(t)

	return errors.Join(err1, err2)
}

func (c *Conn) maybeCloseConn() error {
	if !c.readDone || !c.writeDone {
		return nil
	}

	return closeWs(c.Conn)
}

func (c *Conn) Close() error {
	c.writeDone = true
	c.maybeCloseConn()

	return c.writeMsg(MsgTypeDone, nil)
}
