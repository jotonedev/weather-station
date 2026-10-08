package http

import (
	"context"
	"net"
	"net/http"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

//goland:noinspection GoSnakeCaseUsage
const (
	NET_PROTOCOL = "tcp"
	NET_ADDRESS  = "0.0.0.0:8080"
)

func NewHTTPServer(lc fx.Lifecycle, mux *http.ServeMux, log *zap.Logger) *http.Server {
	srv := &http.Server{Addr: NET_ADDRESS, Handler: mux}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ln, err := net.Listen(NET_PROTOCOL, srv.Addr)
			if err != nil {
				return err
			}
			log.Info("Starting HTTP server at", zap.String("addr", srv.Addr))

			go func() {
				err := srv.Serve(ln)
				if err != nil {
					panic(err)
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
