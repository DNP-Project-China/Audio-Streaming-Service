package routes

import (
	"encoding/json"
	"net/http"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

func (h *HealthHandler) Pattern() string {
	return "/health"
}

func (h *HealthHandler) Method() string {
	return http.MethodGet
}

func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(NewHealthResponse()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
