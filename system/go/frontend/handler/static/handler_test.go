package static

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/zap"

	staticctrl "weather-station/frontend/controller/static"
)

func newTestHandler(t *testing.T) *Handler {
	t.Helper()
	ctrl, err := staticctrl.New(staticctrl.ControllerParams{
		Logger: zap.NewNop(),
	})
	if err != nil {
		t.Fatalf("failed to create static controller: %v", err)
	}

	h, err := New(HandlerParams{
		Logger:           zap.NewNop(),
		StaticController: ctrl,
	})
	if err != nil {
		t.Fatalf("failed to create static handler: %v", err)
	}
	return h
}

func decompressGzip(t *testing.T, data []byte) []byte {
	t.Helper()
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
	}
	defer reader.Close()
	decompressed, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("failed to decompress gzip data: %v", err)
	}
	return decompressed
}

func TestHandler_ServeHTTP_SuccessCSS(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/static/css/styles.css", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/css; charset=utf-8" {
		t.Fatalf("expected Content-Type text/css; charset=utf-8, got %q", ct)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "public, max-age=0, must-revalidate" {
		t.Fatalf("expected Cache-Control public, max-age=0, must-revalidate, got %q", cc)
	}
	if vary := rec.Header().Get("Vary"); vary != "Accept-Encoding" {
		t.Fatalf("expected Vary Accept-Encoding, got %q", vary)
	}
	if ce := rec.Header().Get("Content-Encoding"); ce != "" {
		t.Fatalf("expected no Content-Encoding without Accept-Encoding header, got %q", ce)
	}
	if etag := rec.Header().Get("ETag"); etag == "" || !strings.HasPrefix(etag, "W/\"") {
		t.Fatalf("expected weak quoted ETag, got %q", etag)
	}
	if rec.Body.Len() == 0 {
		t.Fatal("expected non-empty response body")
	}
}

func TestHandler_ServeHTTP_GzipCompression(t *testing.T) {
	h := newTestHandler(t)

	uncompressedReq := httptest.NewRequest(http.MethodGet, "/static/css/styles.css", nil)
	uncompressedRec := httptest.NewRecorder()
	h.ServeHTTP(uncompressedRec, uncompressedReq)
	rawBytes := uncompressedRec.Body.Bytes()

	gzipReq := httptest.NewRequest(http.MethodGet, "/static/css/styles.css", nil)
	gzipReq.Header.Set("Accept-Encoding", "gzip, deflate, br")
	gzipRec := httptest.NewRecorder()
	h.ServeHTTP(gzipRec, gzipReq)

	if gzipRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", gzipRec.Code)
	}
	if ce := gzipRec.Header().Get("Content-Encoding"); ce != "gzip" {
		t.Fatalf("expected Content-Encoding gzip, got %q", ce)
	}
	if gzipRec.Body.Len() >= len(rawBytes) {
		t.Fatalf("expected gzipped body to be smaller: raw=%d, gzip=%d", len(rawBytes), gzipRec.Body.Len())
	}
	decompressed := decompressGzip(t, gzipRec.Body.Bytes())
	if !bytes.Equal(decompressed, rawBytes) {
		t.Fatal("decompressed content does not match uncompressed content")
	}
}

func TestHandler_ServeHTTP_GzipDisabledByWeight(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/static/css/styles.css", nil)
	req.Header.Set("Accept-Encoding", "gzip;q=0")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if ce := rec.Header().Get("Content-Encoding"); ce != "" {
		t.Fatalf("expected no Content-Encoding when q=0, got %q", ce)
	}
}

func TestHandler_ServeHTTP_FontNoCompression(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/static/font/weathericons-regular-webfont.woff2", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "font/woff2" {
		t.Fatalf("expected Content-Type font/woff2, got %q", ct)
	}
	if ce := rec.Header().Get("Content-Encoding"); ce != "" {
		t.Fatalf("expected no Content-Encoding for woff2, got %q", ce)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "public, max-age=31536000, immutable" {
		t.Fatalf("expected immutable Cache-Control, got %q", cc)
	}
}

func TestHandler_ServeHTTP_ConditionalRequest_304(t *testing.T) {
	h := newTestHandler(t)

	initialReq := httptest.NewRequest(http.MethodGet, "/static/css/styles.css", nil)
	initialReq.Header.Set("Accept-Encoding", "gzip")
	initialRec := httptest.NewRecorder()
	h.ServeHTTP(initialRec, initialReq)

	etag := initialRec.Header().Get("ETag")
	if etag == "" {
		t.Fatal("expected ETag in initial response")
	}

	rawCleanTag := strings.TrimPrefix(etag, "W/")

	testCases := []struct {
		name        string
		ifNoneMatch string
	}{
		{name: "exact tag", ifNoneMatch: etag},
		{name: "clean tag without W/", ifNoneMatch: rawCleanTag},
		{name: "wildcard", ifNoneMatch: "*"},
		{name: "tag list", ifNoneMatch: "\"other-tag\", " + etag},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			condReq := httptest.NewRequest(http.MethodGet, "/static/css/styles.css", nil)
			condReq.Header.Set("Accept-Encoding", "gzip")
			condReq.Header.Set("If-None-Match", tc.ifNoneMatch)
			condRec := httptest.NewRecorder()

			h.ServeHTTP(condRec, condReq)

			if condRec.Code != http.StatusNotModified {
				t.Fatalf("expected status 304, got %d", condRec.Code)
			}
			if condRec.Header().Get("ETag") != etag {
				t.Fatalf("expected ETag %s, got %s", etag, condRec.Header().Get("ETag"))
			}
			if condRec.Header().Get("Cache-Control") != "public, max-age=0, must-revalidate" {
				t.Fatalf("expected Cache-Control header in 304 response")
			}
			if condRec.Header().Get("Vary") != "Accept-Encoding" {
				t.Fatalf("expected Vary header in 304 response")
			}
			if condRec.Body.Len() != 0 {
				t.Fatalf("expected empty body in 304 response, got %d bytes", condRec.Body.Len())
			}
		})
	}
}

func TestHandler_ServeHTTP_ConditionalRequest_Mismatch(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/static/css/styles.css", nil)
	req.Header.Set("If-None-Match", "\"outdated-etag\"")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 on mismatch, got %d", rec.Code)
	}
	if rec.Body.Len() == 0 {
		t.Fatal("expected response body on mismatch")
	}
}

func TestHandler_ServeHTTP_HEAD(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodHead, "/static/css/styles.css", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if rec.Header().Get("ETag") == "" {
		t.Fatal("expected ETag header on HEAD request")
	}
	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("expected Content-Encoding gzip on HEAD, got %q", rec.Header().Get("Content-Encoding"))
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("expected empty body on HEAD request, got %d bytes", rec.Body.Len())
	}
}

func TestHandler_ServeHTTP_NotFoundAndTraversal(t *testing.T) {
	h := newTestHandler(t)

	paths := []string{
		"/static/css/missing.css",
		"/static/../secret.txt",
		"/static/unknown.ext",
	}

	for _, p := range paths {
		req := httptest.NewRequest(http.MethodGet, p, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected status 404 for %s, got %d", p, rec.Code)
		}
	}
}

func TestHandler_ServeHTTP_MethodNotAllowed(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/static/css/styles.css", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", rec.Code)
	}
}

func TestHandler_Pattern(t *testing.T) {
	h := newTestHandler(t)
	if h.Pattern() != "/static/" {
		t.Fatalf("expected /static/, got %s", h.Pattern())
	}
}
