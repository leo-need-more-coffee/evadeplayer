CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TYPE video_status AS ENUM ('pending', 'processing', 'ready', 'failed');

CREATE TABLE videos (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    status        video_status NOT NULL DEFAULT 'pending',
    progress      SMALLINT     NOT NULL DEFAULT 0,
    original_path TEXT         NOT NULL,
    duration      FLOAT,
    width         INT,
    height        INT,
    size_bytes    BIGINT       NOT NULL DEFAULT 0,
    error_message TEXT,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_videos_status     ON videos(status);
CREATE INDEX idx_videos_created_at ON videos(created_at DESC);

CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER videos_updated_at
    BEFORE UPDATE ON videos
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();
