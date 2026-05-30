DROP TRIGGER IF EXISTS videos_updated_at ON videos;
DROP FUNCTION IF EXISTS update_updated_at;
DROP TABLE IF EXISTS videos;
DROP TYPE IF EXISTS video_status;
DROP EXTENSION IF EXISTS "pgcrypto";
