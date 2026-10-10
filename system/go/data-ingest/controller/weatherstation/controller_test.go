package weatherstation

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"weather-station/data-ingest/gateway/database"
	"weather-station/data-ingest/internal/mocks"
	protodatabasev1 "weather-station/gen/go/weather/database/v1"
)

func TestControllerStoreReading(t *testing.T) {
	testTime := time.Date(2026, 10, 10, 10, 0, 0, 0, time.UTC)
	errGateway := errors.New("gateway failure")

	testCases := []struct {
		name        string
		reading     Reading
		setup       func(client *mocks.MockReadingDatabaseServiceClient)
		wantErr     bool
		expectedErr error
	}{
		{
			name: "success storing valid reading",
			reading: Reading{
				NodeID:    "station-alpha",
				Timestamp: testTime,
			},
			setup: func(client *mocks.MockReadingDatabaseServiceClient) {
				client.EXPECT().
					StoreReadings(gomock.Any(), gomock.AssignableToTypeOf(&protodatabasev1.StoreReadingsRequest{})).
					DoAndReturn(func(ctx context.Context, in *protodatabasev1.StoreReadingsRequest, opts ...grpc.CallOption) (*protodatabasev1.StoreReadingsResponse, error) {
						if len(in.Readings) != 1 {
							t.Fatalf("stored readings length = %d, want 1", len(in.Readings))
						}
						if in.Readings[0].NodeId != "station-alpha" {
							t.Fatalf("stored NodeId = %q, want station-alpha", in.Readings[0].NodeId)
						}
						return &protodatabasev1.StoreReadingsResponse{Stored: 1}, nil
					})
			},
			wantErr: false,
		},
		{
			name: "sets default timestamp when zero",
			reading: Reading{
				NodeID: "station-beta",
			},
			setup: func(client *mocks.MockReadingDatabaseServiceClient) {
				client.EXPECT().
					StoreReadings(gomock.Any(), gomock.AssignableToTypeOf(&protodatabasev1.StoreReadingsRequest{})).
					DoAndReturn(func(ctx context.Context, in *protodatabasev1.StoreReadingsRequest, opts ...grpc.CallOption) (*protodatabasev1.StoreReadingsResponse, error) {
						if len(in.Readings) != 1 {
							t.Fatalf("stored readings length = %d, want 1", len(in.Readings))
						}
						if in.Readings[0].Timestamp == nil {
							t.Fatal("expected timestamp to be populated, got nil")
						}
						return &protodatabasev1.StoreReadingsResponse{Stored: 1}, nil
					})
			},
			wantErr: false,
		},
		{
			name: "missing node id returns error",
			reading: Reading{
				Timestamp: testTime,
			},
			setup:       func(client *mocks.MockReadingDatabaseServiceClient) {},
			wantErr:     true,
			expectedErr: ErrMissingNodeID,
		},
		{
			name: "gateway error is wrapped",
			reading: Reading{
				NodeID:    "station-gamma",
				Timestamp: testTime,
			},
			setup: func(client *mocks.MockReadingDatabaseServiceClient) {
				client.EXPECT().
					StoreReadings(gomock.Any(), gomock.Any()).
					Return(nil, errGateway)
			},
			wantErr: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockClient := mocks.NewMockReadingDatabaseServiceClient(ctrl)
			testCase.setup(mockClient)

			dbGateway, err := database.New(database.GatewayParams{
				Logger: zap.NewNop(),
				Config: database.Config{Address: "127.0.0.1:8090"},
				Client: mockClient,
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

			err = controller.StoreReading(context.Background(), testCase.reading)
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

func TestControllerStoreReadings(t *testing.T) {
	testTime := time.Date(2026, 10, 10, 10, 0, 0, 0, time.UTC)
	errGateway := errors.New("gateway batch failure")

	testCases := []struct {
		name        string
		readings    []Reading
		setup       func(client *mocks.MockReadingDatabaseServiceClient)
		wantErr     bool
		expectedErr error
	}{
		{
			name:        "empty batch returns without error",
			readings:    nil,
			setup:       func(client *mocks.MockReadingDatabaseServiceClient) {},
			wantErr:     false,
		},
		{
			name: "success storing multiple readings",
			readings: []Reading{
				{NodeID: "station-1", Timestamp: testTime},
				{NodeID: "station-2", Timestamp: testTime},
			},
			setup: func(client *mocks.MockReadingDatabaseServiceClient) {
				client.EXPECT().
					StoreReadings(gomock.Any(), gomock.AssignableToTypeOf(&protodatabasev1.StoreReadingsRequest{})).
					DoAndReturn(func(ctx context.Context, in *protodatabasev1.StoreReadingsRequest, opts ...grpc.CallOption) (*protodatabasev1.StoreReadingsResponse, error) {
						if len(in.Readings) != 2 {
							t.Fatalf("stored readings length = %d, want 2", len(in.Readings))
						}
						if in.Readings[0].NodeId != "station-1" || in.Readings[1].NodeId != "station-2" {
							t.Fatalf("unexpected stored node ids: %v, %v", in.Readings[0].NodeId, in.Readings[1].NodeId)
						}
						return &protodatabasev1.StoreReadingsResponse{Stored: 2}, nil
					})
			},
			wantErr: false,
		},
		{
			name: "missing node id in batch returns error",
			readings: []Reading{
				{NodeID: "station-1", Timestamp: testTime},
				{Timestamp: testTime},
			},
			setup:       func(client *mocks.MockReadingDatabaseServiceClient) {},
			wantErr:     true,
			expectedErr: ErrMissingNodeID,
		},
		{
			name: "gateway error is wrapped in batch",
			readings: []Reading{
				{NodeID: "station-1", Timestamp: testTime},
			},
			setup: func(client *mocks.MockReadingDatabaseServiceClient) {
				client.EXPECT().
					StoreReadings(gomock.Any(), gomock.Any()).
					Return(nil, errGateway)
			},
			wantErr: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockClient := mocks.NewMockReadingDatabaseServiceClient(ctrl)
			testCase.setup(mockClient)

			dbGateway, err := database.New(database.GatewayParams{
				Logger: zap.NewNop(),
				Config: database.Config{Address: "127.0.0.1:8090"},
				Client: mockClient,
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

			err = controller.StoreReadings(context.Background(), testCase.readings)
			if testCase.wantErr {
				if err == nil {
					t.Fatal("StoreReadings() want error, got nil")
				}
				if testCase.expectedErr != nil && !errors.Is(err, testCase.expectedErr) {
					t.Fatalf("StoreReadings() error = %v, want %v", err, testCase.expectedErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("StoreReadings() unexpected error = %v", err)
			}
		})
	}
}
