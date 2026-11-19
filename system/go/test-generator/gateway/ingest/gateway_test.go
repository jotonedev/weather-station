package ingest

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestNewConfig(t *testing.T) {
	tests := []struct {
		name     string
		envURL   string
		envUser  string
		envPass  string
		wantURL  string
		wantUser string
	}{
		{
			name:     "default url when env is unset",
			wantURL:  defaultIngestURL,
			wantUser: "",
		},
		{
			name:     "custom url and credentials from env",
			envURL:   "http://localhost:8081/readings",
			envUser:  "admin",
			envPass:  "secret",
			wantURL:  "http://localhost:8081/readings",
			wantUser: "admin",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("INGEST_URL", tc.envURL)
			t.Setenv("HTTP_AUTH_USERNAME", tc.envUser)
			t.Setenv("HTTP_AUTH_PASSWORD", tc.envPass)

			cfg := NewConfig()
			if cfg.URL != tc.wantURL {
				t.Fatalf("URL = %q, want %q", cfg.URL, tc.wantURL)
			}
			if cfg.Username != tc.wantUser {
				t.Fatalf("Username = %q, want %q", cfg.Username, tc.wantUser)
			}
		})
	}
}

func TestGatewaySendReadings(t *testing.T) {
	sample := []ReadingPayload{
		{
			NodeID:      "station-alpha",
			Timestamp:   time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC),
			Temperature: 21.5,
			Humidity:    55.0,
		},
	}

	tests := []struct {
		name       string
		readings   []ReadingPayload
		statusCode int
		wantAuth   bool
		wantErr    error
	}{
		{
			name:       "empty slice returns nil without calling server",
			readings:   nil,
			statusCode: http.StatusOK,
		},
		{
			name:       "valid batch with basic auth succeeds",
			readings:   sample,
			statusCode: http.StatusOK,
			wantAuth:   true,
		},
		{
			name:       "non-2xx status returns ErrUnexpectedStatus",
			readings:   sample,
			statusCode: http.StatusUnauthorized,
			wantAuth:   true,
			wantErr:    ErrUnexpectedStatus,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tc.wantAuth {
					user, pass, ok := r.BasicAuth()
					if !ok || user != "admin" || pass != "secret" {
						w.WriteHeader(http.StatusUnauthorized)
						return
					}
				}
				if len(tc.readings) > 0 {
					if r.Method != http.MethodPost {
						t.Errorf("method = %q, want POST", r.Method)
					}
					if got := r.Header.Get("Content-Type"); got != "application/json" {
						t.Errorf("Content-Type = %q, want application/json", got)
					}
					var body batchRequest
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Errorf("decode request body: %v", err)
					} else if len(body.Readings) != len(tc.readings) {
						t.Errorf("readings count = %d, want %d", len(body.Readings), len(tc.readings))
					} else {
						for index, want := range tc.readings {
							got := body.Readings[index]
							if got.NodeID != want.NodeID || !got.Timestamp.Equal(want.Timestamp) || got.Temperature != want.Temperature {
								t.Errorf("reading %d = %+v, want node_id %q, timestamp %v, temperature %v",
									index, got, want.NodeID, want.Timestamp, want.Temperature)
							}
						}
					}
				}
				w.WriteHeader(tc.statusCode)
			}))
			defer srv.Close()

			gw, err := New(GatewayParams{
				Logger: zap.NewNop(),
				Client: srv.Client(),
				Config: Config{
					URL:      srv.URL,
					Username: "admin",
					Password: "secret",
				},
			})
			if err != nil {
				t.Fatalf("New() unexpected error = %v", err)
			}

			err = gw.SendReadings(context.Background(), tc.readings)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("SendReadings() error = %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("SendReadings() unexpected error = %v", err)
			}
		})
	}
}
