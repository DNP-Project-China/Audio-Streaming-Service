package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/DNP-Project-China/Audio-Streaming-Service/core-api/events"
	"github.com/DNP-Project-China/Audio-Streaming-Service/core-api/repositories"
	"github.com/DNP-Project-China/Audio-Streaming-Service/core-api/server"
	"github.com/DNP-Project-China/Audio-Streaming-Service/core-api/usecases"
)

// Handler for uploading new tracks
type UploadHandler struct {
	queries *repositories.Queries
	tracks  *usecases.TrackStorage
	jobs    events.TranscodePublisher
	cfg     *server.Config
}

// Response for successful track upload
type TrackUploadResponse struct {
	TrackID          string `json:"track_id"`
	Artist           string `json:"artist"`
	Title            string `json:"title"`
	OriginalFilename string `json:"original_filename"`
	Status           string `json:"status"`
	UploadedAt       string `json:"uploaded_at"`
}

// Error response format
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// DI constructor for UploadHandler
func NewUploadHandler(cfg *server.Config, queries *repositories.Queries, tracks *usecases.TrackStorage, jobs events.TranscodePublisher) *UploadHandler {
	return &UploadHandler{queries: queries, tracks: tracks, jobs: jobs, cfg: cfg}
}

// Route pattern for this handler
func (h *UploadHandler) Pattern() string {
	return "/upload"
}

// HTTP method for this handler
func (h *UploadHandler) Method() string {
	return http.MethodPost
}

func (h *UploadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Limit request body size to prevent abuse
	r.Body = http.MaxBytesReader(w, r.Body, h.cfg.UploadMaxBytes)

	if err := r.ParseMultipartForm(h.cfg.UploadMaxBytes); err != nil {
		h.respondError(w, http.StatusRequestEntityTooLarge, "payload_too_large", "file exceeds max size")
		return
	}

	// Validate form fields
	artist := strings.TrimSpace(r.FormValue("artist"))
	title := strings.TrimSpace(r.FormValue("title"))
	if artist == "" || title == "" {
		h.respondError(w, http.StatusBadRequest, "invalid_request", "artist and title are required")
		return
	}
	if len(artist) > 255 || len(title) > 255 {
		h.respondError(w, http.StatusBadRequest, "invalid_request", "artist/title length must be <= 255")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid_request", "file is required")
		return
	}
	defer file.Close()

	if !isSupportedAudio(header.Filename) {
		h.respondError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "unsupported audio format")
		return
	}

	body, err := io.ReadAll(file)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid_request", "failed to read uploaded file")
		return
	}

	// Generate a unique object ID for storing the original file
	objectID, err := newObjectID()
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "internal_error", "failed to generate object id")
		return
	}

	// Store the original file
	stored, err := h.tracks.PutOriginal(context.Background(), objectID, header.Filename, body, header.Header.Get("Content-Type"))
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "internal_error", "failed to store uploaded file")
		return
	}

	// Create track record in database with status "pending"
	track, err := h.queries.CreateTrack(context.Background(), repositories.CreateTrackParams{
		Artist:            artist,
		Title:             title,
		OriginalFilename:  header.Filename,
		OriginalObjectKey: stored.Key,
		OriginalSize:      stored.Size,
		Status:            repositories.TrackStatusPending,
	})
	if err != nil {
		_ = h.tracks.Delete(context.Background(), stored.Key)
		h.respondError(w, http.StatusInternalServerError, "internal_error", "failed to create track metadata")
		return
	}

	// Publish transcode job to Kafka
	if err := h.jobs.PublishCreated(context.Background(), track.ID.String(), track.OriginalObjectKey, h.cfg.TranscodeJobPriority); err != nil {
		_ = h.queries.DeleteTrackByID(context.Background(), track.ID)
		_ = h.tracks.Delete(context.Background(), stored.Key)
		h.respondError(w, http.StatusInternalServerError, "internal_error", "failed to enqueue transcode job")
		return
	}

	// Respond with the created track info
	response := TrackUploadResponse{
		TrackID:          track.ID.String(),
		Artist:           track.Artist,
		Title:            track.Title,
		OriginalFilename: track.OriginalFilename,
		Status:           string(track.Status),
		UploadedAt:       track.UploadedAt.Time.UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(response)
}

// Helper method to respond with JSON error messages
func (h *UploadHandler) respondError(w http.ResponseWriter, status int, code string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorResponse{Error: code, Message: message})
}

// Check if the uploaded file has a supported audio extension
func isSupportedAudio(filename string) bool {
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(filename)))
	if ext == "" {
		return false
	}

	supported := map[string]struct{}{
		".mp3":  {},
		".flac": {},
		".wav":  {},
	}

	_, ok := supported[ext]
	return ok
}

// Generate a random 16-byte hex string to use as an object ID
func newObjectID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}
