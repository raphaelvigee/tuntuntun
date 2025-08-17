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
	handler tuntuntun.Handler
	peerId  uint64

	requestIdc     atomic.Uint64
	forwardRequest chan forwardRequest
}

type forwardRequest struct {
	requestId uint64
}

func NewClient(opener tuntuntun.Opener, handler tuntuntun.Handler) *Client {
	return &Client{
		opener:  opener,
		handler: handler,
	}
}

func (h *Client) Start(ctx context.Context) (chan error, error) {
	doneCh := make(chan error, 1)
	readyCh := make(chan struct{}, 1)
	errCh := make(chan error, 1)
	go func() {
		err := h.run(ctx, readyCh)
		if err != nil {
			errCh <- err
		}
		doneCh <- err
	}()

	select {
	case <-readyCh:
		return doneCh, nil
	case err := <-errCh:
		return doneCh, err
	}
}

func (h *Client) run(ctx context.Context, ready chan struct{}) error {
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

	ready <- struct{}{}

	var g errgroup.Group
	g.Go(func() error {
		return h.runReader(ctx, conn)
	})
	g.Go(func() error {
		return h.runWriter(ctx, conn)
	})

	return g.Wait()
}

func (h *Client) Open(ctx context.Context, handler tuntuntun.Handler) error {
	conn, err := h.opener.Open(ctx)
	if err != nil {
		return err
	}

	err = WriteConnInit(conn, ConnTypeTun)
	if err != nil {
		return err
	}

	err = WriteTunInit(conn, h.peerId, 0)
	if err != nil {
		return err
	}

	return handler.ServeConn(ctx, conn)
}

func (h *Client) runReader(ctx context.Context, controlConn io.ReadWriteCloser) error {
	dec := json.NewDecoder(controlConn)
	for {
		var msg ControlMessage
		err := dec.Decode(&msg)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}

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
			return nil
		}
	}
}

func (h *Client) GetPeerDescriptor() *PeerDescriptor {
	return &PeerDescriptor{
		ID: h.peerId,
		open: func(ctx context.Context, p *PeerDescriptor, handler tuntuntun.Handler) error {
			return h.Open(ctx, handler)
		},
	}
}
