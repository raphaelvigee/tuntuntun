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
	"net/url"
	"os"
	"strings"
	"tuntuntun"
	"tuntuntun/tuntunfwd"
	"tuntuntun/tuntunh2"
	"tuntuntun/tuntunhttp"
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
		name := flag.String("name", strings.Join(args, " "), "connection name")
		mux := flag.Bool("mux", true, "enable mux")
		flag.CommandLine.Parse(args[1:])

		u, err := url.Parse(*addr)
		if err != nil {
			log.Fatal(err)
		}

		q := u.Query()
		q.Set("name", *name)
		u.RawQuery = q.Encode()

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

				l, err := net.Listen("tcp4", ":0")
				if err != nil {
					return err
				}

				req := tuntunhttp.RequestFromContext(ctx)

				name := req.URL.Query().Get("name")

				fmt.Printf("[%v] Listening on %v\n", name, l.Addr())

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
