ALTER TABLE videos
    DROP COLUMN IF EXISTS episode_number,
    DROP COLUMN IF EXISTS season_number,
    DROP COLUMN IF EXISTS series_id;

DROP TABLE IF EXISTS series;
