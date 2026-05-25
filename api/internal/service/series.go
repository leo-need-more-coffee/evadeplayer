package service

import (
	"context"
	"fmt"

	"github.com/evadeplayer/api/internal/model"
)

type SeriesService struct {
	repo          SeriesStorer
	publicBaseURL string
}

func NewSeriesService(repo SeriesStorer, publicHost string) *SeriesService {
	return &SeriesService{repo: repo, publicBaseURL: publicHost}
}

type CreateSeriesInput struct {
	UserID      string
	Title       string
	Description string
}

func (s *SeriesService) CreateSeries(ctx context.Context, in *CreateSeriesInput) (*model.Series, error) {
	series := &model.Series{
		UserID:      in.UserID,
		Title:       in.Title,
		Description: in.Description,
	}
	if err := s.repo.Create(ctx, series); err != nil {
		return nil, fmt.Errorf("create series: %w", err)
	}
	return series, nil
}

func (s *SeriesService) GetSeries(ctx context.Context, id string) (*model.SeriesDetail, error) {
	detail, err := s.repo.FindByIDWithEpisodes(ctx, id)
	if err != nil {
		return nil, err
	}
	for i := range detail.Seasons {
		for j := range detail.Seasons[i].Episodes {
			ep := &detail.Seasons[i].Episodes[j]
			if ep.Status == model.StatusReady {
				ep.PreviewURL = s.previewURL(ep.ID)
			}
		}
	}
	for i := range detail.Episodes {
		if detail.Episodes[i].Status == model.StatusReady {
			detail.Episodes[i].PreviewURL = s.previewURL(detail.Episodes[i].ID)
		}
	}
	return detail, nil
}

func (s *SeriesService) previewURL(videoID string) string {
	return fmt.Sprintf("%s/thumbnails/%s/preview.jpg", s.publicBaseURL, videoID)
}

func (s *SeriesService) ListSeries(ctx context.Context, page, pageSize int) ([]*model.Series, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.List(ctx, pageSize, (page-1)*pageSize)
}
