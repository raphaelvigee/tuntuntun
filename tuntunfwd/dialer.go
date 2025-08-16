package tuntunfwd

import (
	"context"
	"encoding/json"
	"net"
	"tuntuntun"
)

func DialContext(ctx context.Context, opener tuntuntun.Opener, remoteAddr string) (net.Conn, error) {
	rconn, err := opener.Open(ctx)
	if err != nil {
		return nil, err
	}

	b, err := json.Marshal(Init{Addr: remoteAddr})
	if err != nil {
		return nil, err
	}

	_, err = rconn.Write(b)
	if err != nil {
		return nil, err
	}

	return rconn, nil
}
