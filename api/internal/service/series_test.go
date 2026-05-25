package service_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/evadeplayer/api/internal/model"
	"github.com/evadeplayer/api/internal/repository"
	"github.com/evadeplayer/api/internal/service"
)

// fakeSeriesStore is shared across series_test.go and upload_test.go.
type fakeSeriesStore struct {
	mu      sync.Mutex
	seq     int
	byID    map[string]*model.Series
	list    []*model.Series
	details map[string]*model.SeriesDetail // pre-set for FindByIDWithEpisodes
}

func newFakeSeriesStore() *fakeSeriesStore {
	return &fakeSeriesStore{
		byID:    make(map[string]*model.Series),
		details: make(map[string]*model.SeriesDetail),
	}
}

func (f *fakeSeriesStore) Create(_ context.Context, s *model.Series) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seq++
	s.ID = fmt.Sprintf("series-%d", f.seq)
	s.CreatedAt = time.Now()
	s.UpdatedAt = time.Now()
	cp := *s
	f.byID[s.ID] = &cp
	f.list = append(f.list, &cp)
	return nil
}

func (f *fakeSeriesStore) FindByID(_ context.Context, id string) (*model.Series, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.byID[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	cp := *s
	return &cp, nil
}

func (f *fakeSeriesStore) FindByIDWithEpisodes(_ context.Context, id string) (*model.SeriesDetail, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if d, ok := f.details[id]; ok {
		return d, nil
	}
	s, ok := f.byID[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return &model.SeriesDetail{
		Series:   *s,
		Seasons:  []model.Season{},
		Episodes: []model.Episode{},
	}, nil
}

func (f *fakeSeriesStore) List(_ context.Context, limit, offset int) ([]*model.Series, int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	total := len(f.list)
	var out []*model.Series
	for i, s := range f.list {
		if i < offset {
			continue
		}
		if len(out) >= limit {
			break
		}
		cp := *s
		out = append(out, &cp)
	}
	return out, total, nil
}

func newSeriesSvc(store *fakeSeriesStore) *service.SeriesService {
	return service.NewSeriesService(store, "http://localhost/hls")
}

// --- CreateSeries ---

func TestSeriesService_CreateSeries(t *testing.T) {
	store := newFakeSeriesStore()
	svc := newSeriesSvc(store)

	s, err := svc.CreateSeries(context.Background(), &service.CreateSeriesInput{
		UserID:      "user-1",
		Title:       "My Show",
		Description: "A cool show",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.ID == "" {
		t.Error("ID must be set after creation")
	}
	if s.Title != "My Show" {
		t.Errorf("title: got %q, want %q", s.Title, "My Show")
	}
	if s.UserID != "user-1" {
		t.Errorf("user_id: got %q, want %q", s.UserID, "user-1")
	}
}

// --- GetSeries ---

func TestSeriesService_GetSeries_NotFound(t *testing.T) {
	svc := newSeriesSvc(newFakeSeriesStore())
	_, err := svc.GetSeries(context.Background(), "no-such-id")
	if !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestSeriesService_GetSeries_Empty(t *testing.T) {
	store := newFakeSeriesStore()
	ctx := context.Background()
	s, _ := newSeriesSvc(store).CreateSeries(ctx, &service.CreateSeriesInput{UserID: "u1", Title: "Show"})

	detail, err := newSeriesSvc(store).GetSeries(ctx, s.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(detail.Seasons) != 0 {
		t.Errorf("expected 0 seasons, got %d", len(detail.Seasons))
	}
	if len(detail.Episodes) != 0 {
		t.Errorf("expected 0 episodes, got %d", len(detail.Episodes))
	}
}

func TestSeriesService_GetSeries_WithSeasons(t *testing.T) {
	store := newFakeSeriesStore()
	ctx := context.Background()
	s, _ := newSeriesSvc(store).CreateSeries(ctx, &service.CreateSeriesInput{UserID: "u1", Title: "Show"})

	sn1, ep1 := 1, 1
	sn2, ep2 := 2, 1
	store.details[s.ID] = &model.SeriesDetail{
		Series: *mustFindSeries(store, ctx, s.ID),
		Seasons: []model.Season{
			{SeasonNumber: 1, Episodes: []model.Episode{
				{ID: "ep-s1e1", Title: "Pilot", SeasonNumber: &sn1, EpisodeNumber: &ep1, Status: model.StatusReady},
			}},
			{SeasonNumber: 2, Episodes: []model.Episode{
				{ID: "ep-s2e1", Title: "Premiere", SeasonNumber: &sn2, EpisodeNumber: &ep2, Status: model.StatusPending},
			}},
		},
		Episodes: []model.Episode{},
	}

	detail, err := newSeriesSvc(store).GetSeries(ctx, s.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(detail.Seasons) != 2 {
		t.Fatalf("expected 2 seasons, got %d", len(detail.Seasons))
	}
	if detail.Seasons[0].SeasonNumber != 1 {
		t.Errorf("expected season 1 first, got %d", detail.Seasons[0].SeasonNumber)
	}
	if len(detail.Seasons[0].Episodes) != 1 {
		t.Errorf("expected 1 episode in season 1, got %d", len(detail.Seasons[0].Episodes))
	}
}

func TestSeriesService_GetSeries_ThumbnailAddedForReadyEpisodes(t *testing.T) {
	store := newFakeSeriesStore()
	ctx := context.Background()
	s, _ := newSeriesSvc(store).CreateSeries(ctx, &service.CreateSeriesInput{UserID: "u1", Title: "Show"})

	sn := 1
	store.details[s.ID] = &model.SeriesDetail{
		Series: *mustFindSeries(store, ctx, s.ID),
		Seasons: []model.Season{
			{SeasonNumber: 1, Episodes: []model.Episode{
				{ID: "ep-ready", Title: "Ep1", SeasonNumber: &sn, Status: model.StatusReady},
				{ID: "ep-pending", Title: "Ep2", SeasonNumber: &sn, Status: model.StatusPending},
			}},
		},
		Episodes: []model.Episode{},
	}

	detail, err := newSeriesSvc(store).GetSeries(ctx, s.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	eps := detail.Seasons[0].Episodes
	if !strings.Contains(eps[0].PreviewURL, "ep-ready") {
		t.Errorf("ready episode must have preview URL containing its ID, got %q", eps[0].PreviewURL)
	}
	if !strings.Contains(eps[0].PreviewURL, "/preview.jpg") {
		t.Errorf("ready episode preview must point to preview.jpg, got %q", eps[0].PreviewURL)
	}
	if eps[1].PreviewURL != "" {
		t.Errorf("pending episode must NOT have preview URL, got %q", eps[1].PreviewURL)
	}
}

func TestSeriesService_GetSeries_WithoutSeasons(t *testing.T) {
	store := newFakeSeriesStore()
	ctx := context.Background()
	s, _ := newSeriesSvc(store).CreateSeries(ctx, &service.CreateSeriesInput{UserID: "u1", Title: "Show"})

	store.details[s.ID] = &model.SeriesDetail{
		Series:  *mustFindSeries(store, ctx, s.ID),
		Seasons: []model.Season{},
		Episodes: []model.Episode{
			{ID: "ep-1", Title: "Part 1", Status: model.StatusReady},
			{ID: "ep-2", Title: "Part 2", Status: model.StatusReady},
		},
	}

	detail, err := newSeriesSvc(store).GetSeries(ctx, s.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(detail.Seasons) != 0 {
		t.Errorf("expected 0 seasons, got %d", len(detail.Seasons))
	}
	if len(detail.Episodes) != 2 {
		t.Fatalf("expected 2 top-level episodes, got %d", len(detail.Episodes))
	}
	if !strings.Contains(detail.Episodes[0].PreviewURL, "ep-1") {
		t.Errorf("ready top-level episode must have preview URL, got %q", detail.Episodes[0].PreviewURL)
	}
	if !strings.Contains(detail.Episodes[0].PreviewURL, "/preview.jpg") {
		t.Errorf("ready top-level episode preview must point to preview.jpg, got %q", detail.Episodes[0].PreviewURL)
	}
}

// --- ListSeries ---

func TestSeriesService_ListSeries_Empty(t *testing.T) {
	svc := newSeriesSvc(newFakeSeriesStore())
	items, total, err := svc.ListSeries(context.Background(), 1, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 0 || len(items) != 0 {
		t.Errorf("expected 0 items/total, got %d/%d", len(items), total)
	}
}

func TestSeriesService_ListSeries_Pagination(t *testing.T) {
	store := newFakeSeriesStore()
	svc := newSeriesSvc(store)
	ctx := context.Background()
	for i := range 5 {
		_, _ = svc.CreateSeries(ctx, &service.CreateSeriesInput{
			UserID: "u1",
			Title:  fmt.Sprintf("Show %d", i+1),
		})
	}

	items, total, err := svc.ListSeries(ctx, 1, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 5 {
		t.Errorf("expected total=5, got %d", total)
	}
	if len(items) != 3 {
		t.Errorf("expected 3 items on page 1, got %d", len(items))
	}

	items2, _, _ := svc.ListSeries(ctx, 2, 3)
	if len(items2) != 2 {
		t.Errorf("expected 2 items on page 2, got %d", len(items2))
	}
}

// mustFindSeries is a test helper that fetches a series from the store or panics.
func mustFindSeries(store *fakeSeriesStore, ctx context.Context, id string) *model.Series {
	s, err := store.FindByID(ctx, id)
	if err != nil {
		panic(fmt.Sprintf("mustFindSeries: %v", err))
	}
	return s
}
