package e2e

import (
	"net/http"
	"tuntuntun"
	"tuntuntun/tuntunh2"
	"tuntuntun/tuntunws"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

type Setup struct {
	Opener func(srvUrl string) tuntuntun.Opener
	Server func(handler tuntuntun.Handler) http.Handler
}

var setup = map[string]Setup{
	"h2": {
		Server: func(handler tuntuntun.Handler) http.Handler {
			h2s := &http2.Server{
				MaxConcurrentStreams: 250,
			}

			return h2c.NewHandler(tuntunh2.NewServer(handler), h2s)
		},
		Opener: func(srvUrl string) tuntuntun.Opener {
			return tuntunh2.NewClient(srvUrl)
		},
	},
	"ws": {
		Server: func(handler tuntuntun.Handler) http.Handler {
			return tuntunws.NewServer(handler)
		},
		Opener: func(srvUrl string) tuntuntun.Opener {
			return tuntunws.NewClient(srvUrl)
		},
	},
}
