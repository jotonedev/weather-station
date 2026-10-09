package http

import (
	"context"
	"errors"
	"net"
	"net/http"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

//goland:noinspection GoSnakeCaseUsage
const NET_PROTOCOL = "tcp"

func NewHTTPServer(lc fx.Lifecycle, mux *http.ServeMux, log *zap.Logger, cfg Config) *http.Server {
	srv := &http.Server{Addr: cfg.Address, Handler: mux}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ln, err := net.Listen(NET_PROTOCOL, srv.Addr)
			if err != nil {
				return err
			}
			log.Info("Starting HTTP server", zap.String("addr", srv.Addr))

			go func() {
				if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
					log.Error("HTTP server failed", zap.Error(err))
				}
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})

	return srv
}
