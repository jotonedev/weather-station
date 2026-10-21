package nodes

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"weather-station/common/auth/basic"
	"weather-station/frontend/controller/nodeadmin"
	"weather-station/frontend/gateway/database"
	"weather-station/frontend/internal/mocks"
	protodatabasev1 "weather-station/gen/go/weather/database/v1"
)

func TestHandlerServeHTTP(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		form       url.Values
		omitAuth   bool
		setup      func(client *mocks.MockNodeDatabaseServiceClient)
		wantStatus int
		wantBody   string
	}{
		{
			name:       "unauthorized without credentials",
			method:     http.MethodGet,
			omitAuth:   true,
			setup:      func(client *mocks.MockNodeDatabaseServiceClient) {},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:   "GET renders registered nodes",
			method: http.MethodGet,
			setup: func(client *mocks.MockNodeDatabaseServiceClient) {
				client.EXPECT().
					ListNodeCoordinates(gomock.Any(), gomock.Any()).
					Return(&protodatabasev1.ListNodeCoordinatesResponse{
						Nodes: []*protodatabasev1.NodeCoordinates{
							{NodeId: "station-alpha", LocationName: "Hilltop", Latitude: 45.0, Longitude: 7.0, ElevationM: 300},
						},
					}, nil)
			},
			wantStatus: http.StatusOK,
			wantBody:   "station-alpha",
		},
		{
			name:   "POST create saves node and renders page",
			method: http.MethodPost,
			form: url.Values{
				"action":        {"create"},
				"node_id":       {"station-new"},
				"location_name": {"Roof"},
				"latitude":      {"45.1"},
				"longitude":     {"7.7"},
				"elevation_m":   {"250"},
			},
			setup: func(client *mocks.MockNodeDatabaseServiceClient) {
				client.EXPECT().
					CreateNode(gomock.Any(), gomock.Any()).
					Return(&protodatabasev1.CreateNodeResponse{}, nil)
				client.EXPECT().
					ListNodeCoordinates(gomock.Any(), gomock.Any()).
					Return(&protodatabasev1.ListNodeCoordinatesResponse{}, nil)
			},
			wantStatus: http.StatusOK,
			wantBody:   "Saved node",
		},
		{
			name:   "POST delete removes node and renders page",
			method: http.MethodPost,
			form: url.Values{
				"action":  {"delete"},
				"node_id": {"station-alpha"},
			},
			setup: func(client *mocks.MockNodeDatabaseServiceClient) {
				client.EXPECT().
					DeleteNode(gomock.Any(), &protodatabasev1.DeleteNodeRequest{NodeId: "station-alpha"}).
					Return(&protodatabasev1.DeleteNodeResponse{}, nil)
				client.EXPECT().
					ListNodeCoordinates(gomock.Any(), gomock.Any()).
					Return(&protodatabasev1.ListNodeCoordinatesResponse{}, nil)
			},
			wantStatus: http.StatusOK,
			wantBody:   "Deleted node",
		},
		{
			name:   "POST unknown action returns 400",
			method: http.MethodPost,
			form: url.Values{
				"action": {"invalid"},
			},
			setup:      func(client *mocks.MockNodeDatabaseServiceClient) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "DELETE method returns 405",
			method:     http.MethodDelete,
			setup:      func(client *mocks.MockNodeDatabaseServiceClient) {},
			wantStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			nodeClient := mocks.NewMockNodeDatabaseServiceClient(ctrl)
			tc.setup(nodeClient)

			gw, err := database.New(database.GatewayParams{
				Logger:     zap.NewNop(),
				Config:     database.Config{Address: "127.0.0.1:8090"},
				NodeClient: nodeClient,
			})
			if err != nil {
				t.Fatalf("database.New() error = %v", err)
			}

			controller, err := nodeadmin.New(nodeadmin.ControllerParams{
				Logger:   zap.NewNop(),
				Database: gw,
			})
			if err != nil {
				t.Fatalf("nodeadmin.New() error = %v", err)
			}

			auth := basic.NewAuthenticator(basic.Config{
				Username: "admin",
				Password: "password",
			})

			handler, err := New(HandlerParams{
				Logger:     zap.NewNop(),
				Controller: controller,
				Auth:       auth,
			})
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}

			var bodyReader *strings.Reader
			if tc.form != nil {
				bodyReader = strings.NewReader(tc.form.Encode())
			} else {
				bodyReader = strings.NewReader("")
			}

			req := httptest.NewRequest(tc.method, "/nodes", bodyReader)
			if tc.form != nil {
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			}
			if !tc.omitAuth {
				req.SetBasicAuth("admin", "password")
			}

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if tc.wantBody != "" && !strings.Contains(rec.Body.String(), tc.wantBody) {
				t.Fatalf("body %q does not contain %q", rec.Body.String(), tc.wantBody)
			}
		})
	}
}
