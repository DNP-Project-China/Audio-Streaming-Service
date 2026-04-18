CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TYPE track_status AS ENUM ('pending', 'processing', 'ready', 'failed');

CREATE TABLE tracks (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  artist VARCHAR(255) NOT NULL,
  title VARCHAR(255) NOT NULL,
  original_filename VARCHAR(255) NOT NULL,
  original_object_key VARCHAR(512) NOT NULL,
  original_size BIGINT NOT NULL CHECK (original_size >= 0),
  status track_status NOT NULL DEFAULT 'pending',
  uploaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  hls_playlist_key VARCHAR(512),
  total_plays BIGINT NOT NULL DEFAULT 0 CHECK (total_plays >= 0)
);

CREATE INDEX idx_tracks_status_uploaded_at ON tracks (status, uploaded_at DESC);
CREATE INDEX idx_tracks_uploaded_at ON tracks (uploaded_at DESC);