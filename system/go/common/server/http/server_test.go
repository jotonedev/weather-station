package http

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
)

type staticRoute struct {
	pattern string
	status  int
	body    string
}

func (route *staticRoute) Pattern() string {
	return route.pattern
}

func (route *staticRoute) ServeHTTP(writer http.ResponseWriter, _ *http.Request) {
	writer.WriteHeader(route.status)
	_, _ = writer.Write([]byte(route.body))
}

func reserveFreeLoopbackAddress(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen(NET_PROTOCOL, "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() unexpected error = %v", err)
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("listener.Close() unexpected error = %v", err)
	}
	return address
}

func TestNewHTTPMux(t *testing.T) {
	routes := []Route{
		&staticRoute{
			pattern: "/ping",
			status:  http.StatusOK,
			body:    "pong",
		},
	}
	mux := NewHTTPMux(routes)

	testCases := []struct {
		name       string
		path       string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "registered route returns expected response",
			path:       "/ping",
			wantStatus: http.StatusOK,
			wantBody:   "pong",
		},
		{
			name:       "unknown route returns not found",
			path:       "/unknown",
			wantStatus: http.StatusNotFound,
			wantBody:   "404 page not found\n",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, testCase.path, nil)
			recorder := httptest.NewRecorder()
			mux.ServeHTTP(recorder, request)

			if recorder.Code != testCase.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, testCase.wantStatus)
			}
			if recorder.Body.String() != testCase.wantBody {
				t.Fatalf("body = %q, want %q", recorder.Body.String(), testCase.wantBody)
			}
		})
	}
}

func TestHTTPServerLifecycle(t *testing.T) {
	testCases := []struct {
		name         string
		setupAddress func(t *testing.T) string
		wantStartErr bool
	}{
		{
			name: "starts serves and stops gracefully",
			setupAddress: func(t *testing.T) string {
				return reserveFreeLoopbackAddress(t)
			},
		},
		{
			name: "returns error when port already in use",
			setupAddress: func(t *testing.T) string {
				listener, err := net.Listen(NET_PROTOCOL, "127.0.0.1:0")
				if err != nil {
					t.Fatalf("net.Listen() unexpected error = %v", err)
				}
				t.Cleanup(func() {
					_ = listener.Close()
				})
				return listener.Addr().String()
			},
			wantStartErr: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			lifecycle := fxtest.NewLifecycle(t)
			address := testCase.setupAddress(t)
			mux := NewHTTPMux([]Route{
				&staticRoute{
					pattern: "/ping",
					status:  http.StatusOK,
					body:    "pong",
				},
			})

			server := NewHTTPServer(lifecycle, mux, zap.NewNop(), Config{Address: address})
			if server == nil {
				t.Fatal("NewHTTPServer() returned nil")
			}

			startCtx, cancelStart := context.WithTimeout(context.Background(), time.Second)
			defer cancelStart()

			startErr := lifecycle.Start(startCtx)
			if testCase.wantStartErr {
				if startErr == nil {
					t.Fatal("lifecycle.Start() want error, got nil")
				}
				return
			}
			if startErr != nil {
				t.Fatalf("lifecycle.Start() unexpected error = %v", startErr)
			}

			client := &http.Client{Timeout: time.Second}
			targetURL := "http://" + address + "/ping"
			response, err := client.Get(targetURL)
			if err != nil {
				t.Fatalf("Get() unexpected error = %v", err)
			}
			body, readErr := io.ReadAll(response.Body)
			_ = response.Body.Close()
			if readErr != nil {
				t.Fatalf("io.ReadAll() unexpected error = %v", readErr)
			}
			if response.StatusCode != http.StatusOK {
				t.Fatalf("StatusCode = %d, want %d", response.StatusCode, http.StatusOK)
			}
			if string(body) != "pong" {
				t.Fatalf("body = %q, want pong", string(body))
			}

			stopCtx, cancelStop := context.WithTimeout(context.Background(), time.Second)
			defer cancelStop()
			if err := lifecycle.Stop(stopCtx); err != nil {
				t.Fatalf("lifecycle.Stop() unexpected error = %v", err)
			}

			if closedResp, err := client.Get(targetURL); err == nil {
				_ = closedResp.Body.Close()
				t.Fatal("Get() after stop want error, got nil")
			}
		})
	}
}
