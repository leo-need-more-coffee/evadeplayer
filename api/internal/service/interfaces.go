package service

import (
	"context"
	"io"

	"github.com/evadeplayer/api/internal/model"
)

type UserStorer interface {
	Create(ctx context.Context, u *model.User) error
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	FindByID(ctx context.Context, id string) (*model.User, error)
}

type VideoStorer interface {
	CreateWithID(ctx context.Context, v *model.Video) error
	FindByID(ctx context.Context, id string) (*model.Video, error)
	FindVersionsByID(ctx context.Context, videoID string) ([]*model.VideoVersion, error)
	List(ctx context.Context, limit, offset int) ([]*model.VideoListItem, int, error)
	UpdateStatus(ctx context.Context, id string, status model.VideoStatus, errMsg *string) error
}

type SeriesStorer interface {
	Create(ctx context.Context, s *model.Series) error
	FindByID(ctx context.Context, id string) (*model.Series, error)
	FindByIDWithEpisodes(ctx context.Context, id string) (*model.SeriesDetail, error)
	List(ctx context.Context, limit, offset int) ([]*model.Series, int, error)
}

type TaskEnqueuer interface {
	Enqueue(ctx context.Context, task *model.TranscodeTask) error
}

type BlobStorage interface {
	Upload(ctx context.Context, filePath string, r io.Reader, contentType string) error
	Delete(ctx context.Context, filePath string) error
}
