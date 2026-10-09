package logging

import (
	"os"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

// Module provides the application logger and routes fx events through it.
var Module = fx.Module("logging",
	fx.Provide(New),
	fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
		return &fxevent.ZapLogger{Logger: log}
	}),
)

// New returns a production logger when GO_ENV is "production", otherwise a development logger.
func New() (*zap.Logger, error) {
	if os.Getenv("GO_ENV") == "production" {
		return zap.NewProduction()
	}
	return zap.NewDevelopment()
}
