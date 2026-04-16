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
	"strings"
	"testing"
	"time"

	"github.com/DNP-Project-China/Audio-Streaming-Service/core-api/events"
	"github.com/DNP-Project-China/Audio-Streaming-Service/core-api/repositories"
	"github.com/DNP-Project-China/Audio-Streaming-Service/core-api/server"
	"github.com/DNP-Project-China/Audio-Streaming-Service/core-api/storage"
	"github.com/DNP-Project-China/Audio-Streaming-Service/core-api/usecases"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type uploadResp struct {
	TrackID          string `json:"track_id"`
	Artist           string `json:"artist"`
	Title            string `json:"title"`
	OriginalFilename string `json:"original_filename"`
	Status           string `json:"status"`
	UploadedAt       string `json:"uploaded_at"`
}

type spyPublisher struct {
	called   bool
	trackID  string
	path     string
	priority int
	err      error
}

func (s *spyPublisher) PublishCreated(ctx context.Context, trackID string, path string, priority int) error {
	s.called = true
	s.trackID = trackID
	s.path = path
	s.priority = priority
	return s.err
}

var _ events.TranscodePublisher = (*spyPublisher)(nil)

func TestUpload_StoresFileAndRoundTripsBytes(t *testing.T) {
	artist := "Eminem-test-ok"
	title := fmt.Sprintf("Mock Song OK %d", time.Now().UnixNano())

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
	pub := &spyPublisher{}
	h := NewUploadHandler(queries, trackStore, pub)

	fixturePath := filepath.Join("testdata", "testfile.mp3")
	fixtureBytes, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	_ = mw.WriteField("artist", artist)
	_ = mw.WriteField("title", title)

	fw, err := mw.CreateFormFile("file", "testfile.mp3")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := fw.Write(fixtureBytes); err != nil {
		t.Fatalf("write multipart payload: %v", err)
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	res := httptest.NewRecorder()

	h.ServeHTTP(res, req)

	if res.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d, body=%s", res.Code, res.Body.String())
	}

	var out uploadResp
	if err := json.Unmarshal(res.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.TrackID == "" {
		t.Fatalf("response track_id is empty")
	}
	if out.Status != "pending" {
		t.Fatalf("expected pending status, got %q", out.Status)
	}
	if out.Artist != artist || out.Title != title {
		t.Fatalf("unexpected response metadata: artist=%q title=%q", out.Artist, out.Title)
	}
	if !pub.called {
		t.Fatalf("expected transcode job publish call")
	}
	if pub.trackID != out.TrackID {
		t.Fatalf("publisher trackID mismatch: want=%s got=%s", out.TrackID, pub.trackID)
	}
	if pub.priority != 1 {
		t.Fatalf("expected priority=1, got %d", pub.priority)
	}

	var trackID pgtype.UUID
	if err := trackID.Scan(out.TrackID); err != nil {
		t.Fatalf("parse track uuid: %v", err)
	}

	track, err := queries.GetTrackByID(ctx, trackID)
	if err != nil {
		t.Fatalf("load track from db: %v", err)
	}

	storedBytes, err := s3store.GetObject(ctx, track.OriginalObjectKey)
	if err != nil {
		t.Fatalf("get object from s3: %v", err)
	}

	if !bytes.Equal(fixtureBytes, storedBytes) {
		t.Fatalf("uploaded file bytes differ from stored object bytes")
	}

	if err := s3store.DeleteObject(ctx, track.OriginalObjectKey); err != nil {
		t.Fatalf("cleanup s3 object: %v", err)
	}
	if err := queries.DeleteTrackByID(ctx, trackID); err != nil {
		t.Fatalf("cleanup db row: %v", err)
	}
}

func TestUpload_Returns500WhenPublishFails(t *testing.T) {
	artist := "Eminem-test-fail"
	title := fmt.Sprintf("Mock Song FAIL %d", time.Now().UnixNano())

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
	pub := &spyPublisher{err: fmt.Errorf("kafka unavailable")}
	h := NewUploadHandler(queries, trackStore, pub)

	fixturePath := filepath.Join("testdata", "testfile.mp3")
	fixtureBytes, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	_ = mw.WriteField("artist", artist)
	_ = mw.WriteField("title", title)

	fw, err := mw.CreateFormFile("file", "testfile.mp3")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := fw.Write(fixtureBytes); err != nil {
		t.Fatalf("write multipart payload: %v", err)
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)

	if res.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d, body=%s", res.Code, res.Body.String())
	}

	tracks, err := queries.ListTracks(ctx)
	if err != nil {
		t.Fatalf("list tracks: %v", err)
	}
	for _, tr := range tracks {
		if tr.Artist == artist && tr.Title == title {
			t.Fatalf("expected rollback on publish failure, found track id=%s", tr.ID.String())
		}
	}
}

func newTestPool(ctx context.Context, cfg *server.Config) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL())
	if err == nil {
		if pingErr := pool.Ping(ctx); pingErr == nil {
			return pool, nil
		}
		pool.Close()
	}

	if strings.EqualFold(strings.TrimSpace(cfg.PostgresHost), "postgres") {
		fallback := *cfg
		fallback.PostgresHost = "localhost"
		pool, fallbackErr := pgxpool.New(ctx, fallback.DatabaseURL())
		if fallbackErr != nil {
			return nil, fmt.Errorf("primary db connect failed (%v), fallback connect failed (%v)", err, fallbackErr)
		}

		if pingErr := pool.Ping(ctx); pingErr != nil {
			pool.Close()
			return nil, fmt.Errorf("primary db connect failed (%v), fallback ping failed (%v)", err, pingErr)
		}

		return pool, nil
	}

	if err != nil {
		return nil, err
	}

	return nil, fmt.Errorf("db ping failed")
}
