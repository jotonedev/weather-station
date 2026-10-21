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
)

type mockRunner struct {
	mu           sync.Mutex
	hourlyCalls  []int32
	dailyCalls   []int32
	failHourly   error
	failDaily    error
}

func newMockRunner() *mockRunner {
	return &mockRunner{}
}

func (m *mockRunner) RunHourlyForecast(ctx context.Context, horizonHours int32) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hourlyCalls = append(m.hourlyCalls, horizonHours)
	return m.failHourly
}

func (m *mockRunner) RunDailyForecast(ctx context.Context, horizonDays int32) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dailyCalls = append(m.dailyCalls, horizonDays)
	return m.failDaily
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
	if len(runner.hourlyCalls) != 1 || runner.hourlyCalls[0] != 24 {
		t.Fatalf("expected hourly forecast of 24h, got %v", runner.hourlyCalls)
	}
	if len(runner.dailyCalls) != 1 || runner.dailyCalls[0] != 10 {
		t.Fatalf("expected daily forecast of 10d, got %v", runner.dailyCalls)
	}
}

func TestRunInferenceJobFailure(t *testing.T) {
	runner := newMockRunner()
	runner.failHourly = errors.New("hourly failed")
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
