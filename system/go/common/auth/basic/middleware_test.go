package basic

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewConfig(t *testing.T) {
	tests := []struct {
		name         string
		envUser      string
		envPass      string
		wantUsername string
		wantPassword string
	}{
		{
			name:         "reads environment variables",
			envUser:      "station-admin",
			envPass:      "secret-pass",
			wantUsername: "station-admin",
			wantPassword: "secret-pass",
		},
		{
			name:         "defaults to empty when unset",
			envUser:      "",
			envPass:      "",
			wantUsername: "",
			wantPassword: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("HTTP_AUTH_USERNAME", tc.envUser)
			t.Setenv("HTTP_AUTH_PASSWORD", tc.envPass)

			cfg := NewConfig()
			if cfg.Username != tc.wantUsername {
				t.Fatalf("got username %q, want %q", cfg.Username, tc.wantUsername)
			}
			if cfg.Password != tc.wantPassword {
				t.Fatalf("got password %q, want %q", cfg.Password, tc.wantPassword)
			}
		})
	}
}

func TestAuthenticator_RequireAuth(t *testing.T) {
	tests := []struct {
		name       string
		cfg        Config
		reqUser    string
		reqPass    string
		withHeader bool
		wantOK     bool
		wantStatus int
	}{
		{
			name: "valid credentials succeed",
			cfg: Config{
				Username: "admin",
				Password: "password123",
			},
			reqUser:    "admin",
			reqPass:    "password123",
			withHeader: true,
			wantOK:     true,
			wantStatus: http.StatusOK,
		},
		{
			name: "wrong password fails with 401",
			cfg: Config{
				Username: "admin",
				Password: "password123",
			},
			reqUser:    "admin",
			reqPass:    "wrong",
			withHeader: true,
			wantOK:     false,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "wrong username fails with 401",
			cfg: Config{
				Username: "admin",
				Password: "password123",
			},
			reqUser:    "other",
			reqPass:    "password123",
			withHeader: true,
			wantOK:     false,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "missing authorization header fails with 401",
			cfg: Config{
				Username: "admin",
				Password: "password123",
			},
			withHeader: false,
			wantOK:     false,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "unconfigured server credentials reject all requests",
			cfg: Config{
				Username: "",
				Password: "",
			},
			reqUser:    "admin",
			reqPass:    "password123",
			withHeader: true,
			wantOK:     false,
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			auth := NewAuthenticator(tc.cfg)
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if tc.withHeader {
				req.SetBasicAuth(tc.reqUser, tc.reqPass)
			}
			rec := httptest.NewRecorder()

			gotOK := auth.RequireAuth(rec, req)
			if gotOK != tc.wantOK {
				t.Fatalf("RequireAuth() = %v, want %v", gotOK, tc.wantOK)
			}
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if !tc.wantOK && rec.Header().Get("WWW-Authenticate") == "" {
				t.Fatal("expected WWW-Authenticate header on 401 response")
			}
		})
	}
}
