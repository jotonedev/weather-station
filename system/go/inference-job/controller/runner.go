package controller

import (
	"context"

	"go.uber.org/fx"
	"go.uber.org/zap"

	"weather-station/inference-job/controller/inference"
)

// ForecastRunner abstracts running forecasts for weather predictions.
type ForecastRunner interface {
	RunHourlyForecast(ctx context.Context, horizonHours int32) error
	RunDailyForecast(ctx context.Context, horizonDays int32) error
}

// RunnerParams defines dependencies required to execute the inference job.
type RunnerParams struct {
	fx.In

	Lifecycle  fx.Lifecycle
	Shutdowner fx.Shutdowner
	Controller *inference.Controller
	Logger     *zap.Logger
}

// RunInferenceJob registers a lifecycle hook that executes forecasts and shuts down the application.
func RunInferenceJob(params RunnerParams) {
	runInferenceJob(params.Lifecycle, params.Shutdowner, params.Controller, params.Logger)
}

func runInferenceJob(lc fx.Lifecycle, shutdowner fx.Shutdowner, runner ForecastRunner, log *zap.Logger) {
	ctx, cancel := context.WithCancel(context.Background())

	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			go func() {
				defer cancel()
				var exitCode int
				log.Info("Starting weather inference job forecast execution")

				if err := runner.RunHourlyForecast(ctx, 24); err != nil {
					log.Error("24-hour weather forecast failed", zap.Error(err))
					exitCode = 1
				}

				if err := runner.RunDailyForecast(ctx, 10); err != nil {
					log.Error("10-day weather forecast failed", zap.Error(err))
					exitCode = 1
				}

				log.Info("Weather inference job execution finished", zap.Int("exit_code", exitCode))
				if err := shutdowner.Shutdown(fx.ExitCode(exitCode)); err != nil {
					log.Error("Failed to trigger application shutdown", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(_ context.Context) error {
			cancel()
			return nil
		},
	})
}
