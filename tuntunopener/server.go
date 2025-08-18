package tuntunopener

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"tuntuntun"

	"golang.org/x/sync/errgroup"
)

const ControlMessageV1 = 1

type ControlMessage struct {
	Version      int                  `json:"version"`
	InitRequest  *InitRequestMessage  `json:"init_request,omitempty"`
	InitResponse *InitResponseMessage `json:"init_response,omitempty"`
	ConnRequest  *ConnRequestMessage  `json:"conn_request,omitempty"`
}

type InitRequestMessage struct{}

type InitResponseMessage struct {
	PeerID uint64 `json:"peer_id"`
}

type ConnRequestMessage struct {
	RequestID uint64 `json:"req_id"`
}

type ConnType uint16

const (
	ConnTypeControl ConnType = 1
	ConnTypeTun     ConnType = 2
)

const ConnInitV1 = 1

const ConnInitHeader = 100

func ReadConnInit(r io.Reader) (ConnType, error) {
	b := make([]byte, ConnInitHeader)
	_, err := io.ReadFull(r, b)
	if err != nil {
		return 0, err
	}

	version := binary.LittleEndian.Uint16(b[0:2])
	if version != ConnInitV1 {
		return 0, errors.New("invalid version")
	}

	connType := ConnType(binary.LittleEndian.Uint16(b[2:4]))

	return connType, nil
}

func WriteConnInit(r io.Writer, connType ConnType) error {
	b := make([]byte, ConnInitHeader)
	binary.LittleEndian.PutUint16(b[0:2], ConnInitV1)
	binary.LittleEndian.PutUint16(b[2:4], uint16(connType))

	_, err := r.Write(b)
	if err != nil {
		return err
	}

	return nil
}

const TunInitHeader = 100

const TunInitV1 = 1

func ReadTunInit(r io.Reader) (uint64, uint64, error) {
	b := make([]byte, TunInitHeader)
	_, err := io.ReadFull(r, b)
	if err != nil {
		return 0, 0, err
	}

	version := binary.LittleEndian.Uint16(b[0:2])
	if version != TunInitV1 {
		return 0, 0, errors.New("invalid version")
	}

	peerId := binary.LittleEndian.Uint64(b[2:10])
	reqId := binary.LittleEndian.Uint64(b[10:18])

	return peerId, reqId, nil
}

func WriteTunInit(r io.Writer, peerId, reqId uint64) error {
	b := make([]byte, TunInitHeader)
	binary.LittleEndian.PutUint16(b[0:2], TunInitV1)
	binary.LittleEndian.PutUint64(b[2:10], peerId)
	binary.LittleEndian.PutUint64(b[10:18], reqId)

	_, err := r.Write(b)
	if err != nil {
		return err
	}

	return nil
}

type PeerHandler interface {
	tuntuntun.Handler

	OnPeer(ctx context.Context, h *PeerDescriptor)
}

type PeerHandlerFunc struct {
	OnPeerFunc    func(ctx context.Context, h *PeerDescriptor)
	ServeConnFunc func(ctx context.Context, conn io.ReadWriteCloser) error
}

func (p PeerHandlerFunc) OnPeer(ctx context.Context, h *PeerDescriptor) {
	if p.OnPeerFunc == nil {
		return
	}

	p.OnPeerFunc(ctx, h)
}

func (p PeerHandlerFunc) ServeConn(ctx context.Context, conn io.ReadWriteCloser) error {
	if p.ServeConnFunc == nil {
		return errors.New("not implemented")
	}

	return p.ServeConnFunc(ctx, conn)
}

type Server struct {
	handlerFactory func() (PeerHandler, error)

	peersm  sync.Mutex
	peers   map[uint64]*PeerDescriptor
	peerIdc atomic.Uint64

	openRequest chan openRequest
}

type openRequest struct {
	reqId  uint64
	doneCh chan error
}

type PeerDescriptor struct {
	open func(ctx context.Context, p *PeerDescriptor, handler tuntuntun.Handler) error
	ID   uint64

	handler tuntuntun.Handler

	reqIdc     atomic.Uint64
	reqHandler map[uint64]tuntuntun.Handler
	ctx        context.Context
}

func (p *PeerDescriptor) Open(ctx context.Context, handler tuntuntun.Handler) error {
	return p.open(ctx, p, handler)
}

func NewServer(handlerFactory func() (PeerHandler, error)) *Server {
	return &Server{
		handlerFactory: handlerFactory,
		peers:          map[uint64]*PeerDescriptor{},
		openRequest:    make(chan openRequest),
	}
}

func (s *Server) ServeConn(ctx context.Context, conn io.ReadWriteCloser) error {
	connType, err := ReadConnInit(conn)
	if err != nil {
		return err
	}

	switch connType {
	case ConnTypeControl:
		var init ControlMessage
		err := json.NewDecoder(conn).Decode(&init)
		if err != nil {
			return err
		}

		if init.Version != ControlMessageV1 {
			return errors.New("invalid version")
		}

		if init.InitRequest == nil {
			return fmt.Errorf("expected init_request, got %#v", init)
		}

		handler, err := s.handlerFactory()
		if err != nil {
			return err
		}

		peerHandle := &PeerDescriptor{
			ID:         s.peerIdc.Add(1),
			handler:    handler,
			reqHandler: make(map[uint64]tuntuntun.Handler),
			open: func(ctx context.Context, p *PeerDescriptor, handler tuntuntun.Handler) error {
				return s.peerOpen(ctx, p, handler)
			},
			ctx: ctx,
		}

		err = json.NewEncoder(conn).Encode(&ControlMessage{
			Version: ControlMessageV1,
			InitResponse: &InitResponseMessage{
				PeerID: peerHandle.ID,
			},
		})
		if err != nil {
			return err
		}

		s.peersm.Lock()
		s.peers[peerHandle.ID] = peerHandle
		s.peersm.Unlock()

		defer func() {
			s.peersm.Lock()
			delete(s.peers, peerHandle.ID)
			s.peersm.Unlock()
		}()

		onPeerCtx, onPeerCancel := context.WithCancel(ctx)
		defer onPeerCancel()
		go handler.OnPeer(onPeerCtx, peerHandle)

		var g errgroup.Group
		g.Go(func() error {
			return s.runReader(ctx, conn)
		})
		g.Go(func() error {
			return s.runWriter(ctx, conn)
		})

		return g.Wait()
	case ConnTypeTun:
		peerId, reqId, err := ReadTunInit(conn)
		if err != nil {
			return err
		}

		s.peersm.Lock()
		h, ok := s.peers[peerId]
		s.peersm.Unlock()

		if !ok {
			return errors.New("unknown peer")
		}

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		if h.ctx != nil {
			go func() {
				select {
				case <-h.ctx.Done():
					cancel() // cancel all child tuns if the control tunnel goes down
				case <-ctx.Done():
				}
			}()
		}

		if reqId == 0 {
			return h.handler.ServeConn(ctx, conn)
		} else {
			reqh, ok := h.reqHandler[reqId]
			if !ok {
				return fmt.Errorf("unknown req id %d", reqId)
			}

			return reqh.ServeConn(ctx, conn)
		}
	default:
		return errors.New("invalid conn type")
	}
}

func (s *Server) runReader(ctx context.Context, controlConn io.ReadWriteCloser) error {
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
			//go s.handleConnRequest(ctx, msg.ConnRequest)
		default:
			return errors.New("invalid init request")
		}
	}
}

func (s *Server) runWriter(ctx context.Context, controlConn io.ReadWriteCloser) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case msg := <-s.openRequest:
			err := json.NewEncoder(controlConn).Encode(ControlMessage{
				Version: ControlMessageV1,
				ConnRequest: &ConnRequestMessage{
					RequestID: msg.reqId,
				},
			})
			msg.doneCh <- err
			if err != nil {
				return err
			}
		}
	}
}

func (s *Server) peerOpen(ctx context.Context, p *PeerDescriptor, handler tuntuntun.Handler) error {
	reqId := p.reqIdc.Add(1)

	p.reqHandler[reqId] = handler

	req := openRequest{
		reqId:  reqId,
		doneCh: make(chan error, 1),
	}

	select {
	case <-ctx.Done():
		return nil
	case s.openRequest <- req:
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-req.doneCh:
			return err
		}
	}
}
