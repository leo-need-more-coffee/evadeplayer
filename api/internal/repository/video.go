package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/evadeplayer/api/internal/model"
)

type VideoRepo struct {
	db *pgxpool.Pool
}

func NewVideoRepo(db *pgxpool.Pool) *VideoRepo {
	return &VideoRepo{db: db}
}

func (r *VideoRepo) CreateWithID(ctx context.Context, v *model.Video) error {
	q := `INSERT INTO videos (id, user_id, title, description, original_path, size_bytes,
	                          series_id, season_number, episode_number,
	                          version_of, version_label, version_description)
	      VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	      RETURNING status, created_at, updated_at`
	err := r.db.QueryRow(ctx, q,
		v.ID, v.UserID, v.Title, v.Description, v.OriginalPath, v.SizeBytes,
		v.SeriesID, v.SeasonNumber, v.EpisodeNumber,
		v.VersionOf, v.VersionLabel, v.VersionDescription,
	).Scan(&v.Status, &v.CreatedAt, &v.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create video with id: %w", err)
	}
	return nil
}

func (r *VideoRepo) FindByID(ctx context.Context, id string) (*model.Video, error) {
	v := &model.Video{}
	q := `SELECT id, user_id, title, description, status, progress, original_path,
	             duration, width, height, size_bytes, error_message,
	             series_id, season_number, episode_number,
	             version_of, version_label, version_description,
	             created_at, updated_at
	      FROM videos WHERE id = $1`
	err := r.db.QueryRow(ctx, q, id).Scan(
		&v.ID, &v.UserID, &v.Title, &v.Description, &v.Status, &v.Progress, &v.OriginalPath,
		&v.Duration, &v.Width, &v.Height, &v.SizeBytes, &v.ErrorMessage,
		&v.SeriesID, &v.SeasonNumber, &v.EpisodeNumber,
		&v.VersionOf, &v.VersionLabel, &v.VersionDescription,
		&v.CreatedAt, &v.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find video by id: %w", err)
	}
	return v, nil
}

func (r *VideoRepo) FindVersionsByID(ctx context.Context, videoID string) ([]*model.VideoVersion, error) {
	q := `SELECT id, COALESCE(version_label, ''), COALESCE(version_description, ''), status
	      FROM videos WHERE version_of = $1 ORDER BY created_at LIMIT 50`
	rows, err := r.db.Query(ctx, q, videoID)
	if err != nil {
		return nil, fmt.Errorf("find versions: %w", err)
	}
	defer rows.Close()

	var versions []*model.VideoVersion
	for rows.Next() {
		v := &model.VideoVersion{}
		if err := rows.Scan(&v.ID, &v.Label, &v.Description, &v.Status); err != nil {
			return nil, fmt.Errorf("scan version: %w", err)
		}
		versions = append(versions, v)
	}
	return versions, rows.Err()
}

func (r *VideoRepo) List(ctx context.Context, limit, offset int) ([]*model.VideoListItem, int, error) {
	q := `SELECT id, user_id, title, status, duration, width, height, size_bytes,
	             series_id, season_number, episode_number,
	             version_of, version_label, version_description, created_at,
	             COUNT(*) OVER() AS total
	      FROM videos ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	rows, err := r.db.Query(ctx, q, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list videos: %w", err)
	}
	defer rows.Close()

	var total int
	var items []*model.VideoListItem
	for rows.Next() {
		item := &model.VideoListItem{}
		if err := rows.Scan(
			&item.ID, &item.UserID, &item.Title, &item.Status,
			&item.Duration, &item.Width, &item.Height, &item.SizeBytes,
			&item.SeriesID, &item.SeasonNumber, &item.EpisodeNumber,
			&item.VersionOf, &item.VersionLabel, &item.VersionDescription, &item.CreatedAt,
			&total,
		); err != nil {
			return nil, 0, fmt.Errorf("scan video row: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("video rows error: %w", err)
	}
	return items, total, nil
}

func (r *VideoRepo) UpdateStatus(ctx context.Context, id string, status model.VideoStatus, errMsg *string) error {
	q := `UPDATE videos SET status = $1, error_message = $2 WHERE id = $3`
	_, err := r.db.Exec(ctx, q, status, errMsg, id)
	if err != nil {
		return fmt.Errorf("update video status: %w", err)
	}
	return nil
}

func (r *VideoRepo) UpdateMetadata(ctx context.Context, id string, duration float64, width, height int) error {
	q := `UPDATE videos SET duration = $1, width = $2, height = $3 WHERE id = $4`
	_, err := r.db.Exec(ctx, q, duration, width, height, id)
	if err != nil {
		return fmt.Errorf("update video metadata: %w", err)
	}
	return nil
}
