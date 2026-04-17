package routes

type HealthResponse struct {
	Status string `json:"status"`
}

func NewHealthResponse() HealthResponse {
	return HealthResponse{Status: "ok"}
}
