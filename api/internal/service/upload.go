package service

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/google/uuid"

	"github.com/evadeplayer/api/internal/model"
	"github.com/evadeplayer/api/internal/repository"
)

var ErrSeriesNotFound = errors.New("series not found")
var ErrVideoNotFound = errors.New("video not found")
var ErrVersionChaining = errors.New("cannot create a version of a version")

type UploadService struct {
	videoRepo  VideoStorer
	seriesRepo SeriesStorer
	storage    BlobStorage
	producer   TaskEnqueuer
}

func NewUploadService(
	videoRepo VideoStorer,
	seriesRepo SeriesStorer,
	storage BlobStorage,
	producer TaskEnqueuer,
) *UploadService {
	return &UploadService{
		videoRepo:  videoRepo,
		seriesRepo: seriesRepo,
		storage:    storage,
		producer:   producer,
	}
}

type UploadInput struct {
	UserID             string
	Title              string
	Description        string
	FileExt            string
	Size               int64
	Reader             io.Reader
	PreviewReader      io.Reader
	PreviewContentType string
	SeriesID           *string
	SeasonNumber       *int
	EpisodeNumber      *int
	VersionOf          *string
	VersionLabel       *string
	VersionDescription *string
}

func (s *UploadService) Upload(ctx context.Context, in *UploadInput) (*model.Video, error) {
	if in.VersionOf != nil {
		parent, err := s.videoRepo.FindByID(ctx, *in.VersionOf)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return nil, ErrVideoNotFound
			}
			return nil, fmt.Errorf("find parent video: %w", err)
		}
		if parent.VersionOf != nil {
			return nil, ErrVersionChaining
		}
		// Versions are accessed through their parent — clear series fields.
		in.SeriesID = nil
		in.SeasonNumber = nil
		in.EpisodeNumber = nil
	} else if in.SeriesID != nil {
		series, err := s.seriesRepo.FindByID(ctx, *in.SeriesID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return nil, ErrSeriesNotFound
			}
			return nil, fmt.Errorf("find series: %w", err)
		}
		if series.UserID != in.UserID {
			return nil, ErrSeriesNotFound
		}
	}

	videoID := uuid.New().String()
	originalPath := fmt.Sprintf("originals/%s/original%s", videoID, in.FileExt)
	previewPath := fmt.Sprintf("thumbnails/%s/preview.jpg", videoID)

	if err := s.storage.Upload(ctx, originalPath, in.Reader, "application/octet-stream"); err != nil {
		return nil, fmt.Errorf("upload original to storage: %w", err)
	}
	hasPreviewOverride := in.PreviewReader != nil
	if hasPreviewOverride {
		if err := s.storage.Upload(ctx, previewPath, in.PreviewReader, in.PreviewContentType); err != nil {
			_ = s.storage.Delete(context.Background(), originalPath)
			return nil, fmt.Errorf("upload preview to storage: %w", err)
		}
	}

	v := &model.Video{
		ID:                 videoID,
		UserID:             in.UserID,
		Title:              in.Title,
		Description:        in.Description,
		OriginalPath:       originalPath,
		SizeBytes:          in.Size,
		SeriesID:           in.SeriesID,
		SeasonNumber:       in.SeasonNumber,
		EpisodeNumber:      in.EpisodeNumber,
		VersionOf:          in.VersionOf,
		VersionLabel:       in.VersionLabel,
		VersionDescription: in.VersionDescription,
	}
	if err := s.videoRepo.CreateWithID(ctx, v); err != nil {
		_ = s.storage.Delete(context.Background(), originalPath)
		if hasPreviewOverride {
			_ = s.storage.Delete(context.Background(), previewPath)
		}
		return nil, fmt.Errorf("create video record: %w", err)
	}

	task := &model.TranscodeTask{
		VideoID:         v.ID,
		OriginalPath:    originalPath,
		PreviewOverride: hasPreviewOverride,
	}
	if err := s.producer.Enqueue(ctx, task); err != nil {
		errMsg := err.Error()
		_ = s.videoRepo.UpdateStatus(ctx, v.ID, model.StatusFailed, &errMsg)
		return nil, fmt.Errorf("enqueue transcode task: %w", err)
	}

	return v, nil
}
