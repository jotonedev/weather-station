package website

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	homepage "weather-station/frontend/controller/homepage"
	"weather-station/frontend/controller/weatherdata"
	"weather-station/frontend/gateway/database"
	protodatabasev1 "weather-station/gen/go/weather/database/v1"
)

type fakeReadingClient struct {
	protodatabasev1.ReadingDatabaseServiceClient
}

func (f *fakeReadingClient) GetLatestReadings(ctx context.Context, in *protodatabasev1.GetLatestReadingsRequest, opts ...grpc.CallOption) (*protodatabasev1.GetLatestReadingsResponse, error) {
	temp := float32(20.0)
	return &protodatabasev1.GetLatestReadingsResponse{
		Readings: []*protodatabasev1.Reading{
			{NodeId: in.NodeId, Temperature: &temp},
		},
	}, nil
}

func TestWebsiteHandlerServeHTTP(t *testing.T) {
	testCases := []struct {
		name           string
		method         string
		url            string
		wantStatusCode int
		wantContains   string
	}{
		{
			name:           "success serving homepage on GET",
			method:         http.MethodGet,
			url:            "/",
			wantStatusCode: http.StatusOK,
			wantContains:   "Weather Station",
		},
		{
			name:           "success querying specific station via node_id query param",
			method:         http.MethodGet,
			url:            "/?node_id=station-omega",
			wantStatusCode: http.StatusOK,
			wantContains:   "station-omega",
		},
		{
			name:           "method not allowed for POST",
			method:         http.MethodPost,
			url:            "/",
			wantStatusCode: http.StatusMethodNotAllowed,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			dbGateway, err := database.New(database.GatewayParams{
				Logger:        zap.NewNop(),
				Config:        database.Config{Address: "127.0.0.1:8090"},
				ReadingClient: &fakeReadingClient{},
			})
			if err != nil {
				t.Fatalf("database.New() unexpected error = %v", err)
			}

			dataCtrl, err := weatherdata.New(weatherdata.ControllerParams{
				Logger:   zap.NewNop(),
				Database: dbGateway,
			})
			if err != nil {
				t.Fatalf("weatherdata.New() unexpected error = %v", err)
			}

			templateCtrl, err := homepage.New(homepage.ControllerParams{
				Logger:      zap.NewNop(),
				WeatherData: dataCtrl,
			})
			if err != nil {
				t.Fatalf("homepage.New() unexpected error = %v", err)
			}

			handler, err := New(HandlerParams{
				Logger:             zap.NewNop(),
				HomepageController: templateCtrl,
			})
			if err != nil {
				t.Fatalf("New() unexpected error = %v", err)
			}

			request := httptest.NewRequest(testCase.method, testCase.url, nil)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)

			if recorder.Code != testCase.wantStatusCode {
				t.Fatalf("ServeHTTP() status = %d, want %d", recorder.Code, testCase.wantStatusCode)
			}

			if testCase.wantContains != "" {
				body := recorder.Body.String()
				if !strings.Contains(body, testCase.wantContains) {
					t.Fatalf("body does not contain %q", testCase.wantContains)
				}
			}
		})
	}
}

func TestHandlerPattern(t *testing.T) {
	handler := &Handler{}
	if pattern := handler.Pattern(); pattern != "/" {
		t.Fatalf("Pattern() = %q, want /", pattern)
	}
}
