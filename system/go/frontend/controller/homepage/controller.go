package homepage

import (
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Controller struct {
	log *zap.Logger
}

type ControllerParams struct {
	fx.In

	Logger *zap.Logger
}

func New(params ControllerParams) (*Controller, error) {
	return &Controller{
		log: params.Logger,
	}, nil
}
