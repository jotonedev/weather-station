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
	storeReadingsFunc func(ctx context.Context, in *protodatabasev1.StoreReadingsRequest, opts ...grpc.CallOption) (*protodatabasev1.StoreReadingsResponse, error)
}

func (f *fakeReadingClient) StoreReadings(ctx context.Context, in *protodatabasev1.StoreReadingsRequest, opts ...grpc.CallOption) (*protodatabasev1.StoreReadingsResponse, error) {
	if f.storeReadingsFunc != nil {
		return f.storeReadingsFunc(ctx, in, opts...)
	}
	return &protodatabasev1.StoreReadingsResponse{Stored: uint64(len(in.Readings))}, nil
}

func TestGatewayStoreReading(t *testing.T) {
	errDatabaseFailed := errors.New("connection failed")

	testCases := []struct {
		name              string
		reading           *protodatabasev1.Reading
		storeReadingsFunc func(ctx context.Context, in *protodatabasev1.StoreReadingsRequest, opts ...grpc.CallOption) (*protodatabasev1.StoreReadingsResponse, error)
		wantErr           bool
		expectedErr       error
	}{
		{
			name: "success storing single reading",
			reading: &protodatabasev1.Reading{
				NodeId: "node-1",
			},
			storeReadingsFunc: func(ctx context.Context, in *protodatabasev1.StoreReadingsRequest, opts ...grpc.CallOption) (*protodatabasev1.StoreReadingsResponse, error) {
				if len(in.Readings) != 1 || in.Readings[0].NodeId != "node-1" {
					t.Fatalf("unexpected readings: %v", in.Readings)
				}
				return &protodatabasev1.StoreReadingsResponse{Stored: 1}, nil
			},
			wantErr: false,
		},
		{
			name:        "nil reading returns error",
			reading:     nil,
			wantErr:     true,
			expectedErr: ErrNilReading,
		},
		{
			name: "database service error is wrapped",
			reading: &protodatabasev1.Reading{
				NodeId: "node-1",
			},
			storeReadingsFunc: func(ctx context.Context, in *protodatabasev1.StoreReadingsRequest, opts ...grpc.CallOption) (*protodatabasev1.StoreReadingsResponse, error) {
				return nil, errDatabaseFailed
			},
			wantErr: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			client := &fakeReadingClient{storeReadingsFunc: testCase.storeReadingsFunc}
			gateway, err := New(GatewayParams{
				Logger: zap.NewNop(),
				Config: Config{Address: "127.0.0.1:8090"},
				Client: client,
			})
			if err != nil {
				t.Fatalf("New() unexpected error = %v", err)
			}

			err = gateway.StoreReading(context.Background(), testCase.reading)
			if testCase.wantErr {
				if err == nil {
					t.Fatal("StoreReading() want error, got nil")
				}
				if testCase.expectedErr != nil && !errors.Is(err, testCase.expectedErr) {
					t.Fatalf("StoreReading() error = %v, want %v", err, testCase.expectedErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("StoreReading() unexpected error = %v", err)
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
