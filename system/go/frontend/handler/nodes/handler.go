package nodes

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"go.uber.org/fx"
	"go.uber.org/zap"

	"weather-station/common/auth/basic"
	"weather-station/frontend/controller/nodeadmin"
)

// Handler serves the HTTP Basic Auth protected node administration interface.
type Handler struct {
	log        *zap.Logger
	controller *nodeadmin.Controller
	auth       *basic.Authenticator
}

// HandlerParams defines dependencies for Handler.
type HandlerParams struct {
	fx.In

	Logger     *zap.Logger
	Controller *nodeadmin.Controller
	Auth       *basic.Authenticator
}

// New creates a new nodes Handler instance.
func New(params HandlerParams) (*Handler, error) {
	return &Handler{
		log:        params.Logger,
		controller: params.Controller,
		auth:       params.Auth,
	}, nil
}

// Pattern returns the routing pattern for node management.
func (*Handler) Pattern() string {
	return "/nodes"
}

// ServeHTTP handles GET and POST requests for managing station nodes.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !h.auth.RequireAuth(w, r) {
		return
	}

	var statusMsg string
	var errorMsg string

	switch r.Method {
	case http.MethodGet:
		// Render current state.
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form data", http.StatusBadRequest)
			return
		}
		action := strings.TrimSpace(r.FormValue("action"))
		switch action {
		case "create":
			input, err := parseCreateForm(r)
			if err != nil {
				errorMsg = err.Error()
			} else if err := h.controller.SaveNode(r.Context(), input); err != nil {
				errorMsg = err.Error()
			} else {
				statusMsg = fmt.Sprintf("Saved node %q.", strings.TrimSpace(input.NodeID))
			}
		case "delete":
			nodeID := strings.TrimSpace(r.FormValue("node_id"))
			if err := h.controller.DeleteNode(r.Context(), nodeID); err != nil {
				errorMsg = err.Error()
			} else {
				statusMsg = fmt.Sprintf("Deleted node %q and all associated data.", nodeID)
			}
		default:
			http.Error(w, "invalid action", http.StatusBadRequest)
			return
		}
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	htmlBytes, err := h.controller.Render(r.Context(), statusMsg, errorMsg)
	if err != nil {
		h.log.Error("Failed to render nodes page", zap.Error(err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(htmlBytes)
}

func parseCreateForm(r *http.Request) (nodeadmin.NodeFormInput, error) {
	lat, err := strconv.ParseFloat(strings.TrimSpace(r.FormValue("latitude")), 64)
	if err != nil {
		return nodeadmin.NodeFormInput{}, fmt.Errorf("invalid latitude value")
	}
	lon, err := strconv.ParseFloat(strings.TrimSpace(r.FormValue("longitude")), 64)
	if err != nil {
		return nodeadmin.NodeFormInput{}, fmt.Errorf("invalid longitude value")
	}
	elev, err := strconv.ParseFloat(strings.TrimSpace(r.FormValue("elevation_m")), 64)
	if err != nil {
		return nodeadmin.NodeFormInput{}, fmt.Errorf("invalid elevation value")
	}

	return nodeadmin.NodeFormInput{
		NodeID:       r.FormValue("node_id"),
		LocationName: r.FormValue("location_name"),
		Latitude:     lat,
		Longitude:    lon,
		ElevationM:   elev,
	}, nil
}
