package tuntunh2

import (
	"context"
	"net/http"
)

type responseWriterCloser struct {
	http.ResponseWriter
	f     http.Flusher
	close context.CancelFunc
}

func (w *responseWriterCloser) Write(data []byte) (int, error) {
	n, err := w.ResponseWriter.Write(data)
	w.f.Flush()

	return n, err
}

func (w *responseWriterCloser) Close() error {
	// Currently server side close of connection is not supported in Go.
	// The server closes the connection when the http.Handler function returns.
	// We use connection context and cancel function as a work-around.
	w.close()

	return nil
}
