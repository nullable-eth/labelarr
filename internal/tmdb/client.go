package tmdb

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/nullable-eth/labelarr/internal/config"
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
		return nil, fmt.Errorf("tmdb API returned status %d for movie %s", resp.StatusCode, tmdbID)
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

	return keywords, nil
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
		return nil, fmt.Errorf("tmdb API returned status %d for TV show %s", resp.StatusCode, tmdbID)
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

	return keywords, nil
}
