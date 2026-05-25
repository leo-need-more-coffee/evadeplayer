ALTER TABLE videos
    DROP COLUMN IF EXISTS version_label,
    DROP COLUMN IF EXISTS version_of;
