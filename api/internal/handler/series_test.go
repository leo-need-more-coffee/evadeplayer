package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/evadeplayer/api/internal/handler"
	"github.com/evadeplayer/api/internal/model"
	"github.com/evadeplayer/api/internal/repository"
	"github.com/evadeplayer/api/internal/service"
)

// memSeries implements service.SeriesStorer for handler tests.
type memSeries struct {
	mu      sync.Mutex
	seq     int
	byID    map[string]*model.Series
	list    []*model.Series
	details map[string]*model.SeriesDetail
}

func newMemSeries() *memSeries {
	return &memSeries{
		byID:    make(map[string]*model.Series),
		details: make(map[string]*model.SeriesDetail),
	}
}

func (m *memSeries) Create(_ context.Context, s *model.Series) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.seq++
	s.ID = fmt.Sprintf("series-%d", m.seq)
	s.CreatedAt = time.Now()
	s.UpdatedAt = time.Now()
	cp := *s
	m.byID[s.ID] = &cp
	m.list = append(m.list, &cp)
	return nil
}

func (m *memSeries) FindByID(_ context.Context, id string) (*model.Series, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.byID[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	cp := *s
	return &cp, nil
}

func (m *memSeries) FindByIDWithEpisodes(_ context.Context, id string) (*model.SeriesDetail, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if d, ok := m.details[id]; ok {
		return d, nil
	}
	s, ok := m.byID[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return &model.SeriesDetail{
		Series:   *s,
		Seasons:  []model.Season{},
		Episodes: []model.Episode{},
	}, nil
}

func (m *memSeries) List(_ context.Context, limit, offset int) ([]*model.Series, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	total := len(m.list)
	var out []*model.Series
	for i, s := range m.list {
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

func newSeriesHandler(store *memSeries) *handler.SeriesHandler {
	svc := service.NewSeriesService(store, "http://localhost/hls")
	return handler.NewSeriesHandler(svc)
}

func jsonBody(v any) *bytes.Buffer {
	b, _ := json.Marshal(v)
	return bytes.NewBuffer(b)
}

// --- CreateSeries ---

func TestSeriesHandler_Create_OK(t *testing.T) {
	h := newSeriesHandler(newMemSeries())
	body := jsonBody(map[string]string{"title": "My Show", "description": "Cool"})
	req := httptest.NewRequest(http.MethodPost, "/series", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.CreateSeries(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body)
	}
	var resp map[string]any
	_ = json.NewDecoder(rr.Body).Decode(&resp)
	if resp["id"] == nil || resp["id"] == "" {
		t.Error("response must include id")
	}
	if resp["title"] != "My Show" {
		t.Errorf("title: got %v, want %q", resp["title"], "My Show")
	}
}

func TestSeriesHandler_Create_MissingTitle(t *testing.T) {
	h := newSeriesHandler(newMemSeries())
	body := jsonBody(map[string]string{"description": "No title"})
	req := httptest.NewRequest(http.MethodPost, "/series", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.CreateSeries(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestSeriesHandler_Create_InvalidJSON(t *testing.T) {
	h := newSeriesHandler(newMemSeries())
	req := httptest.NewRequest(http.MethodPost, "/series", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.CreateSeries(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestSeriesHandler_Create_WhitespaceTitle(t *testing.T) {
	h := newSeriesHandler(newMemSeries())
	body := jsonBody(map[string]string{"title": "   "})
	req := httptest.NewRequest(http.MethodPost, "/series", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.CreateSeries(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for whitespace-only title, got %d", rr.Code)
	}
}

// --- GetSeries ---

func TestSeriesHandler_Get_OK(t *testing.T) {
	store := newMemSeries()
	h := newSeriesHandler(store)
	ctx := context.Background()

	// seed a series
	s := &model.Series{UserID: "u1", Title: "Test Show"}
	_ = store.Create(ctx, s)

	req := httptest.NewRequest(http.MethodGet, "/series/"+s.ID, nil)
	req.SetPathValue("id", s.ID)
	rr := httptest.NewRecorder()

	h.GetSeries(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body)
	}
	var resp map[string]any
	_ = json.NewDecoder(rr.Body).Decode(&resp)
	if resp["id"] != s.ID {
		t.Errorf("id: got %v, want %q", resp["id"], s.ID)
	}
	if resp["seasons"] == nil {
		t.Error("response must include seasons")
	}
	if resp["episodes"] == nil {
		t.Error("response must include episodes")
	}
}

func TestSeriesHandler_Get_NotFound(t *testing.T) {
	h := newSeriesHandler(newMemSeries())
	req := httptest.NewRequest(http.MethodGet, "/series/no-such-id", nil)
	req.SetPathValue("id", "no-such-id")
	rr := httptest.NewRecorder()

	h.GetSeries(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestSeriesHandler_Get_WithSeasons(t *testing.T) {
	store := newMemSeries()
	h := newSeriesHandler(store)
	ctx := context.Background()

	s := &model.Series{UserID: "u1", Title: "Show"}
	_ = store.Create(ctx, s)

	sn, ep := 1, 1
	store.details[s.ID] = &model.SeriesDetail{
		Series: *s,
		Seasons: []model.Season{
			{SeasonNumber: 1, Episodes: []model.Episode{
				{ID: "ep-1", Title: "Pilot", SeasonNumber: &sn, EpisodeNumber: &ep, Status: model.StatusReady},
			}},
		},
		Episodes: []model.Episode{},
	}

	req := httptest.NewRequest(http.MethodGet, "/series/"+s.ID, nil)
	req.SetPathValue("id", s.ID)
	rr := httptest.NewRecorder()
	h.GetSeries(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body)
	}

	var resp struct {
		Seasons []struct {
			SeasonNumber int              `json:"season_number"`
			Episodes     []map[string]any `json:"episodes"`
		} `json:"seasons"`
	}
	_ = json.NewDecoder(rr.Body).Decode(&resp)
	if len(resp.Seasons) != 1 {
		t.Fatalf("expected 1 season, got %d", len(resp.Seasons))
	}
	if resp.Seasons[0].SeasonNumber != 1 {
		t.Errorf("expected season_number=1, got %d", resp.Seasons[0].SeasonNumber)
	}
	if len(resp.Seasons[0].Episodes) != 1 {
		t.Errorf("expected 1 episode, got %d", len(resp.Seasons[0].Episodes))
	}
}

// --- ListSeries ---

func TestSeriesHandler_List_Empty(t *testing.T) {
	h := newSeriesHandler(newMemSeries())
	req := httptest.NewRequest(http.MethodGet, "/series", nil)
	rr := httptest.NewRecorder()

	h.ListSeries(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var resp struct {
		Items []any `json:"items"`
		Total int   `json:"total"`
	}
	_ = json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Total != 0 {
		t.Errorf("expected total=0, got %d", resp.Total)
	}
}

func TestSeriesHandler_List_ReturnsSeries(t *testing.T) {
	store := newMemSeries()
	ctx := context.Background()
	for i := range 3 {
		s := &model.Series{UserID: "u1", Title: fmt.Sprintf("Show %d", i+1)}
		_ = store.Create(ctx, s)
	}

	h := newSeriesHandler(store)
	req := httptest.NewRequest(http.MethodGet, "/series?page=1&page_size=10", nil)
	rr := httptest.NewRecorder()
	h.ListSeries(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var resp struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
	}
	_ = json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Total != 3 {
		t.Errorf("expected total=3, got %d", resp.Total)
	}
	if len(resp.Items) != 3 {
		t.Errorf("expected 3 items, got %d", len(resp.Items))
	}
}
