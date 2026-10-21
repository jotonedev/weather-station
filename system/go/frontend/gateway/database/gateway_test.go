package database

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"google.golang.org/grpc/health/grpc_health_v1"

	"weather-station/frontend/internal/mocks"
	protodatabasev1 "weather-station/gen/go/weather/database/v1"
)

func TestGatewayGetLatestReadings(t *testing.T) {
	errDatabaseFailed := errors.New("readings unavailable")

	testCases := []struct {
		name      string
		nodeID    string
		setup     func(client *mocks.MockReadingDatabaseServiceClient)
		wantErr   bool
		wantCount int
	}{
		{
			name:   "success returning latest readings",
			nodeID: "node-1",
			setup: func(client *mocks.MockReadingDatabaseServiceClient) {
				client.EXPECT().
					GetLatestReadings(gomock.Any(), gomock.Any()).
					Return(&protodatabasev1.GetLatestReadingsResponse{
						Readings: []*protodatabasev1.Reading{
							{NodeId: "node-1"},
						},
					}, nil)
			},
			wantErr:   false,
			wantCount: 1,
		},
		{
			name:   "database error is wrapped",
			nodeID: "node-1",
			setup: func(client *mocks.MockReadingDatabaseServiceClient) {
				client.EXPECT().
					GetLatestReadings(gomock.Any(), gomock.Any()).
					Return(nil, errDatabaseFailed)
			},
			wantErr: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			client := mocks.NewMockReadingDatabaseServiceClient(ctrl)
			testCase.setup(client)

			gateway, err := New(GatewayParams{
				Logger:        zap.NewNop(),
				Config:        Config{Address: "127.0.0.1:8090"},
				ReadingClient: client,
			})
			if err != nil {
				t.Fatalf("New() unexpected error = %v", err)
			}

			readings, err := gateway.GetLatestReadings(context.Background(), testCase.nodeID, 5)
			if testCase.wantErr {
				if err == nil {
					t.Fatal("GetLatestReadings() want error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("GetLatestReadings() unexpected error = %v", err)
			}
			if len(readings) != testCase.wantCount {
				t.Fatalf("GetLatestReadings() count = %d, want %d", len(readings), testCase.wantCount)
			}
		})
	}
}

func TestGatewayListNodeCoordinates(t *testing.T) {
	ctrl := gomock.NewController(t)
	nodeClient := mocks.NewMockNodeDatabaseServiceClient(ctrl)
	readingClient := mocks.NewMockReadingDatabaseServiceClient(ctrl)

	nodeClient.EXPECT().
		ListNodeCoordinates(gomock.Any(), gomock.Any()).
		Return(&protodatabasev1.ListNodeCoordinatesResponse{
			Nodes: []*protodatabasev1.NodeCoordinates{
				{NodeId: "station-alpha"},
			},
		}, nil)

	gateway, err := New(GatewayParams{
		Logger:        zap.NewNop(),
		Config:        Config{Address: "127.0.0.1:8090"},
		ReadingClient: readingClient,
		NodeClient:    nodeClient,
	})
	if err != nil {
		t.Fatalf("New() unexpected error = %v", err)
	}

	nodes, err := gateway.ListNodeCoordinates(context.Background())
	if err != nil {
		t.Fatalf("ListNodeCoordinates() unexpected error = %v", err)
	}
	if len(nodes) != 1 || nodes[0].NodeId != "station-alpha" {
		t.Fatalf("unexpected nodes: %v", nodes)
	}
}

func TestGatewayCreateAndDeleteNode(t *testing.T) {
	errRPC := errors.New("rpc error")

	tests := []struct {
		name    string
		action  string
		setup   func(client *mocks.MockNodeDatabaseServiceClient)
		wantErr bool
	}{
		{
			name:   "create node succeeds",
			action: "create",
			setup: func(client *mocks.MockNodeDatabaseServiceClient) {
				client.EXPECT().
					CreateNode(gomock.Any(), gomock.Any()).
					Return(&protodatabasev1.CreateNodeResponse{}, nil)
			},
			wantErr: false,
		},
		{
			name:   "create node returns error on rpc failure",
			action: "create",
			setup: func(client *mocks.MockNodeDatabaseServiceClient) {
				client.EXPECT().
					CreateNode(gomock.Any(), gomock.Any()).
					Return(nil, errRPC)
			},
			wantErr: true,
		},
		{
			name:   "delete node succeeds",
			action: "delete",
			setup: func(client *mocks.MockNodeDatabaseServiceClient) {
				client.EXPECT().
					DeleteNode(gomock.Any(), &protodatabasev1.DeleteNodeRequest{NodeId: "station-alpha"}).
					Return(&protodatabasev1.DeleteNodeResponse{}, nil)
			},
			wantErr: false,
		},
		{
			name:   "delete node returns error on rpc failure",
			action: "delete",
			setup: func(client *mocks.MockNodeDatabaseServiceClient) {
				client.EXPECT().
					DeleteNode(gomock.Any(), gomock.Any()).
					Return(nil, errRPC)
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			nodeClient := mocks.NewMockNodeDatabaseServiceClient(ctrl)
			tc.setup(nodeClient)

			gateway, err := New(GatewayParams{
				Logger:     zap.NewNop(),
				Config:     Config{Address: "127.0.0.1:8090"},
				NodeClient: nodeClient,
			})
			if err != nil {
				t.Fatalf("New() unexpected error = %v", err)
			}

			var callErr error
			if tc.action == "create" {
				callErr = gateway.CreateNode(context.Background(), &protodatabasev1.NodeCoordinates{
					NodeId:       "station-alpha",
					Latitude:     45.0,
					Longitude:    7.0,
					ElevationM:   250.0,
					LocationName: "Turin",
				})
			} else {
				callErr = gateway.DeleteNode(context.Background(), "station-alpha")
			}

			if tc.wantErr && callErr == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && callErr != nil {
				t.Fatalf("unexpected error: %v", callErr)
			}
		})
	}
}

func TestGatewayGetForecast(t *testing.T) {
	errForecastFailed := errors.New("forecast unavailable")

	testCases := []struct {
		name      string
		nodeID    string
		withNil   bool
		setup     func(client *mocks.MockPredictionDatabaseServiceClient)
		wantErr   bool
		wantCount int
	}{
		{
			name:    "nil prediction client returns nil slice",
			nodeID:  "station-alpha",
			withNil: true,
		},
		{
			name:   "success returning forecast predictions",
			nodeID: "station-alpha",
			setup: func(client *mocks.MockPredictionDatabaseServiceClient) {
				client.EXPECT().
					GetForecast(gomock.Any(), gomock.Any()).
					Return(&protodatabasev1.GetForecastResponse{
						Predictions: []*protodatabasev1.Prediction{
							{NodeId: "station-alpha", HorizonHours: 1},
							{NodeId: "station-alpha", HorizonHours: 2},
						},
					}, nil)
			},
			wantCount: 2,
		},
		{
			name:   "database error is wrapped",
			nodeID: "station-alpha",
			setup: func(client *mocks.MockPredictionDatabaseServiceClient) {
				client.EXPECT().
					GetForecast(gomock.Any(), gomock.Any()).
					Return(nil, errForecastFailed)
			},
			wantErr: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			readingClient := mocks.NewMockReadingDatabaseServiceClient(ctrl)
			var predictionClient protodatabasev1.PredictionDatabaseServiceClient
			if !testCase.withNil {
				mockPrediction := mocks.NewMockPredictionDatabaseServiceClient(ctrl)
				if testCase.setup != nil {
					testCase.setup(mockPrediction)
				}
				predictionClient = mockPrediction
			}

			gateway, err := New(GatewayParams{
				Logger:           zap.NewNop(),
				Config:           Config{Address: "127.0.0.1:8090"},
				ReadingClient:    readingClient,
				PredictionClient: predictionClient,
			})
			if err != nil {
				t.Fatalf("New() unexpected error = %v", err)
			}

			predictions, err := gateway.GetForecast(context.Background(), testCase.nodeID)
			if testCase.wantErr {
				if err == nil {
					t.Fatal("GetForecast() want error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("GetForecast() unexpected error = %v", err)
			}
			if len(predictions) != testCase.wantCount {
				t.Fatalf("GetForecast() count = %d, want %d", len(predictions), testCase.wantCount)
			}
		})
	}
}

func TestNewGatewayValidation(t *testing.T) {
	_, err := New(GatewayParams{
		Logger: zap.NewNop(),
		Config: Config{Address: "127.0.0.1:8090"},
	})
	if !errors.Is(err, ErrNoClient) {
		t.Fatalf("New() error = %v, want %v", err, ErrNoClient)
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
