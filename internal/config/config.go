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
	Protocol               string
	PlexInsecureSkipVerify bool
	PlexServer             string
	PlexPort               string
	PlexToken              string
	MovieLibraryID         string
	MovieProcessAll        bool
	MovieLibraryExclude    []string
	TVLibraryID            string
	TVProcessAll           bool
	TVLibraryExclude       []string
	ExcludeLabels          []string
	WebhookOnly            bool
	UpdateField            string
	RemoveMode             string
	TMDbReadAccessToken    string
	ProcessTimer           time.Duration

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

	// Webhook configuration
	WebhookEnabled  bool
	WebhookPort     int
	WebhookDebounce time.Duration

	// Keyword prefix configuration
	KeywordPrefix string

	// Batch processing configuration
	BatchSize  int
	BatchDelay time.Duration
	ItemDelay  time.Duration

	// Export configuration
	ExportLabels   []string
	ExportLocation string
	ExportMode     string
}

// Load loads configuration from environment variables
func Load() *Config {
	config := &Config{
		PlexInsecureSkipVerify: getBoolEnvWithDefault("PLEX_INSECURE_SKIP_VERIFY", false),
		PlexServer:             os.Getenv("PLEX_SERVER"),
		PlexPort:               os.Getenv("PLEX_PORT"),
		PlexToken:              os.Getenv("PLEX_TOKEN"),
		MovieLibraryID:         os.Getenv("MOVIE_LIBRARY_ID"),
		MovieProcessAll:        getBoolEnvWithDefault("MOVIE_PROCESS_ALL", false),
		MovieLibraryExclude:    parseCSV(os.Getenv("MOVIE_LIBRARY_EXCLUDE")),
		TVLibraryID:            os.Getenv("TV_LIBRARY_ID"),
		TVProcessAll:           getBoolEnvWithDefault("TV_PROCESS_ALL", false),
		TVLibraryExclude:       parseCSV(os.Getenv("TV_LIBRARY_EXCLUDE")),
		ExcludeLabels:          parseCSV(os.Getenv("EXCLUDE_LABELS")),
		WebhookOnly:            getBoolEnvWithDefault("WEBHOOK_ONLY", false),
		UpdateField:            getEnvWithDefault("UPDATE_FIELD", "label"),
		RemoveMode:             os.Getenv("REMOVE"),
		TMDbReadAccessToken:    os.Getenv("TMDB_READ_ACCESS_TOKEN"),
		ProcessTimer:           getDurationEnvWithDefault("PROCESS_TIMER", "1h"),

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

		// Webhook configuration
		WebhookEnabled:  getBoolEnvWithDefault("WEBHOOK_ENABLED", false),
		WebhookPort:     getIntEnvWithDefault("WEBHOOK_PORT", 9090),
		WebhookDebounce: getDurationEnvWithDefault("WEBHOOK_DEBOUNCE", "30s"),

		// Keyword prefix configuration
		KeywordPrefix: os.Getenv("KEYWORD_PREFIX"),

		// Batch processing configuration
		BatchSize:  getIntEnvWithDefault("BATCH_SIZE", 100),
		BatchDelay: getDurationEnvWithDefault("BATCH_DELAY", "10s"),
		ItemDelay:  getDurationEnvWithDefault("ITEM_DELAY", "500ms"),

		// Export configuration
		ExportLabels:   parseCSV(os.Getenv("EXPORT_LABELS")),
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
	if c.ExportMode != "txt" && c.ExportMode != "json" {
		return fmt.Errorf("EXPORT_MODE must be 'txt' or 'json'")
	}
	if c.WebhookOnly && !c.WebhookEnabled {
		return fmt.Errorf("WEBHOOK_ONLY=true requires WEBHOOK_ENABLED=true")
	}

	if c.WebhookEnabled && (c.WebhookPort < 1 || c.WebhookPort > 65535) {
		return fmt.Errorf("WEBHOOK_PORT must be between 1 and 65535")
	}
	if c.BatchSize < 1 {
		return fmt.Errorf("BATCH_SIZE must be at least 1")
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

	// Validate batch processing configuration
	if c.BatchSize <= 0 {
		return fmt.Errorf("BATCH_SIZE must be greater than 0")
	}
	if c.BatchDelay < 0 {
		return fmt.Errorf("BATCH_DELAY must be 0 or greater")
	}
	if c.ItemDelay < 0 {
		return fmt.Errorf("ITEM_DELAY must be 0 or greater")
	}

	return nil
}

func getEnvWithDefault(envVar, defaultValue string) string {
	if value := os.Getenv(envVar); value != "" {
		return value
	}
	return defaultValue
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

func getIntEnvWithDefault(envVar string, defaultValue int) int {
	value := os.Getenv(envVar)
	if value == "" {
		return defaultValue
	}
	result, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	// Ensure positive values (e.g. BATCH_SIZE, WEBHOOK_PORT must not be <= 0)
	if result <= 0 {
		return defaultValue
	}
	return result
}

func getDurationEnvWithDefault(envVar string, defaultValue string) time.Duration {
	value := getEnvWithDefault(envVar, defaultValue)
	duration, err := time.ParseDuration(value)
	if err != nil {
		fallback, _ := time.ParseDuration(defaultValue)
		return fallback
	}
	return duration
}

func parseCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var out []string
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

// HasExportEnabled returns true if export functionality is enabled
func (c *Config) HasExportEnabled() bool {
	return len(c.ExportLabels) > 0 && c.ExportLocation != ""
}
