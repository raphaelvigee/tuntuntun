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

type Handler interface {
	ServeConn(ctx context.Context, conn io.ReadWriteCloser) error
}

type HandlerFunc func(context.Context, io.ReadWriteCloser) error

func (h HandlerFunc) ServeConn(ctx context.Context, conn io.ReadWriteCloser) error {
	return h(ctx, conn)
}

type Server struct {
	handlerFactory func() Handler
	peersm         sync.Mutex
	peers          map[uint64]*PeerHandle
	peerIdc        atomic.Uint64

	openRequest chan openRequest

	onPeer func(ctx context.Context, h *PeerHandle)
}

type openRequest struct {
	reqId uint64
}

type PeerHandle struct {
	s *Server

	id      uint64
	handler Handler

	reqIdc     atomic.Uint64
	reqHandler map[uint64]Handler
}

func (p *PeerHandle) Open(ctx context.Context, handler Handler) error {
	reqId := p.reqIdc.Add(1)

	p.reqHandler[reqId] = handler

	select {
	case <-ctx.Done():
		return ctx.Err()
	case p.s.openRequest <- openRequest{reqId: reqId}:
		return nil
	}
}

func NewServer(handler func() Handler, onPeer func(ctx context.Context, h *PeerHandle)) *Server {
	return &Server{
		handlerFactory: handler,
		onPeer:         onPeer,
		peers:          map[uint64]*PeerHandle{},
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

		peerId := s.peerIdc.Add(1)
		handler := s.handlerFactory()

		peerHandle := &PeerHandle{
			s:          s,
			id:         peerId,
			handler:    handler,
			reqHandler: make(map[uint64]Handler),
		}

		s.peersm.Lock()
		s.peers[peerId] = peerHandle
		s.peersm.Unlock()

		err = json.NewEncoder(conn).Encode(&ControlMessage{
			Version: ControlMessageV1,
			InitResponse: &InitResponseMessage{
				PeerID: peerId,
			},
		})
		if err != nil {
			return err
		}

		defer func() {
			s.peersm.Lock()
			delete(s.peers, peerId)
			s.peersm.Unlock()
		}()

		if s.onPeer != nil {
			go s.onPeer(ctx, peerHandle)
		}

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
			return ctx.Err()
		case msg := <-s.openRequest:
			err := json.NewEncoder(controlConn).Encode(ControlMessage{
				Version: ControlMessageV1,
				ConnRequest: &ConnRequestMessage{
					RequestID: msg.reqId,
				},
			})
			if err != nil {
				return err
			}
		}
	}
}
