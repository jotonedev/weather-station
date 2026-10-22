package website

import (
	"net/http"

	"go.uber.org/fx"
	"go.uber.org/zap"

	homepage "weather-station/frontend/controller/homepage"
)

// Handler serves the main website HTTP interface.
type Handler struct {
	log                *zap.Logger
	homepageController *homepage.Controller
}

// HandlerParams defines dependencies for Handler.
type HandlerParams struct {
	fx.In

	Logger             *zap.Logger
	HomepageController *homepage.Controller
}

// New creates a new website Handler instance.
func New(params HandlerParams) (*Handler, error) {
	return &Handler{
		log:                params.Logger,
		homepageController: params.HomepageController,
	}, nil
}

// Pattern returns the routing pattern for the website.
func (*Handler) Pattern() string {
	return "/"
}

// ServeHTTP handles requests for the weather dashboard homepage.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	requestedNodeID := r.URL.Query().Get("node_id")
	htmlBytes, err := h.homepageController.Render(r.Context(), requestedNodeID)
	if err != nil {
		h.log.Error("Failed to render homepage", zap.Error(err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(htmlBytes)
}
