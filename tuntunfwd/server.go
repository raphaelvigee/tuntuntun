package tuntunfwd

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"tuntuntun"
	"tuntuntun/tuntunopener"
)

type Server struct {
	cfg    Config
	server *tuntunopener.Server
}

func runListener(ctx context.Context, cfg Config, h *tuntunopener.PeerDescriptor, raddr string, onListen func(ctx context.Context, raddr, laddr string)) {
	l, err := cfg.LocalListen(ctx, ":0")
	if err != nil {
		if cfg.Logger != nil {
			cfg.Logger.Log(ctx, slog.LevelError, "failed to listen", slog.String("err", err.Error()))
		}
		return
	}
	defer l.Close()

	if onListen != nil {
		go onListen(ctx, raddr, l.Addr().String())
	}

	for {
		lconn, err := l.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}

			if cfg.Logger != nil {
				cfg.Logger.Log(ctx, slog.LevelError, "failed to accept", slog.String("err", err.Error()))
			}
			return
		}

		go func() {
			err := h.Open(ctx, tuntuntun.HandlerFunc(func(ctx context.Context, rconn io.ReadWriteCloser) error {
				defer rconn.Close()
				defer lconn.Close()

				err := WriteInit(rconn, raddr)
				if err != nil {
					return err
				}

				return tuntuntun.BidiCopy(rconn, lconn)
			}))
			if err != nil {
				lconn.Close()
				if cfg.Logger != nil {
					cfg.Logger.Log(ctx, slog.LevelError, "failed to open", slog.String("err", err.Error()))
				}
			}
		}()
	}
}

func DefaultPeerHandler(cfg Config, autoForward []string, onListen func(ctx context.Context, raddr, laddr string)) tuntunopener.PeerHandler {
	return tuntunopener.PeerHandlerFunc{
		OnPeerFunc: func(ctx context.Context, h *tuntunopener.PeerDescriptor) {
			for _, addr := range autoForward {
				go runListener(ctx, cfg, h, addr, onListen)
			}
		},
		ServeConnFunc: func(ctx context.Context, rconn io.ReadWriteCloser) error {
			defer rconn.Close()

			msg, err := ReadInit(rconn)
			if err != nil {
				return err
			}

			lconn, err := cfg.LocalDial(ctx, msg.Addr)
			if err != nil {
				return err
			}
			defer lconn.Close()

			return tuntuntun.BidiCopy(rconn, lconn)
		},
	}
}

func NewServer(factory func() (tuntunopener.PeerHandler, error)) *Server {
	return &Server{
		server: tuntunopener.NewServer(factory),
	}
}

func (c *Server) ServeConn(ctx context.Context, conn io.ReadWriteCloser) error {
	return c.server.ServeConn(ctx, conn)
}
