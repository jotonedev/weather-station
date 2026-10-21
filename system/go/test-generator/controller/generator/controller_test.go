package generator

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"weather-station/test-generator/gateway/ingest"
	"weather-station/test-generator/internal/mocks"
)

func TestNewConfig(t *testing.T) {
	tests := []struct {
		name         string
		envInterval  string
		envNodes     string
		wantInterval time.Duration
		wantNodes    int
		wantErr      bool
	}{
		{
			name:         "defaults when env unset",
			wantInterval: defaultInterval,
			wantNodes:    1,
		},
		{
			name:         "custom interval and multiple nodes",
			envInterval:  "5s",
			envNodes:     "station-alpha, station-beta",
			wantInterval: 5 * time.Second,
			wantNodes:    2,
		},
		{
			name:        "invalid duration returns error",
			envInterval: "not-a-duration",
			wantErr:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("GENERATOR_INTERVAL", tc.envInterval)
			t.Setenv("GENERATOR_NODE_IDS", tc.envNodes)

			cfg, err := NewConfig()
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.Interval != tc.wantInterval {
				t.Fatalf("Interval = %v, want %v", cfg.Interval, tc.wantInterval)
			}
			if len(cfg.NodeIDs) != tc.wantNodes {
				t.Fatalf("len(NodeIDs) = %d, want %d", len(cfg.NodeIDs), tc.wantNodes)
			}
		})
	}
}

func TestControllerGenerateAndSend(t *testing.T) {
	errIngestDown := errors.New("ingest unreachable")

	tests := []struct {
		name    string
		nodes   []string
		setup   func(gw *mocks.MockIngestGateway)
		wantErr error
	}{
		{
			name:  "generates and sends reading for each node",
			nodes: []string{"station-alpha", "station-beta"},
			setup: func(gw *mocks.MockIngestGateway) {
				gw.EXPECT().
					SendReadings(gomock.Any(), gomock.AssignableToTypeOf([]ingest.ReadingPayload{})).
					DoAndReturn(func(_ context.Context, readings []ingest.ReadingPayload) error {
						if len(readings) != 2 {
							t.Fatalf("len(readings) = %d, want 2", len(readings))
						}
						return nil
					})
			},
		},
		{
			name:    "empty node list skips gateway call",
			nodes:   nil,
			setup:   func(gw *mocks.MockIngestGateway) {},
			wantErr: nil,
		},
		{
			name:  "gateway error is propagated",
			nodes: []string{"station-alpha"},
			setup: func(gw *mocks.MockIngestGateway) {
				gw.EXPECT().
					SendReadings(gomock.Any(), gomock.Any()).
					Return(errIngestDown)
			},
			wantErr: errIngestDown,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockGW := mocks.NewMockIngestGateway(ctrl)
			tc.setup(mockGW)

			controller, err := New(ControllerParams{
				Logger:        zap.NewNop(),
				Config:        Config{Interval: time.Second, NodeIDs: tc.nodes},
				CustomGateway: mockGW,
			})
			if err != nil {
				t.Fatalf("New() unexpected error = %v", err)
			}

			err = controller.GenerateAndSend(context.Background())
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("GenerateAndSend() error = %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("GenerateAndSend() unexpected error = %v", err)
			}
		})
	}
}
