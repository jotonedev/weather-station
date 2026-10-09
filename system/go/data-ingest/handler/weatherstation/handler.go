package weatherstation

import (
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Handler struct {
	log *zap.Logger
}

type HandlerParams struct {
	fx.In

	Logger *zap.Logger
}

func New(params HandlerParams) (*Handler, error) {
	return &Handler{
		log: params.Logger,
	}, nil
}
