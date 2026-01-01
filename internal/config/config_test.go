package config

import (
	"os"
	"testing"
)

func TestBatchProcessingDefaults(t *testing.T) {
	// Clear environment variables
	os.Unsetenv("BATCH_SIZE")
	os.Unsetenv("BATCH_DELAY_SECONDS")

	config := Load()

	if config.BatchSize != 100 {
		t.Errorf("Expected default BatchSize to be 100, got %d", config.BatchSize)
	}

	if config.BatchDelaySeconds != 10 {
		t.Errorf("Expected default BatchDelaySeconds to be 10, got %d", config.BatchDelaySeconds)
	}
}

func TestBatchProcessingCustomValues(t *testing.T) {
	// Set custom values
	os.Setenv("BATCH_SIZE", "50")
	os.Setenv("BATCH_DELAY_SECONDS", "15")
	defer func() {
		os.Unsetenv("BATCH_SIZE")
		os.Unsetenv("BATCH_DELAY_SECONDS")
	}()

	config := Load()

	if config.BatchSize != 50 {
		t.Errorf("Expected BatchSize to be 50, got %d", config.BatchSize)
	}

	if config.BatchDelaySeconds != 15 {
		t.Errorf("Expected BatchDelaySeconds to be 15, got %d", config.BatchDelaySeconds)
	}
}

func TestBatchProcessingInvalidValues(t *testing.T) {
	// Test invalid batch size
	os.Setenv("BATCH_SIZE", "0")
	os.Setenv("BATCH_DELAY_SECONDS", "10")
	defer func() {
		os.Unsetenv("BATCH_SIZE")
		os.Unsetenv("BATCH_DELAY_SECONDS")
	}()

	config := Load()

	// Should fall back to default for invalid values
	if config.BatchSize != 100 {
		t.Errorf("Expected BatchSize to fall back to default 100 for invalid value, got %d", config.BatchSize)
	}

	// Test negative delay
	os.Setenv("BATCH_SIZE", "50")
	os.Setenv("BATCH_DELAY_SECONDS", "-5")

	config = Load()

	if config.BatchDelaySeconds != 10 {
		t.Errorf("Expected BatchDelaySeconds to fall back to default 10 for negative value, got %d", config.BatchDelaySeconds)
	}
}

func TestBatchProcessingValidation(t *testing.T) {
	config := &Config{
		PlexToken:           "test-token",
		TMDbReadAccessToken: "test-tmdb-token",
		PlexServer:          "localhost",
		PlexPort:            "32400",
		UpdateField:         "label",
		ExportMode:          "txt",
		BatchSize:           0, // Invalid
		BatchDelaySeconds:   10,
	}

	err := config.Validate()
	if err == nil {
		t.Error("Expected validation error for BatchSize <= 0")
	}

	config.BatchSize = 100
	config.BatchDelaySeconds = -1 // Invalid

	err = config.Validate()
	if err == nil {
		t.Error("Expected validation error for BatchDelaySeconds < 0")
	}

	config.BatchDelaySeconds = 0 // Valid (0 means no delay)

	err = config.Validate()
	if err != nil {
		t.Errorf("Expected no validation error, got: %v", err)
	}
}

func TestGetIntEnvWithDefault(t *testing.T) {
	// Test default value when env var is not set
	os.Unsetenv("TEST_INT")
	result := getIntEnvWithDefault("TEST_INT", 42)
	if result != 42 {
		t.Errorf("Expected default value 42, got %d", result)
	}

	// Test valid integer
	os.Setenv("TEST_INT", "123")
	defer os.Unsetenv("TEST_INT")
	result = getIntEnvWithDefault("TEST_INT", 42)
	if result != 123 {
		t.Errorf("Expected parsed value 123, got %d", result)
	}

	// Test invalid integer (should return default)
	os.Setenv("TEST_INT", "invalid")
	result = getIntEnvWithDefault("TEST_INT", 42)
	if result != 42 {
		t.Errorf("Expected default value 42 for invalid input, got %d", result)
	}

	// Test zero value (should return default for batch processing)
	os.Setenv("TEST_INT", "0")
	result = getIntEnvWithDefault("TEST_INT", 42)
	if result != 42 {
		t.Errorf("Expected default value 42 for zero input, got %d", result)
	}

	// Test negative value (should return default for batch processing)
	os.Setenv("TEST_INT", "-10")
	result = getIntEnvWithDefault("TEST_INT", 42)
	if result != 42 {
		t.Errorf("Expected default value 42 for negative input, got %d", result)
	}
}

func TestBackwardCompatibility(t *testing.T) {
	// Clear all batch-related environment variables to simulate old config
	os.Unsetenv("BATCH_SIZE")
	os.Unsetenv("BATCH_DELAY_SECONDS")
	
	// Set only the required environment variables (simulating old config)
	os.Setenv("PLEX_TOKEN", "test-token")
	os.Setenv("TMDB_READ_ACCESS_TOKEN", "test-tmdb-token")
	os.Setenv("PLEX_SERVER", "localhost")
	os.Setenv("PLEX_PORT", "32400")
	defer func() {
		os.Unsetenv("PLEX_TOKEN")
		os.Unsetenv("TMDB_READ_ACCESS_TOKEN")
		os.Unsetenv("PLEX_SERVER")
		os.Unsetenv("PLEX_PORT")
	}()

	config := Load()

	// Verify that batch processing gets sensible defaults
	if config.BatchSize != 100 {
		t.Errorf("Expected default BatchSize 100 for backward compatibility, got %d", config.BatchSize)
	}

	if config.BatchDelaySeconds != 10 {
		t.Errorf("Expected default BatchDelaySeconds 10 for backward compatibility, got %d", config.BatchDelaySeconds)
	}

	// Verify that validation passes with defaults
	err := config.Validate()
	if err != nil {
		t.Errorf("Expected validation to pass with default batch settings, got error: %v", err)
	}

	// Verify that all other existing functionality is preserved
	if config.UpdateField != "label" {
		t.Errorf("Expected default UpdateField 'label', got '%s'", config.UpdateField)
	}

	if config.ExportMode != "txt" {
		t.Errorf("Expected default ExportMode 'txt', got '%s'", config.ExportMode)
	}

	if config.ProcessTimer != time.Hour {
		t.Errorf("Expected default ProcessTimer 1h, got %v", config.ProcessTimer)
	}
}