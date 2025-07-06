package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ProcessedItem represents an item that has been processed
type ProcessedItem struct {
	RatingKey      string    `json:"ratingKey"`
	Title          string    `json:"title"`
	TMDbID         string    `json:"tmdbId"`
	LastProcessed  time.Time `json:"lastProcessed"`
	KeywordsSynced bool      `json:"keywordsSynced"`
	UpdateField    string    `json:"updateField"`
}

// Storage handles persistent storage of processed items
type Storage struct {
	filePath string
	data     map[string]*ProcessedItem
	mutex    sync.RWMutex
}

// NewStorage creates a new storage instance
func NewStorage(dataDir string) (*Storage, error) {
	// Create data directory if it doesn't exist
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}
	
	filePath := filepath.Join(dataDir, "processed_items.json")
	
	s := &Storage{
		filePath: filePath,
		data:     make(map[string]*ProcessedItem),
	}
	
	// Load existing data
	if err := s.load(); err != nil {
		// If file doesn't exist, that's OK - we'll create it
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load existing data: %w", err)
		}
	}
	
	return s, nil
}

// load reads data from the JSON file
func (s *Storage) load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}
	
	return json.Unmarshal(data, &s.data)
}

// save writes data to the JSON file
func (s *Storage) save() error {
	data, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	
	// Write to temp file first, then rename (atomic operation)
	tempFile := s.filePath + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return err
	}
	
	return os.Rename(tempFile, s.filePath)
}

// Get retrieves a processed item by rating key
func (s *Storage) Get(ratingKey string) (*ProcessedItem, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	item, exists := s.data[ratingKey]
	return item, exists
}

// Set stores a processed item
func (s *Storage) Set(item *ProcessedItem) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	s.data[item.RatingKey] = item
	
	return s.save()
}

// GetAll returns all processed items
func (s *Storage) GetAll() map[string]*ProcessedItem {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	// Return a copy to avoid race conditions
	result := make(map[string]*ProcessedItem)
	for k, v := range s.data {
		result[k] = v
	}
	
	return result
}

// Count returns the number of processed items
func (s *Storage) Count() int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	return len(s.data)
}

// Cleanup removes old processed items (older than specified duration)
func (s *Storage) Cleanup(maxAge time.Duration) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	cutoff := time.Now().Add(-maxAge)
	
	for key, item := range s.data {
		if item.LastProcessed.Before(cutoff) {
			delete(s.data, key)
		}
	}
	
	return s.save()
}