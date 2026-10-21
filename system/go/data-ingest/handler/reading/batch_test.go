package reading

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"weather-station/common/auth/basic"
	"weather-station/data-ingest/controller/weatherstation"
	"weather-station/data-ingest/gateway/database"
	"weather-station/data-ingest/internal/mocks"
	protodatabasev1 "weather-station/gen/go/weather/database/v1"
)

func TestBatchHandlerServeHTTP(t *testing.T) {
	errInternal := errors.New("database unavailable")

	testCases := []struct {
		name           string
		method         string
		body           string
		omitAuth       bool
		setup          func(client *mocks.MockReadingDatabaseServiceClient)
		wantStatusCode int
	}{
		{
			name:   "success on json array payload",
			method: http.MethodPost,
			body:   `[{"node_id":"n1","temperature":20.0},{"node_id":"n2","temperature":21.0}]`,
			setup: func(client *mocks.MockReadingDatabaseServiceClient) {
				client.EXPECT().
					StoreReadings(gomock.Any(), gomock.Any()).
					Return(&protodatabasev1.StoreReadingsResponse{Stored: 2}, nil)
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "unauthorized when basic auth is missing",
			method:         http.MethodPost,
			body:           `[{"node_id":"n1","temperature":20.0}]`,
			omitAuth:       true,
			setup:          func(client *mocks.MockReadingDatabaseServiceClient) {},
			wantStatusCode: http.StatusUnauthorized,
		},
		{
			name:   "success on json object payload",
			method: http.MethodPost,
			body:   `{"readings":[{"node_id":"n1","temperature":20.0}]}`,
			setup: func(client *mocks.MockReadingDatabaseServiceClient) {
				client.EXPECT().
					StoreReadings(gomock.Any(), gomock.Any()).
					Return(&protodatabasev1.StoreReadingsResponse{Stored: 1}, nil)
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "method not allowed for GET",
			method:         http.MethodGet,
			body:           `[]`,
			setup:          func(client *mocks.MockReadingDatabaseServiceClient) {},
			wantStatusCode: http.StatusMethodNotAllowed,
		},
		{
			name:           "bad request on empty body",
			method:         http.MethodPost,
			body:           `   `,
			setup:          func(client *mocks.MockReadingDatabaseServiceClient) {},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "bad request on empty readings array",
			method:         http.MethodPost,
			body:           `[]`,
			setup:          func(client *mocks.MockReadingDatabaseServiceClient) {},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "bad request on empty object readings",
			method:         http.MethodPost,
			body:           `{"readings":[]}`,
			setup:          func(client *mocks.MockReadingDatabaseServiceClient) {},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "bad request on malformed array json",
			method:         http.MethodPost,
			body:           `[{"node_id":`,
			setup:          func(client *mocks.MockReadingDatabaseServiceClient) {},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "bad request on missing node_id",
			method:         http.MethodPost,
			body:           `[{"temperature":20.0}]`,
			setup:          func(client *mocks.MockReadingDatabaseServiceClient) {},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:   "internal server error on failure",
			method: http.MethodPost,
			body:   `[{"node_id":"n1"}]`,
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

			auth := basic.NewAuthenticator(basic.Config{
				Username: "testuser",
				Password: "testpassword",
			})

			handler, err := NewBatch(BatchHandlerParams{
				Logger:     zap.NewNop(),
				Controller: controller,
				Auth:       auth,
			})
			if err != nil {
				t.Fatalf("NewBatch() unexpected error = %v", err)
			}

			request := httptest.NewRequest(testCase.method, "/readings", bytes.NewBufferString(testCase.body))
			if !testCase.omitAuth {
				request.SetBasicAuth("testuser", "testpassword")
			}
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
