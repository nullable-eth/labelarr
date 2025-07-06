package tmdb

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/nullable-eth/labelarr/internal/config"
	"github.com/nullable-eth/labelarr/internal/utils"
)

// Client represents a TMDb API client
type Client struct {
	config     *config.Config
	httpClient *http.Client
}

// NewClient creates a new TMDb client
func NewClient(cfg *config.Config) *Client {
	return &Client{
		config:     cfg,
		httpClient: &http.Client{},
	}
}

// GetMovieKeywords fetches keywords for a movie from TMDb
func (c *Client) GetMovieKeywords(tmdbID string) ([]string, error) {
	keywordsURL := fmt.Sprintf("https://api.themoviedb.org/3/movie/%s/keywords", tmdbID)

	req, err := http.NewRequest("GET", keywordsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.config.TMDbReadAccessToken))
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch movie keywords: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		time.Sleep(1 * time.Second)
		return c.GetMovieKeywords(tmdbID)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusUnauthorized {
			return nil, fmt.Errorf("tmdb API authentication failed (status 401) - check your TMDB_READ_ACCESS_TOKEN. Response: %s", string(body))
		}
		return nil, fmt.Errorf("tmdb API returned status %d for movie %s. Response: %s", resp.StatusCode, tmdbID, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var keywordsResponse KeywordsResponse
	if err := json.Unmarshal(body, &keywordsResponse); err != nil {
		return nil, fmt.Errorf("failed to parse keywords response: %w", err)
	}

	keywords := make([]string, len(keywordsResponse.Keywords))
	for i, keyword := range keywordsResponse.Keywords {
		keywords[i] = keyword.Name
	}

	// Normalize keywords for proper capitalization and spelling
	normalizedKeywords := utils.NormalizeKeywords(keywords)
	
	// Show normalization in verbose mode
	if c.config.VerboseLogging {
		for i, original := range keywords {
			if i < len(normalizedKeywords) && original != normalizedKeywords[i] {
				fmt.Printf("   📝 Normalized: \"%s\" → \"%s\"\n", original, normalizedKeywords[i])
			}
		}
	}

	return normalizedKeywords, nil
}

// GetTVShowKeywords fetches keywords for a TV show from TMDb
func (c *Client) GetTVShowKeywords(tmdbID string) ([]string, error) {
	keywordsURL := fmt.Sprintf("https://api.themoviedb.org/3/tv/%s/keywords", tmdbID)

	req, err := http.NewRequest("GET", keywordsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.config.TMDbReadAccessToken))
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch TV show keywords: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		time.Sleep(1 * time.Second)
		return c.GetTVShowKeywords(tmdbID)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusUnauthorized {
			return nil, fmt.Errorf("tmdb API authentication failed (status 401) - check your TMDB_READ_ACCESS_TOKEN. Response: %s", string(body))
		}
		return nil, fmt.Errorf("tmdb API returned status %d for TV show %s. Response: %s", resp.StatusCode, tmdbID, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var tvKeywordsResponse TVKeywordsResponse
	if err := json.Unmarshal(body, &tvKeywordsResponse); err != nil {
		return nil, fmt.Errorf("failed to parse TV keywords response: %w", err)
	}

	keywords := make([]string, len(tvKeywordsResponse.Results))
	for i, keyword := range tvKeywordsResponse.Results {
		keywords[i] = keyword.Name
	}

	// Normalize keywords for proper capitalization and spelling
	normalizedKeywords := utils.NormalizeKeywords(keywords)
	
	// Show normalization in verbose mode
	if c.config.VerboseLogging {
		for i, original := range keywords {
			if i < len(normalizedKeywords) && original != normalizedKeywords[i] {
				fmt.Printf("   📝 Normalized: \"%s\" → \"%s\"\n", original, normalizedKeywords[i])
			}
		}
	}

	return normalizedKeywords, nil
}

// TestConnection tests the TMDb API connection
func (c *Client) TestConnection() error {
	// Test with a known movie ID (The Godfather)
	testURL := "https://api.themoviedb.org/3/movie/238/keywords"
	
	req, err := http.NewRequest("GET", testURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create test request: %w", err)
	}
	
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.config.TMDbReadAccessToken))
	req.Header.Set("Accept", "application/json")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to TMDb API: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == http.StatusUnauthorized {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("TMDb API authentication failed - invalid TMDB_READ_ACCESS_TOKEN. Response: %s", string(body))
	}
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("TMDb API test failed with status %d. Response: %s", resp.StatusCode, string(body))
	}
	
	return nil
}
