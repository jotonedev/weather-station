package database

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"

	"weather-station/data-ingest/internal/mocks"
	protodatabasev1 "weather-station/gen/go/weather/database/v1"
)

func TestGatewayStoreReading(t *testing.T) {
	errDatabaseFailed := errors.New("connection failed")

	testCases := []struct {
		name        string
		reading     *protodatabasev1.Reading
		setup       func(client *mocks.MockReadingDatabaseServiceClient)
		wantErr     bool
		expectedErr error
	}{
		{
			name: "success storing single reading",
			reading: &protodatabasev1.Reading{
				NodeId: "node-1",
			},
			setup: func(client *mocks.MockReadingDatabaseServiceClient) {
				client.EXPECT().
					StoreReadings(gomock.Any(), gomock.AssignableToTypeOf(&protodatabasev1.StoreReadingsRequest{})).
					DoAndReturn(func(ctx context.Context, in *protodatabasev1.StoreReadingsRequest, opts ...grpc.CallOption) (*protodatabasev1.StoreReadingsResponse, error) {
						if len(in.Readings) != 1 || in.Readings[0].NodeId != "node-1" {
							t.Fatalf("unexpected readings: %v", in.Readings)
						}
						return &protodatabasev1.StoreReadingsResponse{Stored: 1}, nil
					})
			},
			wantErr: false,
		},
		{
			name:        "nil reading returns error",
			reading:     nil,
			setup:       func(client *mocks.MockReadingDatabaseServiceClient) {},
			wantErr:     true,
			expectedErr: ErrNilReading,
		},
		{
			name: "database service error is wrapped",
			reading: &protodatabasev1.Reading{
				NodeId: "node-1",
			},
			setup: func(client *mocks.MockReadingDatabaseServiceClient) {
				client.EXPECT().
					StoreReadings(gomock.Any(), gomock.Any()).
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
