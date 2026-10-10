package reading

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"go.uber.org/fx"
	"go.uber.org/zap"

	"weather-station/data-ingest/controller/weatherstation"
)

// BatchHandler handles incoming HTTP requests containing multiple weather readings.
type BatchHandler struct {
	log        *zap.Logger
	controller *weatherstation.Controller
}

// BatchHandlerParams defines dependencies for BatchHandler.
type BatchHandlerParams struct {
	fx.In

	Logger     *zap.Logger
	Controller *weatherstation.Controller
}

// NewBatch creates a new BatchHandler instance.
func NewBatch(params BatchHandlerParams) (*BatchHandler, error) {
	return &BatchHandler{
		log:        params.Logger,
		controller: params.Controller,
	}, nil
}

// Pattern returns the routing pattern for batch readings.
func (*BatchHandler) Pattern() string {
	return "/readings"
}

// ServeHTTP parses a batch of weather readings and forwards them to the controller.
func (h *BatchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	trimmed := bytes.TrimSpace(bodyBytes)
	if len(trimmed) == 0 {
		http.Error(w, "empty request body", http.StatusBadRequest)
		return
	}

	var readings []weatherstation.Reading
	if trimmed[0] == '[' {
		if err := json.Unmarshal(trimmed, &readings); err != nil {
			http.Error(w, "invalid json array", http.StatusBadRequest)
			return
		}
	} else {
		var payload struct {
			Readings []weatherstation.Reading `json:"readings"`
		}
		if err := json.Unmarshal(trimmed, &payload); err != nil {
			http.Error(w, "invalid json payload", http.StatusBadRequest)
			return
		}
		readings = payload.Readings
	}

	if len(readings) == 0 {
		http.Error(w, "no readings provided", http.StatusBadRequest)
		return
	}

	if err := h.controller.StoreReadings(r.Context(), readings); err != nil {
		if errors.Is(err, weatherstation.ErrMissingNodeID) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		h.log.Error("Failed to store readings batch", zap.Error(err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
