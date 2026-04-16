package usecases

import (
	"bytes"
	"context"
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/DNP-Project-China/Audio-Streaming-Service/core-api/storage"
	"go.uber.org/fx"
)

var Module = fx.Options(
	fx.Provide(NewTrackStorage),
)

var allowedSegment = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

type StoredObject struct {
	Key          string
	Size         int64
	LastModified *time.Time
	PublicURL    string
}

type TrackStorage struct {
	store *storage.S3Storage
}

func NewTrackStorage(store *storage.S3Storage) *TrackStorage {
	return &TrackStorage{store: store}
}

// SaveOriginalTrackFile stores source audio under raw/{track_id}/original{ext}.
func (s *TrackStorage) PutOriginal(ctx context.Context, trackID string, originalFilename string, body []byte, contentType string) (StoredObject, error) {
	key, err := s.OriginalKey(trackID, originalFilename)
	if err != nil {
		return StoredObject{}, err
	}

	if contentType == "" {
		contentType = "application/octet-stream"
	}

	if err := s.store.PutObject(ctx, key, bytes.NewReader(body), int64(len(body)), contentType); err != nil {
		return StoredObject{}, err
	}

	url, err := s.store.PublicObjectURL(key)
	if err != nil {
		return StoredObject{}, fmt.Errorf("build public url %q: %w", key, err)
	}

	return StoredObject{
		Key:       key,
		Size:      int64(len(body)),
		PublicURL: url,
	}, nil
}

func (s *TrackStorage) Get(ctx context.Context, key string) ([]byte, error) {
	if err := validateKey(key); err != nil {
		return nil, err
	}

	return s.store.GetObject(ctx, key)
}

func (s *TrackStorage) Delete(ctx context.Context, key string) error {
	if err := validateKey(key); err != nil {
		return err
	}

	return s.store.DeleteObject(ctx, key)
}

func (s *TrackStorage) ListRaw(ctx context.Context, trackID string, maxKeys int32) ([]StoredObject, error) {
	trackID, err := normalizeSegment(trackID, "track id")
	if err != nil {
		return nil, err
	}

	prefix := path.Join("raw", trackID) + "/"
	items, err := s.store.ListObjects(ctx, prefix, maxKeys)
	if err != nil {
		return nil, err
	}

	result := make([]StoredObject, 0, len(items))
	for _, item := range items {
		url, urlErr := s.store.PublicObjectURL(item.Key)
		if urlErr != nil {
			return nil, fmt.Errorf("build public url %q: %w", item.Key, urlErr)
		}

		result = append(result, StoredObject{
			Key:          item.Key,
			Size:         item.Size,
			LastModified: item.LastModified,
			PublicURL:    url,
		})
	}

	return result, nil
}

// PresignOriginalTrackDownload creates temporary URL for GET /download/{track_id}.
func (s *TrackStorage) PresignOriginalDownload(ctx context.Context, trackID string, originalFilename string, expires time.Duration) (string, string, error) {
	key, err := s.OriginalKey(trackID, originalFilename)
	if err != nil {
		return "", "", err
	}

	url, err := s.store.PresignGetURL(ctx, key, expires)
	if err != nil {
		return "", "", err
	}

	return key, url, nil
}

func (s *TrackStorage) PresignUpload(ctx context.Context, key string, contentType string, expires time.Duration) (string, error) {
	if err := validateKey(key); err != nil {
		return "", err
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	return s.store.PresignPutURL(ctx, key, contentType, expires)
}

func (s *TrackStorage) OriginalKey(trackID string, originalFilename string) (string, error) {
	trackID, err := normalizeSegment(trackID, "track id")
	if err != nil {
		return "", err
	}

	originalFilename = strings.TrimSpace(originalFilename)
	if originalFilename == "" {
		return "", fmt.Errorf("invalid original filename")
	}

	ext := strings.ToLower(filepath.Ext(originalFilename))
	if ext == "" {
		ext = ".bin"
	}

	if err := validateAudioExtension(ext); err != nil {
		return "", err
	}

	return path.Join("raw", trackID, "original"+ext), nil
}

func validateAudioExtension(ext string) error {
	switch ext {
	case ".mp3", ".flac", ".wav", ".aac", ".m4a", ".ogg":
		return nil
	default:
		return fmt.Errorf("unsupported file extension: %s", ext)
	}
}

func normalizeSegment(value string, field string) (string, error) {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(path.Clean("/"+value), "/")

	if value == "" || value == "." || strings.Contains(value, "/") {
		return "", fmt.Errorf("invalid %s", field)
	}

	if !allowedSegment.MatchString(value) {
		return "", fmt.Errorf("invalid %s: allowed chars are a-z, A-Z, 0-9, dot, underscore, hyphen", field)
	}

	return value, nil
}

func validateKey(key string) error {
	key = strings.TrimSpace(strings.TrimPrefix(path.Clean("/"+key), "/"))
	if key == "" || key == "." {
		return fmt.Errorf("invalid key")
	}

	return nil
}
