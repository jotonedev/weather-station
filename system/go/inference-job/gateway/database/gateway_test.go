package database

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"google.golang.org/grpc/health/grpc_health_v1"

	protodatabasev1 "weather-station/gen/go/weather/database/v1"
	"weather-station/inference-job/internal/mocks"
)

func TestConfig(t *testing.T) {
	tests := []struct {
		name        string
		envKey      string
		envVal      string
		wantAddress string
	}{
		{
			name:        "default address when no env set",
			wantAddress: "ws-database:8090",
		},
		{
			name:        "database address with port",
			envKey:      "DATABASE_ADDRESS",
			envVal:      "custom-db:5432",
			wantAddress: "custom-db:5432",
		},
		{
			name:        "database address without port appends default port",
			envKey:      "DATABASE_ADDRESS",
			envVal:      "custom-db",
			wantAddress: "custom-db:8090",
		},
		{
			name:        "database grpc address env",
			envKey:      "DATABASE_GRPC_ADDRESS",
			envVal:      "grpc-db:9090",
			wantAddress: "grpc-db:9090",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envKey != "" {
				t.Setenv(tt.envKey, tt.envVal)
			}
			cfg := NewConfig()
			if cfg.Address != tt.wantAddress {
				t.Fatalf("Address = %q, want %q", cfg.Address, tt.wantAddress)
			}
		})
	}
}

func TestGatewayListNodeCoordinates(t *testing.T) {
	logger := zap.NewNop()
	errCall := errors.New("list nodes failed")

	tests := []struct {
		name      string
		setup     func(nodeClient *mocks.MockNodeDatabaseServiceClient)
		wantErr   bool
		wantCount int
	}{
		{
			name: "rpc error",
			setup: func(nodeClient *mocks.MockNodeDatabaseServiceClient) {
				nodeClient.EXPECT().
					ListNodeCoordinates(gomock.Any(), gomock.Any()).
					Return(nil, errCall)
			},
			wantErr: true,
		},
		{
			name: "success with nodes",
			setup: func(nodeClient *mocks.MockNodeDatabaseServiceClient) {
				nodeClient.EXPECT().
					ListNodeCoordinates(gomock.Any(), gomock.Any()).
					Return(&protodatabasev1.ListNodeCoordinatesResponse{
						Nodes: []*protodatabasev1.NodeCoordinates{
							{NodeId: "station-alpha"},
							{NodeId: "station-beta"},
						},
					}, nil)
			},
			wantErr:   false,
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			nodeClient := mocks.NewMockNodeDatabaseServiceClient(ctrl)
			tt.setup(nodeClient)

			gw, err := New(GatewayParams{
				Logger:           logger,
				AggregateClient:  mocks.NewMockReadingAggregateServiceClient(ctrl),
				PredictionClient: mocks.NewMockPredictionDatabaseServiceClient(ctrl),
				NodeClient:       nodeClient,
			})
			if err != nil {
				t.Fatalf("unexpected New error: %v", err)
			}

			nodes, err := gw.ListNodeCoordinates(context.Background())
			if (err != nil) != tt.wantErr {
				t.Fatalf("ListNodeCoordinates() error = %v, wantErr = %v", err, tt.wantErr)
			}
			if !tt.wantErr && len(nodes) != tt.wantCount {
				t.Fatalf("ListNodeCoordinates() count = %d, want %d", len(nodes), tt.wantCount)
			}
		})
	}
}

func TestGatewayGetClimateAggregates(t *testing.T) {
	logger := zap.NewNop()
	errCall := errors.New("query aggregates failed")

	tests := []struct {
		name      string
		nodeID    string
		setup     func(aggClient *mocks.MockReadingAggregateServiceClient)
		wantErr   bool
		wantCount int
	}{
		{
			name:    "empty node id returns error",
			nodeID:  "",
			setup:   func(aggClient *mocks.MockReadingAggregateServiceClient) {},
			wantErr: true,
		},
		{
			name:   "rpc error",
			nodeID: "node-1",
			setup: func(aggClient *mocks.MockReadingAggregateServiceClient) {
				aggClient.EXPECT().
					GetClimateAggregates(gomock.Any(), gomock.Any()).
					Return(nil, errCall)
			},
			wantErr: true,
		},
		{
			name:   "success returns aggregates",
			nodeID: "node-1",
			setup: func(aggClient *mocks.MockReadingAggregateServiceClient) {
				aggClient.EXPECT().
					GetClimateAggregates(gomock.Any(), gomock.Any()).
					Return(&protodatabasev1.GetClimateAggregatesResponse{
						Aggregates: []*protodatabasev1.ClimateAggregate{
							{NodeId: "node-1"},
						},
					}, nil)
			},
			wantErr:   false,
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			aggClient := mocks.NewMockReadingAggregateServiceClient(ctrl)
			tt.setup(aggClient)

			gw, err := New(GatewayParams{
				Logger:           logger,
				AggregateClient:  aggClient,
				PredictionClient: mocks.NewMockPredictionDatabaseServiceClient(ctrl),
				NodeClient:       mocks.NewMockNodeDatabaseServiceClient(ctrl),
			})
			if err != nil {
				t.Fatalf("unexpected New error: %v", err)
			}

			aggregates, err := gw.GetClimateAggregates(context.Background(), tt.nodeID, time.Now().Add(-24*time.Hour), time.Now(), 3600)
			if (err != nil) != tt.wantErr {
				t.Fatalf("GetClimateAggregates() error = %v, wantErr = %v", err, tt.wantErr)
			}
			if !tt.wantErr && len(aggregates) != tt.wantCount {
				t.Fatalf("GetClimateAggregates() count = %d, want %d", len(aggregates), tt.wantCount)
			}
		})
	}
}

func TestGatewayStorePredictions(t *testing.T) {
	logger := zap.NewNop()
	errCall := errors.New("store failed")

	tests := []struct {
		name        string
		predictions []*protodatabasev1.Prediction
		setup       func(predClient *mocks.MockPredictionDatabaseServiceClient)
		wantErr     bool
	}{
		{
			name:        "empty predictions slice succeeds without rpc call",
			predictions: nil,
			setup:       func(predClient *mocks.MockPredictionDatabaseServiceClient) {},
			wantErr:     false,
		},
		{
			name: "rpc error",
			predictions: []*protodatabasev1.Prediction{
				{NodeId: "node-1", HorizonHours: 1},
			},
			setup: func(predClient *mocks.MockPredictionDatabaseServiceClient) {
				predClient.EXPECT().
					StorePredictions(gomock.Any(), gomock.Any()).
					Return(nil, errCall)
			},
			wantErr: true,
		},
		{
			name: "success",
			predictions: []*protodatabasev1.Prediction{
				{NodeId: "node-1", HorizonHours: 1},
			},
			setup: func(predClient *mocks.MockPredictionDatabaseServiceClient) {
				predClient.EXPECT().
					StorePredictions(gomock.Any(), gomock.Any()).
					Return(&protodatabasev1.StorePredictionsResponse{Stored: 1}, nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			predClient := mocks.NewMockPredictionDatabaseServiceClient(ctrl)
			tt.setup(predClient)

			gw, err := New(GatewayParams{
				Logger:           logger,
				AggregateClient:  mocks.NewMockReadingAggregateServiceClient(ctrl),
				PredictionClient: predClient,
				NodeClient:       mocks.NewMockNodeDatabaseServiceClient(ctrl),
			})
			if err != nil {
				t.Fatalf("unexpected New error: %v", err)
			}

			err = gw.StorePredictions(context.Background(), tt.predictions)
			if (err != nil) != tt.wantErr {
				t.Fatalf("StorePredictions() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestGatewayCheck(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name    string
		setup   func(healthMock *mocks.MockHealthClient)
		withNil bool
		wantErr bool
	}{
		{
			name:    "nil health client returns no error",
			withNil: true,
			wantErr: false,
		},
		{
			name: "serving status returns no error",
			setup: func(healthMock *mocks.MockHealthClient) {
				healthMock.EXPECT().
					Check(gomock.Any(), gomock.Any()).
					Return(&grpc_health_v1.HealthCheckResponse{
						Status: grpc_health_v1.HealthCheckResponse_SERVING,
					}, nil)
			},
			wantErr: false,
		},
		{
			name: "not serving status returns error",
			setup: func(healthMock *mocks.MockHealthClient) {
				healthMock.EXPECT().
					Check(gomock.Any(), gomock.Any()).
					Return(&grpc_health_v1.HealthCheckResponse{
						Status: grpc_health_v1.HealthCheckResponse_NOT_SERVING,
					}, nil)
			},
			wantErr: true,
		},
		{
			name: "rpc error returns error",
			setup: func(healthMock *mocks.MockHealthClient) {
				healthMock.EXPECT().
					Check(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("rpc failed"))
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var healthClient grpc_health_v1.HealthClient
			if !tc.withNil {
				healthMock := mocks.NewMockHealthClient(ctrl)
				if tc.setup != nil {
					tc.setup(healthMock)
				}
				healthClient = healthMock
			}

			gateway := &Gateway{
				log:          zap.NewNop(),
				healthClient: healthClient,
			}

			if gateway.Name() != "database" {
				t.Errorf("expected name database, got %q", gateway.Name())
			}

			err := gateway.Check(context.Background())
			if tc.wantErr && err == nil {
				t.Fatal("expected error but got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
