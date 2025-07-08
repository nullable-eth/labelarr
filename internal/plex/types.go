package plex

import (
	"encoding/json"
	"fmt"
)

// Library represents a Plex library
type Library struct {
	Key   string `json:"key"`
	Type  string `json:"type"`
	Title string `json:"title"`
	Agent string `json:"agent"`
}

// LibraryContainer holds library directory information
type LibraryContainer struct {
	Size      int       `json:"size"`
	Directory []Library `json:"Directory"`
}

// LibraryResponse represents the response from library endpoints
type LibraryResponse struct {
	MediaContainer LibraryContainer `json:"MediaContainer"`
}

// Movie represents a Plex movie
type Movie struct {
	RatingKey string       `json:"ratingKey"`
	Title     string       `json:"title"`
	Year      int          `json:"year"`
	Label     []Label      `json:"Label,omitempty"`
	Genre     []Genre      `json:"Genre,omitempty"`
	Guid      FlexibleGuid `json:"Guid,omitempty"`
	Media     []Media      `json:"Media,omitempty"`
}

// MediaItem interface implementation for Movie
func (m Movie) GetRatingKey() string { return m.RatingKey }
func (m Movie) GetTitle() string     { return m.Title }
func (m Movie) GetYear() int         { return m.Year }
func (m Movie) GetGuid() []Guid      { return []Guid(m.Guid) }
func (m Movie) GetMedia() []Media    { return m.Media }
func (m Movie) GetLabel() []Label    { return m.Label }
func (m Movie) GetGenre() []Genre    { return m.Genre }

// TVShow represents a Plex TV show
type TVShow struct {
	RatingKey string       `json:"ratingKey"`
	Title     string       `json:"title"`
	Year      int          `json:"year"`
	Label     []Label      `json:"Label,omitempty"`
	Genre     []Genre      `json:"Genre,omitempty"`
	Guid      FlexibleGuid `json:"Guid,omitempty"`
	Media     []Media      `json:"Media,omitempty"`
}

// MediaItem interface implementation for TVShow
func (t TVShow) GetRatingKey() string { return t.RatingKey }
func (t TVShow) GetTitle() string     { return t.Title }
func (t TVShow) GetYear() int         { return t.Year }
func (t TVShow) GetGuid() []Guid      { return []Guid(t.Guid) }
func (t TVShow) GetMedia() []Media    { return t.Media }
func (t TVShow) GetLabel() []Label    { return t.Label }
func (t TVShow) GetGenre() []Genre    { return t.Genre }

// Label represents a Plex label
type Label struct {
	Tag string `json:"tag"`
}

// Genre represents a Plex genre
type Genre struct {
	Tag string `json:"tag"`
}

// Guid represents a Plex GUID
type Guid struct {
	ID string `json:"id"`
}

// Media represents Plex media information
type Media struct {
	Part []Part `json:"Part,omitempty"`
}

// Part represents a media part with file information
type Part struct {
	File string `json:"file,omitempty"`
	Size int64  `json:"size,omitempty"`
}

// FlexibleGuid handles both string and array formats from Plex API
type FlexibleGuid []Guid

func (fg *FlexibleGuid) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as array first
	var guidArray []Guid
	if err := json.Unmarshal(data, &guidArray); err == nil {
		*fg = FlexibleGuid(guidArray)
		return nil
	}

	// If that fails, try as single string
	var guidString string
	if err := json.Unmarshal(data, &guidString); err == nil {
		*fg = FlexibleGuid([]Guid{{ID: guidString}})
		return nil
	}

	// If both fail, try as single Guid object
	var singleGuid Guid
	if err := json.Unmarshal(data, &singleGuid); err == nil {
		*fg = FlexibleGuid([]Guid{singleGuid})
		return nil
	}

	return fmt.Errorf("cannot unmarshal Guid field")
}

// MediaContainer holds metadata for movies or TV shows
type MediaContainer struct {
	Size     int     `json:"size"`
	Metadata []Movie `json:"Metadata"`
}

// TVShowContainer holds metadata for TV shows
type TVShowContainer struct {
	Size     int      `json:"size"`
	Metadata []TVShow `json:"Metadata"`
}

// PlexResponse represents a standard Plex API response for movies
type PlexResponse struct {
	MediaContainer MediaContainer `json:"MediaContainer"`
}

// TVShowResponse represents a Plex API response for TV shows
type TVShowResponse struct {
	MediaContainer TVShowContainer `json:"MediaContainer"`
}

// Episode represents a Plex TV show episode
type Episode struct {
	RatingKey string  `json:"ratingKey"`
	Title     string  `json:"title"`
	Media     []Media `json:"Media,omitempty"`
}

// EpisodeContainer holds metadata for episodes
type EpisodeContainer struct {
	Size     int       `json:"size"`
	Metadata []Episode `json:"Metadata"`
}

// EpisodeResponse represents a Plex API response for episodes
type EpisodeResponse struct {
	MediaContainer EpisodeContainer `json:"MediaContainer"`
}
