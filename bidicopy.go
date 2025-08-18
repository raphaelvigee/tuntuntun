package tuntuntun

import (
	"errors"
	"fmt"
	"io"
	"sync"
)

func closeWrite(c io.ReadWriteCloser) {
	type closeWriter interface {
		CloseWrite() error
	}
	if cw, ok := c.(closeWriter); ok {
		_ = cw.CloseWrite() // half-close
	} else {
		_ = c.Close() // fallback to full close
	}
}

func BidiCopy(remote, local io.ReadWriteCloser) error {
	var errs [2]error
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		_, err := io.Copy(remote, local)
		if err != nil {
			errs[0] = fmt.Errorf("bidi copy local->remote: %w", err)
		}
		closeWrite(remote)
	}()

	go func() {
		defer wg.Done()
		_, err := io.Copy(local, remote)
		if err != nil {
			errs[1] = fmt.Errorf("bidi copy remote->local: %w", err)
		}
		closeWrite(local)
	}()

	wg.Wait()

	return errors.Join(errs[:]...)
}
