CREATE TABLE series (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title       TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_series_user_id ON series(user_id);
CREATE INDEX idx_series_created_at ON series(created_at DESC);

CREATE TRIGGER series_updated_at
    BEFORE UPDATE ON series
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

ALTER TABLE videos
    ADD COLUMN series_id      UUID REFERENCES series(id) ON DELETE SET NULL,
    ADD COLUMN season_number  INT,
    ADD COLUMN episode_number INT;

CREATE INDEX idx_videos_series_id ON videos(series_id);
