package routes

import (
	"encoding/json"
	"net/http"
)

type HealthHandler struct{}

// DI constructor for HealthHandler
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// Response format for health check endpoint
func (h *HealthHandler) Pattern() string {
	return "/health"
}

// HTTP get method for health check endpoint
func (h *HealthHandler) Method() string {
	return http.MethodGet
}

// Simple healthcheck
func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(NewHealthResponse()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
