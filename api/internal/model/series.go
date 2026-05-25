package model

import "time"

type Series struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	EpisodeCount int       `json:"episode_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Episode struct {
	ID            string         `json:"id"`
	Title         string         `json:"title"`
	SeasonNumber  *int           `json:"season_number,omitempty"`
	EpisodeNumber *int           `json:"episode_number,omitempty"`
	Duration      *float64       `json:"duration,omitempty"`
	Status        VideoStatus    `json:"status"`
	PreviewURL    string         `json:"preview_url,omitempty"`
	Versions      []VideoVersion `json:"versions,omitempty"`
}

type Season struct {
	SeasonNumber int       `json:"season_number"`
	Episodes     []Episode `json:"episodes"`
}

// SeriesDetail is returned by GET /series/{id}. Episodes with a season_number are
// grouped into Seasons; episodes without one appear in the top-level Episodes slice.
type SeriesDetail struct {
	Series
	Seasons  []Season  `json:"seasons"`
	Episodes []Episode `json:"episodes"`
}
