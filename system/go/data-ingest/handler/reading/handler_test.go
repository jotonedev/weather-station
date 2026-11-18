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

func TestHandlerServeHTTP(t *testing.T) {
	errInternal := errors.New("database unavailable")

	testCases := []struct {
		name           string
		method         string
		body           string
		omitAuth       bool
		authUser       string
		authPass       string
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
			name:           "unauthorized when basic auth is missing",
			method:         http.MethodPost,
			body:           `{"node_id":"node-123","temperature":21.5}`,
			omitAuth:       true,
			setup:          func(client *mocks.MockReadingDatabaseServiceClient) {},
			wantStatusCode: http.StatusUnauthorized,
		},
		{
			name:           "unauthorized on invalid basic auth credentials",
			method:         http.MethodPost,
			body:           `{"node_id":"node-123","temperature":21.5}`,
			authUser:       "wronguser",
			authPass:       "wrongpassword",
			setup:          func(client *mocks.MockReadingDatabaseServiceClient) {},
			wantStatusCode: http.StatusUnauthorized,
		},
		{
			name:           "method not allowed for GET request",
			method:         http.MethodGet,
			body:           `{"node_id":"node-123"}`,
			setup:          func(client *mocks.MockReadingDatabaseServiceClient) {},
			wantStatusCode: http.StatusMethodNotAllowed,
		},
		{
			name:           "bad request on empty body",
			method:         http.MethodPost,
			body:           ``,
			setup:          func(client *mocks.MockReadingDatabaseServiceClient) {},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "bad request on malformed json",
			method:         http.MethodPost,
			body:           `{not-valid-json`,
			setup:          func(client *mocks.MockReadingDatabaseServiceClient) {},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "bad request on json array payload",
			method:         http.MethodPost,
			body:           `[]`,
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

	for index, testCase := range testCases {
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

			handler, err := New(HandlerParams{
				Logger:     zap.NewNop(),
				Controller: controller,
				Auth:       auth,
			})
			if err != nil {
				t.Fatalf("New() unexpected error = %v", err)
			}
			if index == 0 && handler.Pattern() != "/reading" {
				t.Fatalf("Pattern() = %q, want /reading", handler.Pattern())
			}

			request := httptest.NewRequest(testCase.method, "/reading", bytes.NewBufferString(testCase.body))
			if !testCase.omitAuth {
				username := testCase.authUser
				password := testCase.authPass
				if username == "" && password == "" {
					username = "testuser"
					password = "testpassword"
				}
				request.SetBasicAuth(username, password)
			}
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)

			if recorder.Code != testCase.wantStatusCode {
				t.Fatalf("ServeHTTP() status = %d, want %d", recorder.Code, testCase.wantStatusCode)
			}
		})
	}
}
