package http

import (
	"net/http"

	"go.uber.org/fx"
)

var Module = fx.Module("http",
	fx.Provide(
		NewHTTPServer,
		fx.Annotate(
			NewHTTPMux,
			fx.ParamTags(`group:"http-routes"`),
		),
	),
	fx.Invoke(func(*http.Server) {}),
)
