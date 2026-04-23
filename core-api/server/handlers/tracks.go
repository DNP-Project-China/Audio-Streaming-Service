package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/DNP-Project-China/Audio-Streaming-Service/core-api/repositories"
)

// Handler for listing tracks with status filter
type TracksHandler struct {
	queries *repositories.Queries
}

// Response item for track list
type TrackListItem struct {
	TrackID          string `json:"track_id"`
	Artist           string `json:"artist"`
	Title            string `json:"title"`
	OriginalFilename string `json:"original_filename"`
	OriginalSize     int64  `json:"original_size"`
	Status           string `json:"status"`
	UploadedAt       string `json:"uploaded_at"`
}

// Response for track list API
type TrackListResponse struct {
	Items []TrackListItem `json:"items"`
	Total int             `json:"total"`
}

// DI constructor for TracksHandler
func NewTracksHandler(queries *repositories.Queries) *TracksHandler {
	return &TracksHandler{queries: queries}
}

// Route pattern for this handler
func (h *TracksHandler) Pattern() string {
	return "/tracks"
}

// HTTP method for this handler
func (h *TracksHandler) Method() string {
	return http.MethodGet
}

func (h *TracksHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Validate query parameters
	statusRaw := strings.TrimSpace(r.URL.Query().Get("status"))
	status, err := parseTrackStatus(statusRaw)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid_request", "invalid status value")
		return
	}

	// Load tracks from database
	tracks, err := h.queries.ListTracksByStatus(context.Background(), status)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "internal_error", "failed to list tracks")
		return
	}

	// Format response items
	items := make([]TrackListItem, 0, len(tracks))
	for _, track := range tracks {
		items = append(items, TrackListItem{
			TrackID:          track.ID.String(),
			Artist:           track.Artist,
			Title:            track.Title,
			OriginalFilename: track.OriginalFilename,
			OriginalSize:     track.OriginalSize,
			Status:           string(track.Status),
			UploadedAt:       track.UploadedAt.Time.UTC().Format(time.RFC3339),
		})
	}

	// Respond with the track list
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(TrackListResponse{Items: items, Total: len(items)})
}

// Helper method to respond with JSON error messages
func (h *TracksHandler) respondError(w http.ResponseWriter, status int, code string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorResponse{Error: code, Message: message})
}

// Parse track status from query parameter, default to "ready" if empty
func parseTrackStatus(statusRaw string) (repositories.TrackStatus, error) {
	if statusRaw == "" {
		return repositories.TrackStatusReady, nil
	}

	switch repositories.TrackStatus(statusRaw) {
	case repositories.TrackStatusPending,
		repositories.TrackStatusProcessing,
		repositories.TrackStatusReady,
		repositories.TrackStatusFailed:
		return repositories.TrackStatus(statusRaw), nil
	default:
		return "", http.ErrNotSupported
	}
}
