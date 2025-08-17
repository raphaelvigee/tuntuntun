package tuntunfwd

import (
	"encoding/json"
	"fmt"
	"io"
)

const V1 = 1

type Message struct {
	Version int    `json:"version"`
	Addr    string `json:"addr"`
}

func WriteInit(conn io.Writer, addr string) error {
	return json.NewEncoder(conn).Encode(Message{
		Version: V1,
		Addr:    addr,
	})
}

func ReadInit(conn io.Reader) (Message, error) {
	var msg Message
	err := json.NewDecoder(conn).Decode(&msg)
	if err != nil {
		return msg, err
	}

	if msg.Version != V1 {
		return msg, fmt.Errorf("unexpected version: %d", msg.Version)
	}

	if msg.Addr == "" {
		return msg, fmt.Errorf("missing addr")
	}

	return msg, err
}
