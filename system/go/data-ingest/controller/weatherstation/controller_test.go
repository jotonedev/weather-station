package weatherstation

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	"weather-station/data-ingest/gateway/database"
	protodatabasev1 "weather-station/gen/go/weather/database/v1"
)

type fakeDatabaseClient struct {
	protodatabasev1.ReadingDatabaseServiceClient
	storedReadings    []*protodatabasev1.Reading
	storeReadingsFunc func(ctx context.Context, in *protodatabasev1.StoreReadingsRequest, opts ...grpc.CallOption) (*protodatabasev1.StoreReadingsResponse, error)
}

func (f *fakeDatabaseClient) StoreReadings(ctx context.Context, in *protodatabasev1.StoreReadingsRequest, opts ...grpc.CallOption) (*protodatabasev1.StoreReadingsResponse, error) {
	f.storedReadings = append(f.storedReadings, in.Readings...)
	if f.storeReadingsFunc != nil {
		return f.storeReadingsFunc(ctx, in, opts...)
	}
	return &protodatabasev1.StoreReadingsResponse{Stored: uint64(len(in.Readings))}, nil
}

func TestControllerStoreReading(t *testing.T) {
	testTime := time.Date(2026, 10, 10, 10, 0, 0, 0, time.UTC)
	errGateway := errors.New("gateway failure")

	testCases := []struct {
		name              string
		reading           Reading
		storeReadingsFunc func(ctx context.Context, in *protodatabasev1.StoreReadingsRequest, opts ...grpc.CallOption) (*protodatabasev1.StoreReadingsResponse, error)
		wantErr           bool
		expectedErr       error
		checkStored       func(t *testing.T, readings []*protodatabasev1.Reading)
	}{
		{
			name: "success storing valid reading",
			reading: Reading{
				NodeID:    "station-alpha",
				Timestamp: testTime,
			},
			wantErr: false,
			checkStored: func(t *testing.T, readings []*protodatabasev1.Reading) {
				if len(readings) != 1 {
					t.Fatalf("stored readings length = %d, want 1", len(readings))
				}
				if readings[0].NodeId != "station-alpha" {
					t.Fatalf("stored NodeId = %q, want station-alpha", readings[0].NodeId)
				}
			},
		},
		{
			name: "sets default timestamp when zero",
			reading: Reading{
				NodeID: "station-beta",
			},
			wantErr: false,
			checkStored: func(t *testing.T, readings []*protodatabasev1.Reading) {
				if len(readings) != 1 {
					t.Fatalf("stored readings length = %d, want 1", len(readings))
				}
				if readings[0].Timestamp == nil {
					t.Fatal("expected timestamp to be populated, got nil")
				}
			},
		},
		{
			name: "missing node id returns error",
			reading: Reading{
				Timestamp: testTime,
			},
			wantErr:     true,
			expectedErr: ErrMissingNodeID,
		},
		{
			name: "gateway error is wrapped",
			reading: Reading{
				NodeID:    "station-gamma",
				Timestamp: testTime,
			},
			storeReadingsFunc: func(ctx context.Context, in *protodatabasev1.StoreReadingsRequest, opts ...grpc.CallOption) (*protodatabasev1.StoreReadingsResponse, error) {
				return nil, errGateway
			},
			wantErr: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fakeClient := &fakeDatabaseClient{storeReadingsFunc: testCase.storeReadingsFunc}
			dbGateway, err := database.New(database.GatewayParams{
				Logger: zap.NewNop(),
				Config: database.Config{Address: "127.0.0.1:8090"},
				Client: fakeClient,
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
			if testCase.checkStored != nil {
				testCase.checkStored(t, fakeClient.storedReadings)
			}
		})
	}
}

func TestControllerStoreReadings(t *testing.T) {
	testTime := time.Date(2026, 10, 10, 10, 0, 0, 0, time.UTC)
	errGateway := errors.New("gateway batch failure")

	testCases := []struct {
		name              string
		readings          []Reading
		storeReadingsFunc func(ctx context.Context, in *protodatabasev1.StoreReadingsRequest, opts ...grpc.CallOption) (*protodatabasev1.StoreReadingsResponse, error)
		wantErr           bool
		expectedErr       error
		checkStored       func(t *testing.T, readings []*protodatabasev1.Reading)
	}{
		{
			name:     "empty batch returns without error",
			readings: nil,
			wantErr:  false,
			checkStored: func(t *testing.T, readings []*protodatabasev1.Reading) {
				if len(readings) != 0 {
					t.Fatalf("stored readings length = %d, want 0", len(readings))
				}
			},
		},
		{
			name: "success storing multiple readings",
			readings: []Reading{
				{NodeID: "station-1", Timestamp: testTime},
				{NodeID: "station-2", Timestamp: testTime},
			},
			wantErr: false,
			checkStored: func(t *testing.T, readings []*protodatabasev1.Reading) {
				if len(readings) != 2 {
					t.Fatalf("stored readings length = %d, want 2", len(readings))
				}
				if readings[0].NodeId != "station-1" || readings[1].NodeId != "station-2" {
					t.Fatalf("unexpected stored node ids: %v, %v", readings[0].NodeId, readings[1].NodeId)
				}
			},
		},
		{
			name: "missing node id in batch returns error",
			readings: []Reading{
				{NodeID: "station-1", Timestamp: testTime},
				{Timestamp: testTime},
			},
			wantErr:     true,
			expectedErr: ErrMissingNodeID,
		},
		{
			name: "gateway error is wrapped in batch",
			readings: []Reading{
				{NodeID: "station-1", Timestamp: testTime},
			},
			storeReadingsFunc: func(ctx context.Context, in *protodatabasev1.StoreReadingsRequest, opts ...grpc.CallOption) (*protodatabasev1.StoreReadingsResponse, error) {
				return nil, errGateway
			},
			wantErr: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fakeClient := &fakeDatabaseClient{storeReadingsFunc: testCase.storeReadingsFunc}
			dbGateway, err := database.New(database.GatewayParams{
				Logger: zap.NewNop(),
				Config: database.Config{Address: "127.0.0.1:8090"},
				Client: fakeClient,
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
			if testCase.checkStored != nil {
				testCase.checkStored(t, fakeClient.storedReadings)
			}
		})
	}
}
