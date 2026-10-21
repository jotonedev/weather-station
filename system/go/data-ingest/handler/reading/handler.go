package reading

import (
	"encoding/json"
	"errors"
	"net/http"

	"go.uber.org/fx"
	"go.uber.org/zap"

	"weather-station/common/auth/basic"
	"weather-station/data-ingest/controller/weatherstation"
)

// Handler handles incoming HTTP requests for weather readings.
type Handler struct {
	log        *zap.Logger
	controller *weatherstation.Controller
	auth       *basic.Authenticator
}

// HandlerParams defines dependencies for Handler.
type HandlerParams struct {
	fx.In

	Logger     *zap.Logger
	Controller *weatherstation.Controller
	Auth       *basic.Authenticator
}

// New creates a new reading Handler instance.
func New(params HandlerParams) (*Handler, error) {
	return &Handler{
		log:        params.Logger,
		controller: params.Controller,
		auth:       params.Auth,
	}, nil
}

// Pattern returns the routing pattern for this handler.
func (*Handler) Pattern() string {
	return "/reading"
}

// ServeHTTP parses incoming weather readings and forwards them to the controller.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !h.auth.RequireAuth(w, r) {
		return
	}

	var reading weatherstation.Reading
	if err := json.NewDecoder(r.Body).Decode(&reading); err != nil {
		http.Error(w, "invalid json payload", http.StatusBadRequest)
		return
	}

	if err := h.controller.StoreReading(r.Context(), reading); err != nil {
		if errors.Is(err, weatherstation.ErrMissingNodeID) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		h.log.Error("Failed to store reading", zap.Error(err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
