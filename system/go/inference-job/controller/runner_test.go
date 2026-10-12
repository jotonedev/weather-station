package controller

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"

	protoinferencev1 "weather-station/gen/go/weather/inference/v1"
)

type mockRunner struct {
	mu        sync.Mutex
	modes     []protoinferencev1.PredictionMode
	failModes map[protoinferencev1.PredictionMode]error
}

func newMockRunner() *mockRunner {
	return &mockRunner{
		failModes: make(map[protoinferencev1.PredictionMode]error),
	}
}

func (m *mockRunner) RunForecast(ctx context.Context, mode protoinferencev1.PredictionMode) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.modes = append(m.modes, mode)
	return m.failModes[mode]
}

func TestRunInferenceJobSuccess(t *testing.T) {
	runner := newMockRunner()
	logger := zap.NewNop()

	app := fxtest.New(
		t,
		fx.Provide(func() ForecastRunner { return runner }),
		fx.Provide(func() *zap.Logger { return logger }),
		fx.Invoke(func(lc fx.Lifecycle, s fx.Shutdowner, r ForecastRunner, l *zap.Logger) {
			runInferenceJob(lc, s, r, l)
		}),
	)

	app.RequireStart()
	defer app.RequireStop()

	select {
	case wait := <-app.Wait():
		if wait.ExitCode != 0 {
			t.Fatalf("expected exit code 0, got %d", wait.ExitCode)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for app shutdown")
	}

	runner.mu.Lock()
	defer runner.mu.Unlock()
	if len(runner.modes) != 2 {
		t.Fatalf("expected 2 forecasts executed, got %d", len(runner.modes))
	}
	if runner.modes[0] != protoinferencev1.PredictionMode_PREDICTION_MODE_NEXT_24_HOURS {
		t.Errorf("expected first mode to be 24h, got %v", runner.modes[0])
	}
	if runner.modes[1] != protoinferencev1.PredictionMode_PREDICTION_MODE_NEXT_10_DAYS {
		t.Errorf("expected second mode to be 10d, got %v", runner.modes[1])
	}
}

func TestRunInferenceJobFailure(t *testing.T) {
	runner := newMockRunner()
	runner.failModes[protoinferencev1.PredictionMode_PREDICTION_MODE_NEXT_24_HOURS] = errors.New("failed")
	logger := zap.NewNop()

	app := fxtest.New(
		t,
		fx.Provide(func() ForecastRunner { return runner }),
		fx.Provide(func() *zap.Logger { return logger }),
		fx.Invoke(func(lc fx.Lifecycle, s fx.Shutdowner, r ForecastRunner, l *zap.Logger) {
			runInferenceJob(lc, s, r, l)
		}),
	)

	app.RequireStart()
	defer app.RequireStop()

	select {
	case wait := <-app.Wait():
		if wait.ExitCode != 1 {
			t.Fatalf("expected exit code 1, got %d", wait.ExitCode)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for app shutdown")
	}
}
