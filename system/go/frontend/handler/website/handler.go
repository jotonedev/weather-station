package website

import (
	"go.uber.org/fx"
	"go.uber.org/zap"

	homepage "weather-station/frontend/controller/homepage"
)

type Handler struct {
	log                *zap.Logger
	homepageController *homepage.Controller
}

type HandlerParams struct {
	fx.In

	Logger             *zap.Logger
	HomepageController *homepage.Controller
}

func New(params HandlerParams) (*Handler, error) {
	return &Handler{
		log:                params.Logger,
		homepageController: params.HomepageController,
	}, nil
}
