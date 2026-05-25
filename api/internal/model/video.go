package model

import "time"

type VideoStatus string

const (
	StatusPending    VideoStatus = "pending"
	StatusProcessing VideoStatus = "processing"
	StatusReady      VideoStatus = "ready"
	StatusFailed     VideoStatus = "failed"
)

// VideoVersion is a lightweight representation of an alternative video version,
// embedded in Video and Episode responses.
type VideoVersion struct {
	ID          string      `json:"id"`
	Label       string      `json:"label"`
	Description string      `json:"description,omitempty"`
	Status      VideoStatus `json:"status"`
	PreviewURL  string      `json:"preview_url,omitempty"`
}

type Video struct {
	ID                 string      `json:"id"`
	UserID             string      `json:"user_id"`
	Title              string      `json:"title"`
	Description        string      `json:"description"`
	Status             VideoStatus `json:"status"`
	Progress           int         `json:"progress"`
	OriginalPath       string      `json:"-"`
	Duration           *float64    `json:"duration,omitempty"`
	Width              *int        `json:"width,omitempty"`
	Height             *int        `json:"height,omitempty"`
	SizeBytes          int64       `json:"size_bytes"`
	ErrorMessage       *string     `json:"error_message,omitempty"`
	SeriesID           *string     `json:"series_id,omitempty"`
	SeasonNumber       *int        `json:"season_number,omitempty"`
	EpisodeNumber      *int        `json:"episode_number,omitempty"`
	VersionOf          *string     `json:"version_of,omitempty"`
	VersionLabel       *string     `json:"version_label,omitempty"`
	VersionDescription *string     `json:"version_description,omitempty"`
	CreatedAt          time.Time   `json:"created_at"`
	UpdatedAt          time.Time   `json:"updated_at"`
}

type VideoListItem struct {
	ID                 string      `json:"id"`
	UserID             string      `json:"user_id"`
	Title              string      `json:"title"`
	Status             VideoStatus `json:"status"`
	Duration           *float64    `json:"duration,omitempty"`
	Width              *int        `json:"width,omitempty"`
	Height             *int        `json:"height,omitempty"`
	SizeBytes          int64       `json:"size_bytes"`
	PreviewURL         string      `json:"preview_url,omitempty"`
	SeriesID           *string     `json:"series_id,omitempty"`
	SeasonNumber       *int        `json:"season_number,omitempty"`
	EpisodeNumber      *int        `json:"episode_number,omitempty"`
	VersionOf          *string     `json:"version_of,omitempty"`
	VersionLabel       *string     `json:"version_label,omitempty"`
	VersionDescription *string     `json:"version_description,omitempty"`
	CreatedAt          time.Time   `json:"created_at"`
}

type TranscodeTask struct {
	VideoID         string `json:"video_id"`
	OriginalPath    string `json:"original_path"`
	PreviewOverride bool   `json:"preview_override,omitempty"`
}
