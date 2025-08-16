package tuntuntun

import (
	"fmt"
	"io"
	"sync"
)

func BidiCopy(remote, local io.ReadWriteCloser) {
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		_, err := io.Copy(remote, local)
		if err != nil {
			fmt.Println("bidi copy local->remote:", err)
		}
		remote.Close()
	}()

	go func() {
		defer wg.Done()
		_, err := io.Copy(local, remote)
		if err != nil {
			fmt.Println("bidi copy remote->local:", err)
		}
		local.Close()
	}()

	wg.Wait()
}
