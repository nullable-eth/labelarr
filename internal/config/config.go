package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all application configuration
type Config struct {
	Protocol        string
	PlexServer      string
	PlexPort        string
	PlexToken       string
	MovieLibraryID  string
	MovieProcessAll bool
	TVLibraryID     string
	TVProcessAll    bool
	UpdateField     string
	RemoveMode      string
	ProcessTimer    time.Duration

	// Radarr configuration
	RadarrURL    string
	RadarrAPIKey string
	UseRadarr    bool

	// Sonarr configuration
	SonarrURL    string
	SonarrAPIKey string
	UseSonarr    bool

	// Logging configuration
	VerboseLogging bool

	// Storage configuration
	DataDir string

	// Force update configuration
	ForceUpdate bool

	// Export configuration
	ExportLabels   []string
	ExportLocation string
	ExportMode     string
}

// Load loads configuration from environment variables
func Load() *Config {
	config := &Config{
		PlexServer:      os.Getenv("PLEX_SERVER"),
		PlexPort:        os.Getenv("PLEX_PORT"),
		PlexToken:       os.Getenv("PLEX_TOKEN"),
		MovieLibraryID:  os.Getenv("MOVIE_LIBRARY_ID"),
		MovieProcessAll: getBoolEnvWithDefault("MOVIE_PROCESS_ALL", false),
		TVLibraryID:     os.Getenv("TV_LIBRARY_ID"),
		TVProcessAll:    getBoolEnvWithDefault("TV_PROCESS_ALL", false),
		UpdateField:     getEnvWithDefault("UPDATE_FIELD", "label"),
		RemoveMode:      os.Getenv("REMOVE"),
		ProcessTimer:    getProcessTimerFromEnv(),

		// Radarr configuration
		RadarrURL:    os.Getenv("RADARR_URL"),
		RadarrAPIKey: os.Getenv("RADARR_API_KEY"),
		UseRadarr:    getBoolEnvWithDefault("USE_RADARR", false),

		// Sonarr configuration
		SonarrURL:    os.Getenv("SONARR_URL"),
		SonarrAPIKey: os.Getenv("SONARR_API_KEY"),
		UseSonarr:    getBoolEnvWithDefault("USE_SONARR", false),

		// Logging configuration
		VerboseLogging: getBoolEnvWithDefault("VERBOSE_LOGGING", false),

		// Storage configuration
		DataDir: os.Getenv("DATA_DIR"), // No default - ephemeral if not set

		// Force update configuration
		ForceUpdate: getBoolEnvWithDefault("FORCE_UPDATE", false),

		// Export configuration
		ExportLabels:   parseExportLabels(os.Getenv("EXPORT_LABELS")),
		ExportLocation: os.Getenv("EXPORT_LOCATION"),
		ExportMode:     getEnvWithDefault("EXPORT_MODE", "txt"),
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
	if c.ExportMode != "txt" && c.ExportMode != "json" {
		return fmt.Errorf("EXPORT_MODE must be 'txt' or 'json'")
	}

	// Validate Radarr configuration if enabled
	if c.UseRadarr {
		if c.RadarrURL == "" {
			return fmt.Errorf("RADARR_URL environment variable is required when USE_RADARR is true")
		}
		if c.RadarrAPIKey == "" {
			return fmt.Errorf("RADARR_API_KEY environment variable is required when USE_RADARR is true")
		}
	}

	// Validate Sonarr configuration if enabled
	if c.UseSonarr {
		if c.SonarrURL == "" {
			return fmt.Errorf("SONARR_URL environment variable is required when USE_SONARR is true")
		}
		if c.SonarrAPIKey == "" {
			return fmt.Errorf("SONARR_API_KEY environment variable is required when USE_SONARR is true")
		}
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

func parseExportLabels(labels string) []string {
	if labels == "" {
		return nil
	}
	// Split and trim whitespace from each label
	rawLabels := strings.Split(labels, ",")
	var cleanLabels []string
	for _, label := range rawLabels {
		trimmed := strings.TrimSpace(label)
		if trimmed != "" {
			cleanLabels = append(cleanLabels, trimmed)
		}
	}
	return cleanLabels
}

// HasExportEnabled returns true if export functionality is enabled
func (c *Config) HasExportEnabled() bool {
	return len(c.ExportLabels) > 0 && c.ExportLocation != ""
}
