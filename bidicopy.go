package tuntuntun

import (
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

func BidiCopy(remote, local io.ReadWriteCloser) {
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		_, err := io.Copy(remote, local)
		if err != nil {
			fmt.Println("bidi copy local->remote:", err)
		}
		closeWrite(remote)
	}()

	go func() {
		defer wg.Done()
		_, err := io.Copy(local, remote)
		if err != nil {
			fmt.Println("bidi copy remote->local:", err)
		}
		closeWrite(local)
	}()

	wg.Wait()
}
