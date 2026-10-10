package weatherdata

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	"weather-station/frontend/gateway/database"
	protodatabasev1 "weather-station/gen/go/weather/database/v1"
)

type fakeReadingClient struct {
	protodatabasev1.ReadingDatabaseServiceClient
	getLatestReadingsFunc func(ctx context.Context, in *protodatabasev1.GetLatestReadingsRequest, opts ...grpc.CallOption) (*protodatabasev1.GetLatestReadingsResponse, error)
}

func (f *fakeReadingClient) GetLatestReadings(ctx context.Context, in *protodatabasev1.GetLatestReadingsRequest, opts ...grpc.CallOption) (*protodatabasev1.GetLatestReadingsResponse, error) {
	if f.getLatestReadingsFunc != nil {
		return f.getLatestReadingsFunc(ctx, in, opts...)
	}
	return &protodatabasev1.GetLatestReadingsResponse{}, nil
}

type fakeNodeClient struct {
	protodatabasev1.NodeDatabaseServiceClient
	listNodeCoordinatesFunc func(ctx context.Context, in *protodatabasev1.ListNodeCoordinatesRequest, opts ...grpc.CallOption) (*protodatabasev1.ListNodeCoordinatesResponse, error)
}

func (f *fakeNodeClient) ListNodeCoordinates(ctx context.Context, in *protodatabasev1.ListNodeCoordinatesRequest, opts ...grpc.CallOption) (*protodatabasev1.ListNodeCoordinatesResponse, error) {
	if f.listNodeCoordinatesFunc != nil {
		return f.listNodeCoordinatesFunc(ctx, in, opts...)
	}
	return &protodatabasev1.ListNodeCoordinatesResponse{}, nil
}

func TestGetDashboardData(t *testing.T) {
	temp := float32(23.4)

	testCases := []struct {
		name                  string
		requestedNode         string
		listNodesFunc         func(ctx context.Context, in *protodatabasev1.ListNodeCoordinatesRequest, opts ...grpc.CallOption) (*protodatabasev1.ListNodeCoordinatesResponse, error)
		getLatestReadingsFunc func(ctx context.Context, in *protodatabasev1.GetLatestReadingsRequest, opts ...grpc.CallOption) (*protodatabasev1.GetLatestReadingsResponse, error)
		check                 func(t *testing.T, data DashboardData)
	}{
		{
			name:          "displays latest readings when available",
			requestedNode: "station-beta",
			listNodesFunc: func(ctx context.Context, in *protodatabasev1.ListNodeCoordinatesRequest, opts ...grpc.CallOption) (*protodatabasev1.ListNodeCoordinatesResponse, error) {
				return &protodatabasev1.ListNodeCoordinatesResponse{
					Nodes: []*protodatabasev1.NodeCoordinates{
						{NodeId: "station-alpha"},
						{NodeId: "station-beta"},
					},
				}, nil
			},
			getLatestReadingsFunc: func(ctx context.Context, in *protodatabasev1.GetLatestReadingsRequest, opts ...grpc.CallOption) (*protodatabasev1.GetLatestReadingsResponse, error) {
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
			getLatestReadingsFunc: func(ctx context.Context, in *protodatabasev1.GetLatestReadingsRequest, opts ...grpc.CallOption) (*protodatabasev1.GetLatestReadingsResponse, error) {
				return nil, errors.New("connection failed")
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
			dbGateway, err := database.New(database.GatewayParams{
				Logger: zap.NewNop(),
				Config: database.Config{Address: "127.0.0.1:8090"},
				ReadingClient: &fakeReadingClient{
					getLatestReadingsFunc: testCase.getLatestReadingsFunc,
				},
				NodeClient: &fakeNodeClient{
					listNodeCoordinatesFunc: testCase.listNodesFunc,
				},
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
