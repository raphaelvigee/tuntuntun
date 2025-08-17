package tuntunopener

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync/atomic"
	"tuntuntun"

	"golang.org/x/sync/errgroup"
)

type Client struct {
	opener  tuntuntun.Opener
	handler Handler
	peerId  uint64

	requestIdc     atomic.Uint64
	forwardRequest chan forwardRequest
}

type forwardRequest struct {
	requestId uint64
}

func NewClient(opener tuntuntun.Opener, handler Handler) *Client {
	return &Client{
		opener:  opener,
		handler: handler,
	}
}

func (h *Client) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	conn, err := h.opener.Open(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	go func() {
		<-ctx.Done()
		conn.Close()
	}()

	err = WriteConnInit(conn, ConnTypeControl)
	if err != nil {
		return err
	}

	err = json.NewEncoder(conn).Encode(ControlMessage{
		Version:     ControlMessageV1,
		InitRequest: &InitRequestMessage{},
	})
	if err != nil {
		return err
	}

	var msg ControlMessage
	err = json.NewDecoder(conn).Decode(&msg)
	if err != nil {
		return err
	}

	if msg.Version != ControlMessageV1 {
		return errors.New("invalid version")
	}

	if msg.InitResponse == nil {
		return errors.New("init response is nil")
	}

	h.peerId = msg.InitResponse.PeerID

	var g errgroup.Group
	g.Go(func() error {
		return h.runReader(ctx, conn)
	})
	g.Go(func() error {
		return h.runWriter(ctx, conn)
	})

	return g.Wait()
}

func (h *Client) Open(ctx context.Context) (io.ReadWriteCloser, error) {
	conn, err := h.opener.Open(ctx)
	if err != nil {
		return nil, err
	}

	err = WriteConnInit(conn, ConnTypeTun)
	if err != nil {
		return nil, err
	}

	err = WriteTunInit(conn, h.peerId, 0)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func (h *Client) runReader(ctx context.Context, controlConn io.ReadWriteCloser) error {
	dec := json.NewDecoder(controlConn)
	for {
		var msg ControlMessage
		err := dec.Decode(&msg)
		if err != nil {
			return err
		}

		if msg.Version != ControlMessageV1 {
			return errors.New("invalid version")
		}

		switch {
		case msg.ConnRequest != nil:
			go h.handleConnRequest(ctx, msg.ConnRequest)
		default:
			return errors.New("invalid init request")
		}
	}
}

func (h *Client) handleConnRequest(ctx context.Context, req *ConnRequestMessage) {
	conn, err := h.opener.Open(ctx)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Close()

	err = WriteConnInit(conn, ConnTypeTun)
	if err != nil {
		fmt.Println(err)
		return
	}

	err = WriteTunInit(conn, h.peerId, req.RequestID)
	if err != nil {
		fmt.Println(err)
		return
	}

	err = h.handler.ServeConn(ctx, conn)
	if err != nil {
		fmt.Println(err)
		return
	}
}

func (h *Client) runWriter(ctx context.Context, controlConn io.ReadWriteCloser) error {
	for {
		select {
		case <-ctx.Done():
			//case msg := <-h.forwardRequest:
			//	err := json.NewEncoder(controlConn).Encode(ControlMessage{
			//		Version: ControlMessageV1,
			//		ConnRequest: &ConnRequestMessage{
			//			RequestID: 0,
			//		},
			//	})
			//	if err != nil {
			//		return err
			//	}
		}
	}
}
