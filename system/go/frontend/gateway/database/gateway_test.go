package database

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/zap"
	"google.golang.org/grpc"

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

func TestGatewayGetLatestReadings(t *testing.T) {
	errDatabaseFailed := errors.New("readings unavailable")

	testCases := []struct {
		name                  string
		nodeID                string
		getLatestReadingsFunc func(ctx context.Context, in *protodatabasev1.GetLatestReadingsRequest, opts ...grpc.CallOption) (*protodatabasev1.GetLatestReadingsResponse, error)
		wantErr               bool
		wantCount             int
	}{
		{
			name:   "success returning latest readings",
			nodeID: "node-1",
			getLatestReadingsFunc: func(ctx context.Context, in *protodatabasev1.GetLatestReadingsRequest, opts ...grpc.CallOption) (*protodatabasev1.GetLatestReadingsResponse, error) {
				if in.NodeId != "node-1" {
					t.Fatalf("NodeId = %q, want node-1", in.NodeId)
				}
				return &protodatabasev1.GetLatestReadingsResponse{
					Readings: []*protodatabasev1.Reading{
						{NodeId: "node-1"},
					},
				}, nil
			},
			wantErr:   false,
			wantCount: 1,
		},
		{
			name:   "database error is wrapped",
			nodeID: "node-1",
			getLatestReadingsFunc: func(ctx context.Context, in *protodatabasev1.GetLatestReadingsRequest, opts ...grpc.CallOption) (*protodatabasev1.GetLatestReadingsResponse, error) {
				return nil, errDatabaseFailed
			},
			wantErr: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			client := &fakeReadingClient{getLatestReadingsFunc: testCase.getLatestReadingsFunc}
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
	nodeClient := &fakeNodeClient{
		listNodeCoordinatesFunc: func(ctx context.Context, in *protodatabasev1.ListNodeCoordinatesRequest, opts ...grpc.CallOption) (*protodatabasev1.ListNodeCoordinatesResponse, error) {
			return &protodatabasev1.ListNodeCoordinatesResponse{
				Nodes: []*protodatabasev1.NodeCoordinates{
					{NodeId: "station-alpha"},
				},
			}, nil
		},
	}

	gateway, err := New(GatewayParams{
		Logger:        zap.NewNop(),
		Config:        Config{Address: "127.0.0.1:8090"},
		ReadingClient: &fakeReadingClient{},
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
