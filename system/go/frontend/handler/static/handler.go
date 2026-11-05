package static

import (
	"errors"
	"net/http"
	"strings"

	"go.uber.org/fx"
	"go.uber.org/zap"

	staticctrl "weather-station/frontend/controller/static"
)

const routePrefix = "/static/"

// Handler serves static CSS, font, and license assets under /static/.
type Handler struct {
	log              *zap.Logger
	staticController *staticctrl.Controller
}

// HandlerParams defines dependencies for Handler.
type HandlerParams struct {
	fx.In

	Logger           *zap.Logger
	StaticController *staticctrl.Controller
}

// New creates a new static Handler instance.
func New(params HandlerParams) (*Handler, error) {
	return &Handler{
		log:              params.Logger,
		staticController: params.StaticController,
	}, nil
}

// Pattern returns the routing pattern for static assets.
func (*Handler) Pattern() string {
	return routePrefix
}

// ServeHTTP handles requests for embedded static files.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !strings.HasPrefix(r.URL.Path, routePrefix) {
		http.NotFound(w, r)
		return
	}

	relPath := strings.TrimPrefix(r.URL.Path, routePrefix)
	asset, err := h.staticController.GetAsset(relPath)
	if err != nil {
		if errors.Is(err, staticctrl.ErrInvalidAssetPath) || errors.Is(err, staticctrl.ErrAssetNotFound) {
			http.NotFound(w, r)
			return
		}
		h.log.Error("Failed to read static asset", zap.String("path", relPath), zap.Error(err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("ETag", asset.ETag)
	w.Header().Set("Cache-Control", asset.CacheControl)
	w.Header().Set("Vary", "Accept-Encoding")

	if matchETag(r.Header.Get("If-None-Match"), asset.ETag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Set("Content-Type", asset.ContentType)
	w.Header().Set("X-Content-Type-Options", "nosniff")

	body := asset.Data
	if len(asset.GzipData) > 0 && acceptsGzip(r.Header.Get("Accept-Encoding")) {
		w.Header().Set("Content-Encoding", "gzip")
		body = asset.GzipData
	}

	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodGet {
		_, _ = w.Write(body)
	}
}

func matchETag(clientHeader, serverETag string) bool {
	if clientHeader == "" {
		return false
	}
	if clientHeader == "*" {
		return true
	}
	serverClean := strings.TrimPrefix(serverETag, "W/")
	for _, rawTag := range strings.Split(clientHeader, ",") {
		tag := strings.TrimSpace(rawTag)
		tag = strings.TrimPrefix(tag, "W/")
		if tag == serverClean {
			return true
		}
	}
	return false
}

func acceptsGzip(header string) bool {
	if header == "" {
		return false
	}
	for _, rawEnc := range strings.Split(header, ",") {
		enc := strings.TrimSpace(rawEnc)
		if enc == "*" || strings.HasPrefix(enc, "gzip") {
			if strings.Contains(enc, ";") {
				parts := strings.Split(enc, ";")
				for _, p := range parts[1:] {
					p = strings.TrimSpace(p)
					if p == "q=0" || p == "q=0.0" || p == "q=0.00" || p == "q=0.000" {
						return false
					}
				}
			}
			return true
		}
	}
	return false
}
