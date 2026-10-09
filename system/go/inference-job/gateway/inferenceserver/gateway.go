package inferenceserver

import (
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Gateway struct {
	log *zap.Logger
}

type GatewayParams struct {
	fx.In

	Logger *zap.Logger
}

func New(params GatewayParams) (*Gateway, error) {
	return &Gateway{
		log: params.Logger,
	}, nil
}
