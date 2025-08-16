package tuntunfwd

import (
	"context"
	"encoding/json"
	"net"
	"tuntuntun"
)

func RemoteDialContext(ctx context.Context, opener tuntuntun.Opener, remoteAddr string) (net.Conn, error) {
	rconn, err := opener.Open(ctx)
	if err != nil {
		return nil, err
	}

	b, err := json.Marshal(Init{
		Version: V1,
		Mode:    ServerForward,
		Addr:    remoteAddr,
	})
	if err != nil {
		return nil, err
	}

	_, err = rconn.Write(b)
	if err != nil {
		return nil, err
	}

	return rconn, nil
}

func LocalForward(ctx context.Context, opener tuntuntun.Opener, localAddr string) error {
	l, err := net.Dial("tcp", localAddr)
	if err != nil {
		return err
	}
	defer l.Close()

	rconn, err := opener.Open(ctx)
	if err != nil {
		return err
	}
	defer rconn.Close()

	b, err := json.Marshal(Init{
		Version: V1,
		Mode:    ClientForward,
	})
	if err != nil {
		return err
	}

	_, err = rconn.Write(b)
	if err != nil {
		return err
	}

	tuntuntun.BidiCopy(rconn, l)

	return nil
}
