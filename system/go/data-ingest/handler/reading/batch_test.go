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

func TestBatchHandlerServeHTTP(t *testing.T) {
	errInternal := errors.New("database unavailable")

	testCases := []struct {
		name              string
		method            string
		body              string
		storeReadingsFunc func(ctx context.Context, in *protodatabasev1.StoreReadingsRequest, opts ...grpc.CallOption) (*protodatabasev1.StoreReadingsResponse, error)
		wantStatusCode    int
	}{
		{
			name:           "success on json array payload",
			method:         http.MethodPost,
			body:           `[{"node_id":"n1","temperature":20.0},{"node_id":"n2","temperature":21.0}]`,
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "success on json object payload",
			method:         http.MethodPost,
			body:           `{"readings":[{"node_id":"n1","temperature":20.0}]}`,
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "method not allowed for GET",
			method:         http.MethodGet,
			body:           `[]`,
			wantStatusCode: http.StatusMethodNotAllowed,
		},
		{
			name:           "bad request on empty body",
			method:         http.MethodPost,
			body:           `   `,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "bad request on empty readings array",
			method:         http.MethodPost,
			body:           `[]`,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "bad request on empty object readings",
			method:         http.MethodPost,
			body:           `{"readings":[]}`,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "bad request on malformed array json",
			method:         http.MethodPost,
			body:           `[{"node_id":`,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "bad request on missing node_id",
			method:         http.MethodPost,
			body:           `[{"temperature":20.0}]`,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:   "internal server error on failure",
			method: http.MethodPost,
			body:   `[{"node_id":"n1"}]`,
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

			handler, err := NewBatch(BatchHandlerParams{
				Logger:     zap.NewNop(),
				Controller: ctrl,
			})
			if err != nil {
				t.Fatalf("NewBatch() unexpected error = %v", err)
			}

			request := httptest.NewRequest(testCase.method, "/readings", bytes.NewBufferString(testCase.body))
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)

			if recorder.Code != testCase.wantStatusCode {
				t.Fatalf("ServeHTTP() status = %d, want %d", recorder.Code, testCase.wantStatusCode)
			}
		})
	}
}

func TestBatchHandlerPattern(t *testing.T) {
	handler := &BatchHandler{}
	if pattern := handler.Pattern(); pattern != "/readings" {
		t.Fatalf("Pattern() = %q, want /readings", pattern)
	}
}
