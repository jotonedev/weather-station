package database

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

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

func TestNewGatewayValidation(t *testing.T) {
	_, err := New(GatewayParams{
		Logger: zap.NewNop(),
		Config: Config{Address: "127.0.0.1:8090"},
	})
	if !errors.Is(err, ErrNoClient) {
		t.Fatalf("New() error = %v, want %v", err, ErrNoClient)
	}
}
