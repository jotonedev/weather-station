package controller

import (
	"context"
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"

	"weather-station/test-generator/controller/generator"
)

// BatchSender abstracts periodic generation and transmission of synthetic readings.
type BatchSender interface {
	Interval() time.Duration
	GenerateAndSend(ctx context.Context) error
}

// RunnerParams defines dependencies required to run the continuous test generator loop.
type RunnerParams struct {
	fx.In

	Lifecycle  fx.Lifecycle
	Controller *generator.Controller
	Logger     *zap.Logger
}

// RunGenerator registers a lifecycle hook that streams synthetic readings until stopped.
func RunGenerator(params RunnerParams) {
	runGenerator(params.Lifecycle, params.Controller, params.Logger)
}

func runGenerator(lc fx.Lifecycle, sender BatchSender, log *zap.Logger) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			interval := sender.Interval()
			log.Info("Starting test data generator loop", zap.Duration("interval", interval))

			go func() {
				defer close(done)

				if err := sender.GenerateAndSend(ctx); err != nil && ctx.Err() == nil {
					log.Warn("Initial test data generation failed", zap.Error(err))
				}

				ticker := time.NewTicker(interval)
				defer ticker.Stop()

				for {
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
						if err := sender.GenerateAndSend(ctx); err != nil && ctx.Err() == nil {
							log.Warn("Periodic test data generation failed", zap.Error(err))
						}
					}
				}
			}()
			return nil
		},
		OnStop: func(stopCtx context.Context) error {
			cancel()
			select {
			case <-done:
			case <-stopCtx.Done():
				return stopCtx.Err()
			}
			return nil
		},
	})
}
