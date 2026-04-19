-- name: CreateTrack :one
INSERT INTO tracks (
  artist,
  title,
  original_filename,
  original_object_key,
  original_size,
  status
) VALUES (
  $1,
  $2,
  $3,
  $4,
  $5,
  $6
)
RETURNING *;

-- name: GetTrackByID :one
SELECT *
FROM tracks
WHERE id = $1
LIMIT 1;

-- name: ListReadyTracks :many
SELECT *
FROM tracks
WHERE status = 'ready'
ORDER BY uploaded_at DESC;

-- name: ListTracksByStatus :many
SELECT *
FROM tracks
WHERE status = $1::track_status
ORDER BY uploaded_at DESC;

-- name: ListTracks :many
SELECT *
FROM tracks
ORDER BY uploaded_at DESC;

-- name: MarkTrackProcessing :one
UPDATE tracks
SET status = 'processing',
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: MarkTrackReady :one
UPDATE tracks
SET status = 'ready',
    hls_playlist_key = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: MarkTrackFailed :one
UPDATE tracks
SET status = 'failed',
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteTrackByID :exec
DELETE FROM tracks
WHERE id = $1;