package tuntunopener

import (
	"encoding/binary"
	"errors"
	"io"
)

type ConnType uint16

const (
	ConnTypeControl ConnType = 1
	ConnTypeTun     ConnType = 2
)

const ConnInitV1 = 1

const connInitHeaderSize = 100

func ReadConnInit(r io.Reader) (ConnType, error) {
	b := make([]byte, connInitHeaderSize)
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
	b := make([]byte, connInitHeaderSize)
	binary.LittleEndian.PutUint16(b[0:2], ConnInitV1)
	binary.LittleEndian.PutUint16(b[2:4], uint16(connType))

	_, err := r.Write(b)
	if err != nil {
		return err
	}

	return nil
}

const TunInitV1 = 1

const tunInitHeaderSize = 100

func ReadTunInit(r io.Reader) (uint64, uint64, error) {
	b := make([]byte, tunInitHeaderSize)
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
	b := make([]byte, tunInitHeaderSize)
	binary.LittleEndian.PutUint16(b[0:2], TunInitV1)
	binary.LittleEndian.PutUint64(b[2:10], peerId)
	binary.LittleEndian.PutUint64(b[10:18], reqId)

	_, err := r.Write(b)
	if err != nil {
		return err
	}

	return nil
}
