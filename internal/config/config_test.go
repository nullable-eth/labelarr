package config

import (
	"os"
	"testing"
	"time"
)

func TestBatchProcessingDefaults(t *testing.T) {
	os.Unsetenv("BATCH_SIZE")
	os.Unsetenv("BATCH_DELAY")
	os.Unsetenv("ITEM_DELAY")

	config := Load()

	if config.BatchSize != 100 {
		t.Errorf("Expected default BatchSize to be 100, got %d", config.BatchSize)
	}

	if config.BatchDelay != 10*time.Second {
		t.Errorf("Expected default BatchDelay to be 10s, got %v", config.BatchDelay)
	}

	if config.ItemDelay != 500*time.Millisecond {
		t.Errorf("Expected default ItemDelay to be 500ms, got %v", config.ItemDelay)
	}
}

func TestBatchProcessingCustomValues(t *testing.T) {
	os.Setenv("BATCH_SIZE", "50")
	os.Setenv("BATCH_DELAY", "15s")
	os.Setenv("ITEM_DELAY", "200ms")
	defer func() {
		os.Unsetenv("BATCH_SIZE")
		os.Unsetenv("BATCH_DELAY")
		os.Unsetenv("ITEM_DELAY")
	}()

	config := Load()

	if config.BatchSize != 50 {
		t.Errorf("Expected BatchSize to be 50, got %d", config.BatchSize)
	}

	if config.BatchDelay != 15*time.Second {
		t.Errorf("Expected BatchDelay to be 15s, got %v", config.BatchDelay)
	}

	if config.ItemDelay != 200*time.Millisecond {
		t.Errorf("Expected ItemDelay to be 200ms, got %v", config.ItemDelay)
	}
}

func TestBatchProcessingInvalidValues(t *testing.T) {
	// BATCH_SIZE=0 should fall back to the default via getIntEnvWithDefault
	os.Setenv("BATCH_SIZE", "0")
	os.Setenv("BATCH_DELAY", "10s")
	defer func() {
		os.Unsetenv("BATCH_SIZE")
		os.Unsetenv("BATCH_DELAY")
	}()

	config := Load()

	if config.BatchSize != 100 {
		t.Errorf("Expected BatchSize to fall back to default 100 for invalid value, got %d", config.BatchSize)
	}

	// Unparseable BATCH_DELAY should fall back to the default
	os.Setenv("BATCH_SIZE", "50")
	os.Setenv("BATCH_DELAY", "nonsense")

	config = Load()

	if config.BatchDelay != 10*time.Second {
		t.Errorf("Expected BatchDelay to fall back to default 10s for invalid value, got %v", config.BatchDelay)
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
		BatchDelay:          10 * time.Second,
		ItemDelay:           500 * time.Millisecond,
	}

	err := config.Validate()
	if err == nil {
		t.Error("Expected validation error for BatchSize <= 0")
	}

	config.BatchSize = 100
	config.BatchDelay = -1 * time.Second // Invalid

	err = config.Validate()
	if err == nil {
		t.Error("Expected validation error for BatchDelay < 0")
	}

	config.BatchDelay = 0 // Valid (0 means no delay)

	err = config.Validate()
	if err != nil {
		t.Errorf("Expected no validation error, got: %v", err)
	}

	config.ItemDelay = -1 * time.Millisecond // Invalid
	err = config.Validate()
	if err == nil {
		t.Error("Expected validation error for ItemDelay < 0")
	}
}

func TestGetIntEnvWithDefault(t *testing.T) {
	os.Unsetenv("TEST_INT")
	result := getIntEnvWithDefault("TEST_INT", 42)
	if result != 42 {
		t.Errorf("Expected default value 42, got %d", result)
	}

	os.Setenv("TEST_INT", "123")
	defer os.Unsetenv("TEST_INT")
	result = getIntEnvWithDefault("TEST_INT", 42)
	if result != 123 {
		t.Errorf("Expected parsed value 123, got %d", result)
	}

	os.Setenv("TEST_INT", "invalid")
	result = getIntEnvWithDefault("TEST_INT", 42)
	if result != 42 {
		t.Errorf("Expected default value 42 for invalid input, got %d", result)
	}

	// Zero should fall back to default (positive-value guarantee)
	os.Setenv("TEST_INT", "0")
	result = getIntEnvWithDefault("TEST_INT", 42)
	if result != 42 {
		t.Errorf("Expected default value 42 for zero input, got %d", result)
	}

	// Negative should fall back to default
	os.Setenv("TEST_INT", "-10")
	result = getIntEnvWithDefault("TEST_INT", 42)
	if result != 42 {
		t.Errorf("Expected default value 42 for negative input, got %d", result)
	}
}

func TestBackwardCompatibility(t *testing.T) {
	os.Unsetenv("BATCH_SIZE")
	os.Unsetenv("BATCH_DELAY")
	os.Unsetenv("ITEM_DELAY")

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

	if config.BatchSize != 100 {
		t.Errorf("Expected default BatchSize 100, got %d", config.BatchSize)
	}

	if config.BatchDelay != 10*time.Second {
		t.Errorf("Expected default BatchDelay 10s, got %v", config.BatchDelay)
	}

	err := config.Validate()
	if err != nil {
		t.Errorf("Expected validation to pass with default batch settings, got error: %v", err)
	}

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
