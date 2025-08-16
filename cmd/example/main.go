package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"tuntuntun"
	"tuntuntun/tuntunfwd"
	"tuntuntun/tuntunh2"
	"tuntuntun/tuntunmux"
	"tuntuntun/tuntunws"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func main() {
	ctx := context.Background()

	args := os.Args[1:]
	for len(args) >= 0 && args[0] == "--" {
		args = args[1:]
	}

	switch args[0] {
	case "client":
		addr := flag.String("addr", "https://localhost:1234", "server address")
		transport := flag.String("transport", "h2", "http transport [h2, ws]")
		remoteAddr := flag.String("remote-addr", "", "remote address to bind")
		localAddr := flag.String("local-addr", "", "local address to bind")
		mux := flag.Bool("mux", true, "enable mux")
		flag.CommandLine.Parse(args[1:])

		var opener tuntuntun.Opener
		switch *transport {
		case "h2":
			opener = tuntunh2.NewClient(*addr)
		case "ws":
			opener = tuntunws.NewClient(*addr)
		default:
			log.Fatal(fmt.Sprintf("unknown transport %q", *transport))
		}

		if *mux {
			ttmux := tuntunmux.NewClient(opener)
			defer ttmux.Close()

			opener = ttmux
		}

		if *remoteAddr != "" {
			err := tuntunfwd.RemoteForward(opener, *remoteAddr, *localAddr)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			err := tuntunfwd.LocalForward(ctx, opener, *localAddr)
			if err != nil {
				log.Fatal(err)
			}
		}
	case "server":
		addr := flag.String("addr", ":1234", "http server address")
		allowForward := flag.Bool("allow-forward", false, "allow forwarding request")
		transport := flag.String("transport", "h2", "http transport [h2, ws]")
		mux := flag.Bool("mux", true, "enable mux")
		flag.CommandLine.Parse(args[1:])

		var handler tuntuntun.Handler = tuntunfwd.NewServer(tuntunfwd.ServerConfig{
			AllowServerForward: func(ctx context.Context, addr string) error {
				if *allowForward {
					return nil // allow all
				} else {
					return errors.New("denied by cli")
				}
			},
			OnClientForward: func(ctx context.Context, conn io.ReadWriteCloser) error {
				defer conn.Close()

				fmt.Println("Connected...")

				l, err := net.Listen("tcp4", ":0")
				if err != nil {
					return err
				}

				fmt.Println("Listening on ", l.Addr())

				lconn, err := l.Accept()
				if err != nil {
					return err
				}

				tuntuntun.BidiCopy(conn, lconn)

				return err
			},
		})

		if *mux {
			handler = tuntunmux.NewServer(handler)
		}

		var httpHandler http.Handler
		switch *transport {
		case "h2":
			h2s := &http2.Server{
				MaxConcurrentStreams: 250,
			}

			httpHandler = h2c.NewHandler(tuntunh2.NewServer(handler), h2s)
		case "ws":
			httpHandler = tuntunws.NewServer(handler)
		default:
			log.Fatal(fmt.Sprintf("unknown transport %q", *transport))
		}

		err := http.ListenAndServe(*addr, httpHandler)
		if err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatal(fmt.Sprintf("unknown command %q", os.Args[1]))
	}
}
