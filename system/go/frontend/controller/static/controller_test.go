package static

import (
	"errors"
	"strings"
	"testing"

	"go.uber.org/zap"
)

func newTestController(t *testing.T) *Controller {
	t.Helper()
	ctrl, err := New(ControllerParams{
		Logger: zap.NewNop(),
	})
	if err != nil {
		t.Fatalf("failed to create static controller: %v", err)
	}
	return ctrl
}

func TestController_GetAsset(t *testing.T) {
	ctrl := newTestController(t)

	asset, err := ctrl.GetAsset("css/styles.css")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(asset.Data) == 0 {
		t.Fatal("expected non-empty asset data")
	}
	if asset.ContentType != "text/css; charset=utf-8" {
		t.Fatalf("expected CSS content type, got %s", asset.ContentType)
	}
	if !strings.HasPrefix(asset.ETag, "\"") {
		t.Fatalf("expected quoted ETag, got %s", asset.ETag)
	}
	if asset.CacheControl != "public, max-age=0, must-revalidate" {
		t.Fatalf("expected revalidate cache control, got %s", asset.CacheControl)
	}
}

func TestController_GetAsset_Font(t *testing.T) {
	ctrl := newTestController(t)

	asset, err := ctrl.GetAsset("font/weathericons-regular-webfont.woff2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if asset.ContentType != "font/woff2" {
		t.Fatalf("expected font/woff2, got %s", asset.ContentType)
	}
	if asset.CacheControl != "public, max-age=31536000, immutable" {
		t.Fatalf("expected immutable cache control, got %s", asset.CacheControl)
	}
}

func TestController_ReadAsset_Compatibility(t *testing.T) {
	ctrl := newTestController(t)

	data, contentType, err := ctrl.ReadAsset("css/styles.css")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) == 0 || contentType != "text/css; charset=utf-8" {
		t.Fatalf("unexpected ReadAsset result: len=%d, type=%s", len(data), contentType)
	}
}

func TestController_GetAsset_Errors(t *testing.T) {
	ctrl := newTestController(t)

	_, err := ctrl.GetAsset("")
	if !errors.Is(err, ErrInvalidAssetPath) {
		t.Fatalf("expected ErrInvalidAssetPath for empty path, got %v", err)
	}

	_, err = ctrl.GetAsset("../secret.txt")
	if !errors.Is(err, ErrInvalidAssetPath) {
		t.Fatalf("expected ErrInvalidAssetPath for traversal, got %v", err)
	}

	_, err = ctrl.GetAsset("css/missing.css")
	if !errors.Is(err, ErrAssetNotFound) {
		t.Fatalf("expected ErrAssetNotFound for missing file, got %v", err)
	}
}
