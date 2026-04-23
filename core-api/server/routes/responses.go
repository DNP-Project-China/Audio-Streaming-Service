package routes

type HealthResponse struct {
	Status string `json:"status"`
}

// Health check response
func NewHealthResponse() HealthResponse {
	return HealthResponse{Status: "ok"}
}
