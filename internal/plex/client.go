package plex

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/nullable-eth/labelarr/internal/config"
)

// Client represents a Plex API client
type Client struct {
	config     *config.Config
	httpClient *http.Client
}

// NewClient creates a new Plex client
func NewClient(cfg *config.Config) *Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	return &Client{
		config:     cfg,
		httpClient: &http.Client{Transport: tr},
	}
}

// GetAllLibraries fetches all libraries from Plex
func (c *Client) GetAllLibraries() ([]Library, error) {
	librariesURL := c.buildURL(fmt.Sprintf("/library/sections?X-Plex-Token=%s", c.config.PlexToken))

	req, err := http.NewRequest("GET", librariesURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("X-Plex-Token", c.config.PlexToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch libraries: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("plex API returned status %d. Response: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var libraryResponse LibraryResponse
	if err := json.Unmarshal(body, &libraryResponse); err != nil {
		return nil, fmt.Errorf("failed to parse library response: %w. Response body: %s", err, string(body))
	}

	return libraryResponse.MediaContainer.Directory, nil
}

// GetMoviesFromLibrary fetches all movies from a specific library
func (c *Client) GetMoviesFromLibrary(libraryID string) ([]Movie, error) {
	moviesURL := c.buildURL(fmt.Sprintf("/library/sections/%s/all", libraryID))

	req, err := http.NewRequest("GET", moviesURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("X-Plex-Token", c.config.PlexToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch movies: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("plex API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var plexResponse PlexResponse
	if err := json.Unmarshal(body, &plexResponse); err != nil {
		return nil, fmt.Errorf("failed to parse movies response: %w", err)
	}

	return plexResponse.MediaContainer.Metadata, nil
}

// GetMovieDetails fetches detailed information for a specific movie
func (c *Client) GetMovieDetails(ratingKey string) (*Movie, error) {
	movieURL := c.buildURL(fmt.Sprintf("/library/metadata/%s", ratingKey))

	req, err := http.NewRequest("GET", movieURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("X-Plex-Token", c.config.PlexToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch movie details: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("plex API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var plexResponse PlexResponse
	if err := json.Unmarshal(body, &plexResponse); err != nil {
		return nil, fmt.Errorf("failed to parse movie details: %w", err)
	}

	if len(plexResponse.MediaContainer.Metadata) == 0 {
		return nil, fmt.Errorf("no movie found with rating key %s", ratingKey)
	}

	return &plexResponse.MediaContainer.Metadata[0], nil
}

// UpdateMediaField updates a media item's field (labels or genres) with new keywords
func (c *Client) UpdateMediaField(mediaID, libraryID string, keywords []string, updateField string, mediaType string) error {
	if c.config.VerboseLogging {
		fmt.Printf("   🌐 Making Plex API call to update %s field with %d keywords\n", updateField, len(keywords))
	}
	return c.updateMediaField(mediaID, libraryID, keywords, updateField, c.getMediaTypeForLibraryType(mediaType))
}

// RemoveMediaFieldKeywords removes keywords from a media item's field
func (c *Client) RemoveMediaFieldKeywords(mediaID, libraryID string, valuesToRemove []string, updateField string, lockField bool, mediaType string) error {
	return c.removeMediaFieldKeywords(mediaID, libraryID, valuesToRemove, updateField, lockField, c.getMediaTypeForLibraryType(mediaType))
}

// GetTVShowsFromLibrary fetches all TV shows from a specific library
func (c *Client) GetTVShowsFromLibrary(libraryID string) ([]TVShow, error) {
	tvShowsURL := c.buildURL(fmt.Sprintf("/library/sections/%s/all", libraryID))

	req, err := http.NewRequest("GET", tvShowsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("X-Plex-Token", c.config.PlexToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch TV shows: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("plex API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var tvShowResponse TVShowResponse
	if err := json.Unmarshal(body, &tvShowResponse); err != nil {
		return nil, fmt.Errorf("failed to parse TV shows response: %w", err)
	}

	return tvShowResponse.MediaContainer.Metadata, nil
}

// GetTVShowDetails fetches detailed information for a specific TV show
func (c *Client) GetTVShowDetails(ratingKey string) (*TVShow, error) {
	tvShowURL := c.buildURL(fmt.Sprintf("/library/metadata/%s", ratingKey))

	req, err := http.NewRequest("GET", tvShowURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("X-Plex-Token", c.config.PlexToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch TV show details: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("plex API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var tvShowResponse TVShowResponse
	if err := json.Unmarshal(body, &tvShowResponse); err != nil {
		return nil, fmt.Errorf("failed to parse TV show details: %w", err)
	}

	if len(tvShowResponse.MediaContainer.Metadata) == 0 {
		return nil, fmt.Errorf("no TV show found with rating key %s", ratingKey)
	}

	return &tvShowResponse.MediaContainer.Metadata[0], nil
}

// GetTVShowEpisodes fetches episodes for a specific TV show (limited for TMDb ID extraction)
func (c *Client) GetTVShowEpisodes(ratingKey string) ([]Episode, error) {
	episodesURL := c.buildURL(fmt.Sprintf("/library/metadata/%s/allLeaves?X-Plex-Container-Start=0&X-Plex-Container-Size=10", ratingKey))

	req, err := http.NewRequest("GET", episodesURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("X-Plex-Token", c.config.PlexToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch TV show episodes: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("plex API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var episodeResponse EpisodeResponse
	if err := json.Unmarshal(body, &episodeResponse); err != nil {
		return nil, fmt.Errorf("failed to parse episodes response: %w", err)
	}

	return episodeResponse.MediaContainer.Metadata, nil
}

// updateMediaField is a generic function to update media fields (movies: type=1, TV shows: type=2)
func (c *Client) updateMediaField(mediaID, libraryID string, keywords []string, updateField string, mediaType int) error {
	startTime := time.Now()
	
	// Build the base URL
	baseURL := c.buildURL(fmt.Sprintf("/library/sections/%s/all", libraryID))

	// Parse the URL to add query parameters properly
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return fmt.Errorf("failed to parse URL: %w", err)
	}

	// Create query parameters
	params := parsedURL.Query()
	params.Set("type", fmt.Sprintf("%d", mediaType))
	params.Set("id", mediaID)
	params.Set("includeExternalMedia", "1")

	// Add indexed label/genre parameters like label[0].tag.tag, label[1].tag.tag, etc.
	for i, keyword := range keywords {
		paramName := fmt.Sprintf("%s[%d].tag.tag", updateField, i)
		params.Set(paramName, keyword)
	}

	params.Set(fmt.Sprintf("%s.locked", updateField), "1")

	// Add the Plex token
	params.Set("X-Plex-Token", c.config.PlexToken)

	// Set the query parameters back to the URL
	parsedURL.RawQuery = params.Encode()

	req, err := http.NewRequest("PUT", parsedURL.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update media field: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("plex API returned status %d when updating media field - Response: %s", resp.StatusCode, string(body))
	}
	
	if c.config.VerboseLogging {
		duration := time.Since(startTime)
		fmt.Printf("   ⏱️ Plex API call completed in %v\n", duration)
	}

	return nil
}

// removeMediaFieldKeywords is a generic function to remove keywords from media fields (movies: type=1, TV shows: type=2)
func (c *Client) removeMediaFieldKeywords(mediaID, libraryID string, valuesToRemove []string, updateField string, lockField bool, mediaType int) error {
	// Build the base URL
	baseURL := c.buildURL(fmt.Sprintf("/library/sections/%s/all", libraryID))

	// Parse the URL to add query parameters properly
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return fmt.Errorf("failed to parse URL: %w", err)
	}

	// Create query parameters
	params := parsedURL.Query()
	params.Set("type", fmt.Sprintf("%d", mediaType))
	params.Set("id", mediaID)
	params.Set("includeExternalMedia", "1")

	// Join values with commas for the -= operator
	combinedValues := strings.Join(valuesToRemove, ",")

	// Add removal parameter using the -= operator
	paramName := fmt.Sprintf("%s[].tag.tag-", updateField)
	params.Set(paramName, combinedValues)

	if lockField {
		params.Set(fmt.Sprintf("%s.locked", updateField), "1")
	} else {
		params.Set(fmt.Sprintf("%s.locked", updateField), "0")
	}
	// Add the Plex token
	params.Set("X-Plex-Token", c.config.PlexToken)

	// Set the query parameters back to the URL
	parsedURL.RawQuery = params.Encode()

	req, err := http.NewRequest("PUT", parsedURL.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to remove media field keywords: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("plex API returned status %d when removing media field keywords - Response: %s", resp.StatusCode, string(body))
	}

	return nil
}

// getMediaTypeForLibraryType converts library type strings to Plex API media type integers
func (c *Client) getMediaTypeForLibraryType(libraryType string) int {
	switch libraryType {
	case "movie":
		return 1
	case "show":
		return 2
	default:
		// Default to 1 for unknown types (could log a warning here)
		return 1
	}
}

// buildURL constructs a full URL for Plex API requests
func (c *Client) buildURL(path string) string {
	return fmt.Sprintf("%s://%s:%s%s", c.config.Protocol, c.config.PlexServer, c.config.PlexPort, path)
}
