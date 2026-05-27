package handler

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/evadeplayer/api/internal/service"
)

const maxPreviewSize = 20 << 20 // 20 MB

var allowedExtensions = map[string]bool{
	".mp4":  true,
	".mkv":  true,
	".mov":  true,
	".avi":  true,
	".webm": true,
	".m4v":  true,
}

var allowedPreviewExtensions = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".webp": "image/webp",
}

type UploadHandler struct {
	svc           *service.UploadService
	maxUploadSize int64
}

func NewUploadHandler(svc *service.UploadService, maxUploadSize int64) *UploadHandler {
	return &UploadHandler{svc: svc, maxUploadSize: maxUploadSize}
}

func (h *UploadHandler) Upload(w http.ResponseWriter, r *http.Request) {
	// Disable the server-level write deadline for large file uploads.
	// WriteTimeout is set globally for short responses, but upload can take minutes.
	rc := http.NewResponseController(w)
	_ = rc.SetWriteDeadline(time.Time{})

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "failed to parse multipart form")
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	if title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	description := r.FormValue("description")

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	if header.Size > h.maxUploadSize {
		writeError(w, http.StatusRequestEntityTooLarge, fmt.Sprintf("file too large, max %d GB", h.maxUploadSize>>30))
		return
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !allowedExtensions[ext] {
		writeError(w, http.StatusBadRequest, "unsupported file format")
		return
	}

	userID := userIDFromCtx(r)

	input := &service.UploadInput{
		UserID:      userID,
		Title:       title,
		Description: description,
		FileExt:     ext,
		Size:        header.Size,
		Reader:      file,
	}
	if sid := strings.TrimSpace(r.FormValue("series_id")); sid != "" {
		input.SeriesID = &sid
	}
	if s := r.FormValue("season_number"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			input.SeasonNumber = &n
		}
	}
	if s := r.FormValue("episode_number"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			input.EpisodeNumber = &n
		}
	}

	preview, previewHeader, err := r.FormFile("preview")
	if err == nil {
		defer preview.Close()
		if isNullPreviewPart(preview, previewHeader) {
			preview = nil
		} else {
			if previewHeader.Size > maxPreviewSize {
				writeError(w, http.StatusRequestEntityTooLarge, fmt.Sprintf("preview too large, max %d MB", maxPreviewSize>>20))
				return
			}
			previewExt := strings.ToLower(filepath.Ext(previewHeader.Filename))
			contentType, ok := allowedPreviewExtensions[previewExt]
			if !ok {
				writeError(w, http.StatusBadRequest, "unsupported preview image format")
				return
			}
			input.PreviewReader = preview
			input.PreviewContentType = contentType
		}
	} else if !errors.Is(err, http.ErrMissingFile) {
		writeError(w, http.StatusBadRequest, "failed to read preview image")
		return
	}

	if vof := strings.TrimSpace(r.FormValue("version_of")); vof != "" {
		input.VersionOf = &vof
	}
	if vl := strings.TrimSpace(r.FormValue("version_label")); vl != "" {
		input.VersionLabel = &vl
	}
	if vd := strings.TrimSpace(r.FormValue("version_description")); vd != "" {
		input.VersionDescription = &vd
	}

	video, err := h.svc.Upload(r.Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrSeriesNotFound):
			writeError(w, http.StatusBadRequest, "series not found")
		case errors.Is(err, service.ErrVideoNotFound):
			writeError(w, http.StatusBadRequest, "video not found")
		case errors.Is(err, service.ErrVersionChaining):
			writeError(w, http.StatusBadRequest, "cannot create a version of a version")
		default:
			writeError(w, http.StatusInternalServerError, "upload failed")
		}
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"id":     video.ID,
		"status": video.Status,
	})
}

func isNullPreviewPart(file multipart.File, header *multipart.FileHeader) bool {
	if header == nil {
		return true
	}
	if strings.TrimSpace(header.Filename) == "" && header.Size == 0 {
		return true
	}
	if header.Size > 4 {
		return false
	}
	data, err := io.ReadAll(io.LimitReader(file, 5))
	_, _ = file.Seek(0, io.SeekStart)
	if err != nil {
		return false
	}
	value := strings.ToLower(strings.TrimSpace(string(data)))
	return value == "" || value == "null"
}
