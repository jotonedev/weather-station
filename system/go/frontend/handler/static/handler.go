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
	data, contentType, err := h.staticController.ReadAsset(relPath)
	if err != nil {
		if errors.Is(err, staticctrl.ErrInvalidAssetPath) || errors.Is(err, staticctrl.ErrAssetNotFound) {
			http.NotFound(w, r)
			return
		}
		h.log.Error("Failed to read static asset", zap.String("path", relPath), zap.Error(err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodGet {
		_, _ = w.Write(data)
	}
}
