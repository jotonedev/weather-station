package weatherdata

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"weather-station/frontend/gateway/database"
	"weather-station/frontend/internal/mocks"
	protodatabasev1 "weather-station/gen/go/weather/database/v1"
)

func TestGetDashboardData(t *testing.T) {
	temp := float32(23.4)

	testCases := []struct {
		name          string
		requestedNode string
		setup         func(readingClient *mocks.MockReadingDatabaseServiceClient, nodeClient *mocks.MockNodeDatabaseServiceClient)
		check         func(t *testing.T, data DashboardData)
	}{
		{
			name:          "displays latest readings when available",
			requestedNode: "station-beta",
			setup: func(readingClient *mocks.MockReadingDatabaseServiceClient, nodeClient *mocks.MockNodeDatabaseServiceClient) {
				nodeClient.EXPECT().
					ListNodeCoordinates(gomock.Any(), gomock.Any()).
					Return(&protodatabasev1.ListNodeCoordinatesResponse{
						Nodes: []*protodatabasev1.NodeCoordinates{
							{NodeId: "station-alpha"},
							{NodeId: "station-beta"},
						},
					}, nil)

				readingClient.EXPECT().
					GetLatestReadings(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, in *protodatabasev1.GetLatestReadingsRequest, opts ...any) (*protodatabasev1.GetLatestReadingsResponse, error) {
						if in.NodeId != "station-beta" {
							t.Fatalf("NodeId = %q, want station-beta", in.NodeId)
						}
						return &protodatabasev1.GetLatestReadingsResponse{
							Readings: []*protodatabasev1.Reading{
								{
									NodeId:      "station-beta",
									Temperature: &temp,
								},
							},
						}, nil
					})
			},
			check: func(t *testing.T, data DashboardData) {
				if data.SelectedNodeID != "station-beta" {
					t.Fatalf("SelectedNodeID = %q, want station-beta", data.SelectedNodeID)
				}
				if len(data.AvailableNodes) != 2 {
					t.Fatalf("AvailableNodes length = %d, want 2", len(data.AvailableNodes))
				}
				if data.LatestReading == nil {
					t.Fatal("expected LatestReading to not be nil")
				}
				if data.LatestReading.Temperature != "23.4 °C" {
					t.Fatalf("Temperature = %q, want 23.4 °C", data.LatestReading.Temperature)
				}
			},
		},
		{
			name:          "handles empty database gracefully with fallback empty state",
			requestedNode: "",
			setup: func(readingClient *mocks.MockReadingDatabaseServiceClient, nodeClient *mocks.MockNodeDatabaseServiceClient) {
				nodeClient.EXPECT().
					ListNodeCoordinates(gomock.Any(), gomock.Any()).
					Return(&protodatabasev1.ListNodeCoordinatesResponse{}, nil)

				readingClient.EXPECT().
					GetLatestReadings(gomock.Any(), gomock.Any()).
					Return(&protodatabasev1.GetLatestReadingsResponse{}, nil)
			},
			check: func(t *testing.T, data DashboardData) {
				if data.SelectedNodeID != "station-alpha" {
					t.Fatalf("SelectedNodeID = %q, want station-alpha", data.SelectedNodeID)
				}
				if data.LatestReading != nil {
					t.Fatalf("LatestReading = %v, want nil", data.LatestReading)
				}
			},
		},
		{
			name:          "sets error message when database fails",
			requestedNode: "station-1",
			setup: func(readingClient *mocks.MockReadingDatabaseServiceClient, nodeClient *mocks.MockNodeDatabaseServiceClient) {
				nodeClient.EXPECT().
					ListNodeCoordinates(gomock.Any(), gomock.Any()).
					Return(&protodatabasev1.ListNodeCoordinatesResponse{}, nil)

				readingClient.EXPECT().
					GetLatestReadings(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("connection failed"))
			},
			check: func(t *testing.T, data DashboardData) {
				if data.ErrorMessage == "" {
					t.Fatal("expected ErrorMessage to be populated")
				}
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			readingClient := mocks.NewMockReadingDatabaseServiceClient(ctrl)
			nodeClient := mocks.NewMockNodeDatabaseServiceClient(ctrl)
			testCase.setup(readingClient, nodeClient)

			dbGateway, err := database.New(database.GatewayParams{
				Logger:        zap.NewNop(),
				Config:        database.Config{Address: "127.0.0.1:8090"},
				ReadingClient: readingClient,
				NodeClient:    nodeClient,
			})
			if err != nil {
				t.Fatalf("database.New() unexpected error = %v", err)
			}

			controller, err := New(ControllerParams{
				Logger:   zap.NewNop(),
				Database: dbGateway,
			})
			if err != nil {
				t.Fatalf("New() unexpected error = %v", err)
			}

			data := controller.GetDashboardData(context.Background(), testCase.requestedNode)
			testCase.check(t, data)
		})
	}
}
