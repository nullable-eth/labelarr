package models

import (
	"time"
)

type Media struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	PlexID    string    `gorm:"uniqueIndex" json:"plex_id"`
	Title     string    `json:"title"`
	Year      int       `json:"year"`
	Type      string    `json:"type"` // "movie" or "tv"
	FilePath  string    `json:"file_path"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Labels    []Label   `gorm:"many2many:media_labels;" json:"labels"`
}

type Label struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"uniqueIndex" json:"name"`
	Description string    `json:"description"`
	Color       string    `json:"color"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Media       []Media   `gorm:"many2many:media_labels;" json:"media"`
}

type MediaLabel struct {
	MediaID uint `gorm:"primaryKey"`
	LabelID uint `gorm:"primaryKey"`
}
