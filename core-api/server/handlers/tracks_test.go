package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/DNP-Project-China/Audio-Streaming-Service/core-api/repositories"
	"github.com/DNP-Project-China/Audio-Streaming-Service/core-api/server"
	"github.com/DNP-Project-China/Audio-Streaming-Service/core-api/storage"
	"github.com/DNP-Project-China/Audio-Streaming-Service/core-api/usecases"
	"github.com/jackc/pgx/v5/pgtype"
)

type trackItemResp struct {
	TrackID          string `json:"track_id"`
	Artist           string `json:"artist"`
	Title            string `json:"title"`
	OriginalFilename string `json:"original_filename"`
	OriginalSize     int64  `json:"original_size"`
	Status           string `json:"status"`
	UploadedAt       string `json:"uploaded_at"`
}

type tracksResp struct {
	Items []trackItemResp `json:"items"`
	Total int             `json:"total"`
}

func TestTracks_OneReadyTrack(t *testing.T) {
	cfg, err := server.NewConfig()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	ctx := context.Background()
	pool, err := newTestPool(ctx, cfg)
	if err != nil {
		t.Fatalf("create db pool: %v", err)
	}
	defer pool.Close()

	queries := repositories.New(pool)
	s3store, err := storage.NewS3Storage(cfg)
	if err != nil {
		t.Fatalf("new s3 storage: %v", err)
	}
	tracksUC := usecases.NewTrackStorage(s3store)
	pub := &spyPublisher{}
	uploadHandler := NewUploadHandler(queries, tracksUC, pub)
	h := NewTracksHandler(queries)

	fixture, err := os.ReadFile(filepath.Join("testdata", "testfile.mp3"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	createdTrackID, err := uploadFixtureTrack(uploadHandler, "One Artist", fmt.Sprintf("One Song %d", time.Now().UnixNano()), "one-song.mp3", fixture)
	if err != nil {
		t.Fatalf("upload fixture track: %v", err)
	}

	var trackUUID pgtype.UUID
	if err := trackUUID.Scan(createdTrackID); err != nil {
		t.Fatalf("parse track uuid: %v", err)
	}

	track, err := queries.MarkTrackReady(ctx, repositories.MarkTrackReadyParams{
		ID:             trackUUID,
		HlsPlaylistKey: pgtype.Text{String: "hls/one-song.m3u8", Valid: true},
	})
	if err != nil {
		t.Fatalf("mark track ready: %v", err)
	}

	t.Cleanup(func() {
		cleanupTrackAndObject(ctx, queries, s3store, track.ID)
	})

	req := httptest.NewRequest(http.MethodGet, "/tracks", nil)
	res := httptest.NewRecorder()

	h.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", res.Code, res.Body.String())
	}

	var out tracksResp
	if err := json.Unmarshal(res.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if out.Total < 1 {
		t.Fatalf("expected at least one ready track, got total=%d", out.Total)
	}

	found := false
	for _, item := range out.Items {
		if item.TrackID == track.ID.String() {
			found = true
			if item.OriginalFilename != "one-song.mp3" {
				t.Fatalf("filename mismatch: want=one-song.mp3 got=%s", item.OriginalFilename)
			}
			if item.Status != "ready" {
				t.Fatalf("status mismatch: want=ready got=%s", item.Status)
			}
		}
	}

	if !found {
		t.Fatalf("created track not found in /tracks response")
	}
}

func TestTracks_ThreeReadyTracksNewestFirst(t *testing.T) {
	cfg, err := server.NewConfig()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	ctx := context.Background()
	pool, err := newTestPool(ctx, cfg)
	if err != nil {
		t.Fatalf("create db pool: %v", err)
	}
	defer pool.Close()

	queries := repositories.New(pool)
	s3store, err := storage.NewS3Storage(cfg)
	if err != nil {
		t.Fatalf("new s3 storage: %v", err)
	}
	tracksUC := usecases.NewTrackStorage(s3store)
	pub := &spyPublisher{}
	uploadHandler := NewUploadHandler(queries, tracksUC, pub)
	h := NewTracksHandler(queries)

	fixture, err := os.ReadFile(filepath.Join("testdata", "testfile.mp3"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	created := make([]repositories.Track, 0, 3)
	for i := 1; i <= 3; i++ {
		trackID, createErr := uploadFixtureTrack(
			uploadHandler,
			fmt.Sprintf("Artist-%d", i),
			fmt.Sprintf("Song-%d-%d", i, time.Now().UnixNano()),
			fmt.Sprintf("song-%d.mp3", i),
			fixture,
		)
		if createErr != nil {
			t.Fatalf("upload track %d: %v", i, createErr)
		}

		var trackUUID pgtype.UUID
		if err := trackUUID.Scan(trackID); err != nil {
			t.Fatalf("parse track uuid %d: %v", i, err)
		}

		track, markErr := queries.MarkTrackReady(ctx, repositories.MarkTrackReadyParams{
			ID:             trackUUID,
			HlsPlaylistKey: pgtype.Text{String: fmt.Sprintf("hls/song-%d.m3u8", i), Valid: true},
		})
		if markErr != nil {
			t.Fatalf("mark track %d ready: %v", i, markErr)
		}

		created = append(created, track)
		time.Sleep(5 * time.Millisecond)
	}
	t.Cleanup(func() {
		for _, track := range created {
			cleanupTrackAndObject(ctx, queries, s3store, track.ID)
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/tracks?status=ready", nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", res.Code, res.Body.String())
	}

	var out tracksResp
	if err := json.Unmarshal(res.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	positions := map[string]int{}
	for idx, item := range out.Items {
		positions[item.TrackID] = idx
	}

	for _, track := range created {
		if _, ok := positions[track.ID.String()]; !ok {
			t.Fatalf("missing created track id=%s in response", track.ID.String())
		}
	}

	first := positions[created[0].ID.String()]
	second := positions[created[1].ID.String()]
	third := positions[created[2].ID.String()]
	if !(third < second && second < first) {
		t.Fatalf("expected newest-first order for inserted tracks, got positions first=%d second=%d third=%d", first, second, third)
	}
}

func TestTracks_InvalidStatus_Returns400(t *testing.T) {
	cfg, err := server.NewConfig()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	ctx := context.Background()
	pool, err := newTestPool(ctx, cfg)
	if err != nil {
		t.Fatalf("create db pool: %v", err)
	}
	defer pool.Close()

	h := NewTracksHandler(repositories.New(pool))
	req := httptest.NewRequest(http.MethodGet, "/tracks?status=wrong", nil)
	res := httptest.NewRecorder()

	h.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d, body=%s", res.Code, res.Body.String())
	}
}

func uploadFixtureTrack(uploadHandler *UploadHandler, artist string, title string, filename string, body []byte) (string, error) {
	payload := &bytes.Buffer{}
	mw := multipart.NewWriter(payload)
	_ = mw.WriteField("artist", artist)
	_ = mw.WriteField("title", title)

	fw, err := mw.CreateFormFile("file", filename)
	if err != nil {
		return "", fmt.Errorf("create form file: %w", err)
	}
	if _, err := fw.Write(body); err != nil {
		return "", fmt.Errorf("write payload: %w", err)
	}
	if err := mw.Close(); err != nil {
		return "", fmt.Errorf("close multipart writer: %w", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/upload", payload)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	res := httptest.NewRecorder()
	uploadHandler.ServeHTTP(res, req)

	if res.Code != http.StatusCreated {
		return "", fmt.Errorf("upload status=%d body=%s", res.Code, res.Body.String())
	}

	var out uploadResp
	if err := json.Unmarshal(res.Body.Bytes(), &out); err != nil {
		return "", fmt.Errorf("decode upload response: %w", err)
	}

	return out.TrackID, nil
}

func cleanupTrackAndObject(ctx context.Context, queries *repositories.Queries, s3store *storage.S3Storage, trackID pgtype.UUID) {
	track, err := queries.GetTrackByID(ctx, trackID)
	if err == nil {
		_ = s3store.DeleteObject(ctx, track.OriginalObjectKey)
	}
	_ = queries.DeleteTrackByID(ctx, trackID)
}
