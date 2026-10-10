package reading

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	"weather-station/data-ingest/controller/weatherstation"
	"weather-station/data-ingest/gateway/database"
	protodatabasev1 "weather-station/gen/go/weather/database/v1"
)

type fakeDatabaseClient struct {
	protodatabasev1.ReadingDatabaseServiceClient
	storeReadingsFunc func(ctx context.Context, in *protodatabasev1.StoreReadingsRequest, opts ...grpc.CallOption) (*protodatabasev1.StoreReadingsResponse, error)
}

func (f *fakeDatabaseClient) StoreReadings(ctx context.Context, in *protodatabasev1.StoreReadingsRequest, opts ...grpc.CallOption) (*protodatabasev1.StoreReadingsResponse, error) {
	if f.storeReadingsFunc != nil {
		return f.storeReadingsFunc(ctx, in, opts...)
	}
	return &protodatabasev1.StoreReadingsResponse{Stored: uint64(len(in.Readings))}, nil
}

func TestHandlerServeHTTP(t *testing.T) {
	errInternal := errors.New("database unavailable")

	testCases := []struct {
		name              string
		method            string
		body              string
		storeReadingsFunc func(ctx context.Context, in *protodatabasev1.StoreReadingsRequest, opts ...grpc.CallOption) (*protodatabasev1.StoreReadingsResponse, error)
		wantStatusCode    int
	}{
		{
			name:           "success on valid reading json",
			method:         http.MethodPost,
			body:           `{"node_id":"node-123","temperature":21.5,"humidity":55.0}`,
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "method not allowed for GET request",
			method:         http.MethodGet,
			body:           `{"node_id":"node-123"}`,
			wantStatusCode: http.StatusMethodNotAllowed,
		},
		{
			name:           "bad request on malformed json",
			method:         http.MethodPost,
			body:           `{not-valid-json`,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "bad request on missing node_id",
			method:         http.MethodPost,
			body:           `{"temperature":21.5}`,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:   "internal server error on downstream failure",
			method: http.MethodPost,
			body:   `{"node_id":"node-123","temperature":21.5}`,
			storeReadingsFunc: func(ctx context.Context, in *protodatabasev1.StoreReadingsRequest, opts ...grpc.CallOption) (*protodatabasev1.StoreReadingsResponse, error) {
				return nil, errInternal
			},
			wantStatusCode: http.StatusInternalServerError,
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

			ctrl, err := weatherstation.New(weatherstation.ControllerParams{
				Logger:   zap.NewNop(),
				Database: dbGateway,
			})
			if err != nil {
				t.Fatalf("weatherstation.New() unexpected error = %v", err)
			}

			handler, err := New(HandlerParams{
				Logger:     zap.NewNop(),
				Controller: ctrl,
			})
			if err != nil {
				t.Fatalf("New() unexpected error = %v", err)
			}

			request := httptest.NewRequest(testCase.method, "/reading", bytes.NewBufferString(testCase.body))
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)

			if recorder.Code != testCase.wantStatusCode {
				t.Fatalf("ServeHTTP() status = %d, want %d", recorder.Code, testCase.wantStatusCode)
			}
		})
	}
}

func TestHandlerPattern(t *testing.T) {
	handler := &Handler{}
	if pattern := handler.Pattern(); pattern != "/reading" {
		t.Fatalf("Pattern() = %q, want /reading", pattern)
	}
}
