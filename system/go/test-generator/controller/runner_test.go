package controller

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/fx/fxtest"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"weather-station/test-generator/internal/mocks"
)

func TestRunGeneratorLifecycle(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSender := mocks.NewMockBatchSender(ctrl)

	var callCount atomic.Int32
	mockSender.EXPECT().
		Interval().
		Return(15 * time.Millisecond)

	mockSender.EXPECT().
		GenerateAndSend(gomock.Any()).
		DoAndReturn(func(_ context.Context) error {
			callCount.Add(1)
			return nil
		}).
		MinTimes(2)

	lc := fxtest.NewLifecycle(t)
	runGenerator(lc, mockSender, zap.NewNop())

	if err := lc.Start(context.Background()); err != nil {
		t.Fatalf("lc.Start() error = %v", err)
	}

	deadline := time.Now().Add(200 * time.Millisecond)
	for callCount.Load() < 2 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	if err := lc.Stop(context.Background()); err != nil {
		t.Fatalf("lc.Stop() error = %v", err)
	}
}
