package tuntunh2

import (
	"errors"
	"net/http"
)

type Upgrader struct {
	StatusCode int
}

var ErrHTTP2NotSupported = errors.New("http2 not supported")

// Accept is used on a server http.Handler to extract a full-duplex communication object with the client.
func (u *Upgrader) Accept(w http.ResponseWriter, r *http.Request) (*Conn, error) {
	if !r.ProtoAtLeast(2, 0) {
		return nil, ErrHTTP2NotSupported
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, ErrHTTP2NotSupported
	}

	c, ctx := newConn(r.Context(), r.Body, &flushWriter{w: w, f: flusher})

	// Update the request context with the connection context.
	// If the connection is closed by the server, it will also notify everything that waits on the request context.
	*r = *r.WithContext(ctx)

	w.WriteHeader(u.StatusCode)
	flusher.Flush()

	return c, nil
}
