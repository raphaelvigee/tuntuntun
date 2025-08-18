package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"tuntuntun"
	"tuntuntun/tuntunfwd"
	"tuntuntun/tuntunh2"
	"tuntuntun/tuntunhttp"
	"tuntuntun/tuntunmux"
	"tuntuntun/tuntunopener"
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
		transport := flag.String("transport", "ws", "http transport [ws, h2]")
		mux := flag.Bool("mux", true, "enable mux")
		flag.CommandLine.Parse(args[1:])

		u, err := url.Parse(*addr)
		if err != nil {
			log.Fatal(err)
		}

		var opener tuntuntun.Opener
		switch *transport {
		case "h2":
			opener = tuntunh2.NewClient(u.String())
		case "ws":
			opener = tuntunws.NewClient(u.String())
		default:
			log.Fatal(fmt.Sprintf("unknown transport %q", *transport))
		}

		if *mux {
			ttmux := tuntunmux.NewClient(opener)
			defer ttmux.Close()

			opener = ttmux
		}

		cfg := tuntunfwd.Config{
			LocalDial: func(ctx context.Context, addr string) (net.Conn, error) {
				return net.Dial("tcp4", addr)
			},
			LocalListen: func(ctx context.Context, addr string) (net.Listener, error) {
				return net.Listen("tcp4", addr)
			},
		}

		client := tuntunfwd.NewClient(
			cfg,
			opener,
			tuntunfwd.DefaultPeerHandler(
				cfg,
				nil,
				func(ctx context.Context, raddr, laddr string) {
					panic("should not happen")
				},
			),
		)

		doneCh, err := client.Start(ctx)
		if err != nil {
			log.Fatal(err)
		}

		err = <-doneCh
		if err != nil {
			log.Fatal(err)
		}
	case "server":
		addr := flag.String("addr", ":1234", "http server address")
		allowForward := flag.Bool("allow-forward", false, "allow forwarding request")
		remoteAddrs := flag.String("remote-addrs", "", "comma-separated addresses to request forwarding")
		transport := flag.String("transport", "ws", "http transport [ws, h2]")
		mux := flag.Bool("mux", true, "enable mux")
		flag.CommandLine.Parse(args[1:])

		var handler tuntuntun.Handler = tuntunfwd.NewServer(func() (tuntunopener.PeerHandler, error) {
			return tuntunfwd.DefaultPeerHandler(
				tuntunfwd.Config{
					LocalDial: func(ctx context.Context, addr string) (net.Conn, error) {
						if *allowForward {
							return net.Dial("tcp4", addr)
						} else {
							return nil, errors.New("denied by cli")
						}

					},
					LocalListen: func(ctx context.Context, addr string) (net.Listener, error) {
						return net.Listen("tcp4", addr)
					},
				},
				strings.Split(*remoteAddrs, ","),
				func(ctx context.Context, raddr, laddr string) {
					fmt.Printf("[%v] Listening on %v\n", raddr, laddr)
				},
			), nil
		})

		if *mux {
			handler = tuntunmux.NewServer(handler)
		}

		var httpHandler http.Handler
		switch *transport {
		case "h2":
			httpHandler = tuntunh2.NewServer(handler)
		case "ws":
			httpHandler = tuntunws.NewServer(handler)
		default:
			log.Fatal(fmt.Sprintf("unknown transport %q", *transport))
		}

		httpHandler = tuntunhttp.Middleware(httpHandler)

		if *transport == "h2" {
			h2s := &http2.Server{
				MaxConcurrentStreams: 250,
			}

			httpHandler = h2c.NewHandler(httpHandler, h2s)
		}

		err := http.ListenAndServe(*addr, httpHandler)
		if err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatal(fmt.Sprintf("unknown command %q", os.Args[1]))
	}
}
