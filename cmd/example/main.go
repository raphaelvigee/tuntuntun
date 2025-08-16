package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"tuntuntun/tuntunfwd"
	"tuntuntun/tuntunh2"
	"tuntuntun/tuntunmux"
)

func main() {
	//ctx := context.Background()

	switch os.Args[1] {
	case "client":
		addr := flag.String("addr", "https://localhost:1234", "http address")
		flag.Parse()

		ttc := tuntunh2.NewClient(*addr)

		ttmux := tuntunmux.NewClient(ttc)
		defer ttmux.Close()

		err := tuntunfwd.Forward(ttmux, ":2222", ":22")
		if err != nil {
			log.Fatal(err)
		}
	case "server":
		addr := flag.String("addr", ":1234", "http server address")
		flag.Parse()

		ttfwd := tuntunfwd.NewServer()
		ttmux := tuntunmux.NewServer(ttfwd)
		tts := tuntunh2.NewServer(ttmux)

		err := http.ListenAndServe(*addr, tts)
		if err != nil {
			log.Fatal(err)
		}
	}
}
