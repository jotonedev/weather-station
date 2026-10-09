package grpc

import (
	"context"
	"net"

	"go.uber.org/fx"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

//goland:noinspection GoSnakeCaseUsage
const NET_PROTOCOL = "tcp"

func NewGRPCServer(lc fx.Lifecycle, log *zap.Logger, cfg Config, registrars []Registrar) *grpc.Server {
	srv := grpc.NewServer()

	for _, registrar := range registrars {
		registrar.Register(srv)
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ln, err := net.Listen(NET_PROTOCOL, cfg.Address)
			if err != nil {
				return err
			}
			log.Info("Starting gRPC server", zap.String("addr", cfg.Address))

			go func() {
				if err := srv.Serve(ln); err != nil {
					log.Error("gRPC server failed", zap.Error(err))
				}
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			srv.GracefulStop()
			return nil
		},
	})

	return srv
}
