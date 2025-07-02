package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration
type Config struct {
	Protocol            string
	PlexServer          string
	PlexPort            string
	PlexToken           string
	MovieLibraryID      string
	MovieProcessAll     bool
	TVLibraryID         string
	TVProcessAll        bool
	UpdateField         string
	RemoveMode          string
	TMDbReadAccessToken string
	ProcessTimer        time.Duration
}

// Load loads configuration from environment variables
func Load() *Config {
	config := &Config{
		PlexServer:          os.Getenv("PLEX_SERVER"),
		PlexPort:            os.Getenv("PLEX_PORT"),
		PlexToken:           os.Getenv("PLEX_TOKEN"),
		MovieLibraryID:      os.Getenv("MOVIE_LIBRARY_ID"),
		MovieProcessAll:     getBoolEnvWithDefault("MOVIE_PROCESS_ALL", false),
		TVLibraryID:         os.Getenv("TV_LIBRARY_ID"),
		TVProcessAll:        getBoolEnvWithDefault("TV_PROCESS_ALL", false),
		UpdateField:         getEnvWithDefault("UPDATE_FIELD", "label"),
		RemoveMode:          os.Getenv("REMOVE"),
		TMDbReadAccessToken: os.Getenv("TMDB_READ_ACCESS_TOKEN"),
		ProcessTimer:        getProcessTimerFromEnv(),
	}

	// Set protocol based on HTTPS requirement
	if getBoolEnvWithDefault("PLEX_REQUIRES_HTTPS", false) {
		config.Protocol = "https"
	} else {
		config.Protocol = "http"
	}

	return config
}

// ProcessMovies returns true if movies should be processed
func (c *Config) ProcessMovies() bool {
	return c.MovieLibraryID != "" || c.MovieProcessAll
}

// ProcessTVShows returns true if TV shows should be processed
func (c *Config) ProcessTVShows() bool {
	return c.TVLibraryID != "" || c.TVProcessAll
}

// IsRemoveMode returns true if the application is in remove mode
func (c *Config) IsRemoveMode() bool {
	return c.RemoveMode != ""
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.PlexToken == "" {
		return fmt.Errorf("PLEX_TOKEN environment variable is required")
	}
	if c.TMDbReadAccessToken == "" {
		return fmt.Errorf("TMDB_READ_ACCESS_TOKEN environment variable is required")
	}
	if c.PlexServer == "" {
		return fmt.Errorf("PLEX_SERVER environment variable is required")
	}
	if c.PlexPort == "" {
		return fmt.Errorf("PLEX_PORT environment variable is required")
	}
	if c.UpdateField != "label" && c.UpdateField != "genre" {
		return fmt.Errorf("UPDATE_FIELD must be 'label' or 'genre'")
	}
	if c.RemoveMode != "" && c.RemoveMode != "lock" && c.RemoveMode != "unlock" {
		return fmt.Errorf("REMOVE must be 'lock' or 'unlock'")
	}
	return nil
}

func getEnvWithDefault(envVar, defaultValue string) string {
	if value := os.Getenv(envVar); value != "" {
		return value
	}
	return defaultValue
}

func getProcessTimerFromEnv() time.Duration {
	timerStr := getEnvWithDefault("PROCESS_TIMER", "1h")
	timer, err := time.ParseDuration(timerStr)
	if err != nil {
		return 5 * time.Minute
	}
	return timer
}

func getBoolEnvWithDefault(envVar string, defaultValue bool) bool {
	value := os.Getenv(envVar)
	if value == "" {
		return defaultValue
	}
	result, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return result
}
