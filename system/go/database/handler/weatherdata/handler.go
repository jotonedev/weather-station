package weatherdata

import (
	"go.uber.org/fx"
	"go.uber.org/zap"

	controller "weather-station/database/controller/weatherdata"
)

type Handler struct {
	log *zap.Logger
	con *controller.Controller
}

type HandlerParams struct {
	fx.In

	Logger     *zap.Logger
	Controller *controller.Controller
}

func New(params HandlerParams) (*Handler, error) {
	return &Handler{
		log: params.Logger,
		con: params.Controller,
	}, nil
}
