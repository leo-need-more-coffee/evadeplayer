ALTER TABLE videos
    ADD COLUMN version_of    UUID REFERENCES videos(id) ON DELETE SET NULL,
    ADD COLUMN version_label TEXT;

CREATE INDEX idx_videos_version_of ON videos(version_of);
