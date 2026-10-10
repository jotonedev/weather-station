package reading

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"weather-station/data-ingest/controller/weatherstation"
	"weather-station/data-ingest/gateway/database"
	"weather-station/data-ingest/internal/mocks"
	protodatabasev1 "weather-station/gen/go/weather/database/v1"
)

func TestHandlerServeHTTP(t *testing.T) {
	errInternal := errors.New("database unavailable")

	testCases := []struct {
		name           string
		method         string
		body           string
		setup          func(client *mocks.MockReadingDatabaseServiceClient)
		wantStatusCode int
	}{
		{
			name:   "success on valid reading json",
			method: http.MethodPost,
			body:   `{"node_id":"node-123","temperature":21.5,"humidity":55.0}`,
			setup: func(client *mocks.MockReadingDatabaseServiceClient) {
				client.EXPECT().
					StoreReadings(gomock.Any(), gomock.Any()).
					Return(&protodatabasev1.StoreReadingsResponse{Stored: 1}, nil)
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "method not allowed for GET request",
			method:         http.MethodGet,
			body:           `{"node_id":"node-123"}`,
			setup:          func(client *mocks.MockReadingDatabaseServiceClient) {},
			wantStatusCode: http.StatusMethodNotAllowed,
		},
		{
			name:           "bad request on malformed json",
			method:         http.MethodPost,
			body:           `{not-valid-json`,
			setup:          func(client *mocks.MockReadingDatabaseServiceClient) {},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "bad request on missing node_id",
			method:         http.MethodPost,
			body:           `{"temperature":21.5}`,
			setup:          func(client *mocks.MockReadingDatabaseServiceClient) {},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:   "internal server error on downstream failure",
			method: http.MethodPost,
			body:   `{"node_id":"node-123","temperature":21.5}`,
			setup: func(client *mocks.MockReadingDatabaseServiceClient) {
				client.EXPECT().
					StoreReadings(gomock.Any(), gomock.Any()).
					Return(nil, errInternal)
			},
			wantStatusCode: http.StatusInternalServerError,
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

			controller, err := weatherstation.New(weatherstation.ControllerParams{
				Logger:   zap.NewNop(),
				Database: dbGateway,
			})
			if err != nil {
				t.Fatalf("weatherstation.New() unexpected error = %v", err)
			}

			handler, err := New(HandlerParams{
				Logger:     zap.NewNop(),
				Controller: controller,
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
