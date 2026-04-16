package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DNP-Project-China/Audio-Streaming-Service/core-api/repositories"
	"github.com/DNP-Project-China/Audio-Streaming-Service/core-api/server"
	"github.com/DNP-Project-China/Audio-Streaming-Service/core-api/storage"
	"github.com/DNP-Project-China/Audio-Streaming-Service/core-api/usecases"
	"github.com/gorilla/mux"
)

type downloadResp struct {
	TrackID          string `json:"track_id"`
	OriginalFilename string `json:"original_filename"`
	DownloadURL      string `json:"download_url"`
}

func TestDownload_ReturnsURLAndDownloadsOriginalBytes(t *testing.T) {
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
	trackStore := usecases.NewTrackStorage(s3store)
	h := NewDownloadHandler(queries, trackStore)

	artist := "Download-test"
	title := fmt.Sprintf("Download song %d", time.Now().UnixNano())
	filename := "download-test.mp3"
	fixture := []byte("ID3-download-test-payload")

	stored, err := trackStore.PutOriginal(ctx, "download-seed", filename, fixture, "audio/mpeg")
	if err != nil {
		t.Fatalf("store original test object: %v", err)
	}
	defer func() {
		_ = s3store.DeleteObject(ctx, stored.Key)
	}()

	track, err := queries.CreateTrack(ctx, repositories.CreateTrackParams{
		Artist:            artist,
		Title:             title,
		OriginalFilename:  filename,
		OriginalObjectKey: stored.Key,
		OriginalSize:      int64(len(fixture)),
		Status:            repositories.TrackStatusPending,
	})
	if err != nil {
		t.Fatalf("create track: %v", err)
	}
	defer func() {
		_ = queries.DeleteTrackByID(ctx, track.ID)
	}()

	req := httptest.NewRequest(http.MethodGet, "/download/"+track.ID.String(), nil)
	req = mux.SetURLVars(req, map[string]string{"track_id": track.ID.String()})
	res := httptest.NewRecorder()

	h.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", res.Code, res.Body.String())
	}

	var out downloadResp
	if err := json.Unmarshal(res.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.TrackID != track.ID.String() {
		t.Fatalf("track_id mismatch: want=%s got=%s", track.ID.String(), out.TrackID)
	}
	if out.OriginalFilename != filename {
		t.Fatalf("original_filename mismatch: want=%s got=%s", filename, out.OriginalFilename)
	}
	if out.DownloadURL == "" {
		t.Fatalf("download_url is empty")
	}

	httpResp, err := http.Get(out.DownloadURL)
	if err != nil {
		t.Fatalf("download by returned url: %v", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		t.Fatalf("expected url status 200, got %d, body=%s", httpResp.StatusCode, string(body))
	}

	downloaded, err := io.ReadAll(httpResp.Body)
	if err != nil {
		t.Fatalf("read downloaded body: %v", err)
	}

	if !bytes.Equal(fixture, downloaded) {
		t.Fatalf("downloaded bytes differ from uploaded original")
	}
}

func TestDownload_InvalidTrackID_Returns400(t *testing.T) {
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

	h := NewDownloadHandler(repositories.New(pool), usecases.NewTrackStorage(mustS3(t, cfg)))
	req := httptest.NewRequest(http.MethodGet, "/download/not-a-uuid", nil)
	req = mux.SetURLVars(req, map[string]string{"track_id": "not-a-uuid"})
	res := httptest.NewRecorder()

	h.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d, body=%s", res.Code, res.Body.String())
	}
}

func TestDownload_MissingTrack_Returns404(t *testing.T) {
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

	h := NewDownloadHandler(repositories.New(pool), usecases.NewTrackStorage(mustS3(t, cfg)))
	missing := "11111111-1111-1111-1111-111111111111"
	req := httptest.NewRequest(http.MethodGet, "/download/"+missing, nil)
	req = mux.SetURLVars(req, map[string]string{"track_id": missing})
	res := httptest.NewRecorder()

	h.ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d, body=%s", res.Code, res.Body.String())
	}
}

func mustS3(t *testing.T, cfg *server.Config) *storage.S3Storage {
	t.Helper()
	s3store, err := storage.NewS3Storage(cfg)
	if err != nil {
		t.Fatalf("new s3 storage: %v", err)
	}
	return s3store
}
