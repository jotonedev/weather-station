package health

import (
	"context"

	"go.uber.org/fx"
)

// Checker verifies the readiness of a component or dependency.
type Checker interface {
	Name() string
	Check(ctx context.Context) error
}

// AsChecker annotates a provider as a readiness checker for the shared health module.
func AsChecker(target any) any {
	return fx.Annotate(
		target,
		fx.As(new(Checker)),
		fx.ResultTags(`group:"readiness-checkers"`),
	)
}
