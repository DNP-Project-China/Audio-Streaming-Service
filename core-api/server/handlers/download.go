package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/DNP-Project-China/Audio-Streaming-Service/core-api/repositories"
	"github.com/DNP-Project-China/Audio-Streaming-Service/core-api/server"
	"github.com/DNP-Project-China/Audio-Streaming-Service/core-api/usecases"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// Handler for downloading original track files
type DownloadHandler struct {
	queries *repositories.Queries
	tracks  *usecases.TrackStorage
	cfg     *server.Config
}

// Response for download handler
type DownloadURLResponse struct {
	TrackID          string `json:"track_id"`
	OriginalFilename string `json:"original_filename"`
	DownloadURL      string `json:"download_url"`
}

// DI constructor for DownloadHandler
func NewDownloadHandler(cfg *server.Config, queries *repositories.Queries, tracks *usecases.TrackStorage) *DownloadHandler {
	return &DownloadHandler{queries: queries, tracks: tracks, cfg: cfg}
}

// Route pattern for this handler
func (h *DownloadHandler) Pattern() string {
	return "/download/{track_id}"
}

// HTTP method for this handler
func (h *DownloadHandler) Method() string {
	return http.MethodGet
}

func (h *DownloadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Validate body and parameters
	trackIDRaw := mux.Vars(r)["track_id"]
	if trackIDRaw == "" {
		h.respondError(w, http.StatusBadRequest, "invalid_request", "track_id is required")
		return
	}

	var trackID pgtype.UUID
	if err := trackID.Scan(trackIDRaw); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid_request", "invalid track_id")
		return
	}

	// Load track info from database
	track, err := h.queries.GetTrackByID(context.Background(), trackID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.respondError(w, http.StatusNotFound, "not_found", "track not found")
			return
		}

		h.respondError(w, http.StatusInternalServerError, "internal_error", "failed to load track")
		return
	}

	// Generate presigned download URL for the original track file
	url, err := h.tracks.PresignDownload(context.Background(), track.OriginalObjectKey, h.cfg.DownloadURLExpires)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "internal_error", "failed to generate download url")
		return
	}

	// Respond with the download URL and track info
	response := DownloadURLResponse{
		TrackID:          track.ID.String(),
		OriginalFilename: track.OriginalFilename,
		DownloadURL:      url,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

// Helper method to respond with JSON error messages
func (h *DownloadHandler) respondError(w http.ResponseWriter, status int, code string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorResponse{Error: code, Message: message})
}
