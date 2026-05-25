package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/evadeplayer/api/internal/repository"
	"github.com/evadeplayer/api/internal/service"
)

type SeriesHandler struct {
	svc *service.SeriesService
}

func NewSeriesHandler(svc *service.SeriesService) *SeriesHandler {
	return &SeriesHandler{svc: svc}
}

func (h *SeriesHandler) CreateSeries(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(body.Title) == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	series, err := h.svc.CreateSeries(r.Context(), &service.CreateSeriesInput{
		UserID:      userIDFromCtx(r),
		Title:       strings.TrimSpace(body.Title),
		Description: body.Description,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusCreated, series)
}

func (h *SeriesHandler) GetSeries(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing series id")
		return
	}

	detail, err := h.svc.GetSeries(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "series not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (h *SeriesHandler) ListSeries(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))

	items, total, err := h.svc.ListSeries(r.Context(), page, pageSize)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
		"total": total,
	})
}
