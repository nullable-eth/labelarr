package sonarr

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Client represents a Sonarr API client
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a new Sonarr API client
func NewClient(baseURL, apiKey string) *Client {
	// Ensure baseURL doesn't have trailing slash
	baseURL = strings.TrimRight(baseURL, "/")
	
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// makeRequest performs an API request to Sonarr
func (c *Client) makeRequest(method, endpoint string, params url.Values) (*http.Response, error) {
	fullURL := fmt.Sprintf("%s%s", c.baseURL, endpoint)
	
	if params != nil && len(params) > 0 {
		fullURL = fmt.Sprintf("%s?%s", fullURL, params.Encode())
	}
	
	req, err := http.NewRequest(method, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	
	req.Header.Set("X-Api-Key", c.apiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("sonarr API returned status %d", resp.StatusCode)
	}
	
	return resp, nil
}

// GetAllSeries retrieves all TV series from Sonarr
func (c *Client) GetAllSeries() ([]Series, error) {
	resp, err := c.makeRequest("GET", "/api/v3/series", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	var series []Series
	if err := json.NewDecoder(resp.Body).Decode(&series); err != nil {
		return nil, fmt.Errorf("error decoding series: %w", err)
	}
	
	return series, nil
}

// GetSeriesByTMDbID retrieves a series by its TMDb ID
func (c *Client) GetSeriesByTMDbID(tmdbID int) (*Series, error) {
	series, err := c.GetAllSeries()
	if err != nil {
		return nil, err
	}
	
	for _, s := range series {
		if s.TMDBID == tmdbID {
			return &s, nil
		}
	}
	
	return nil, fmt.Errorf("series with TMDb ID %d not found", tmdbID)
}

// GetSeriesByTVDbID retrieves a series by its TVDb ID
func (c *Client) GetSeriesByTVDbID(tvdbID int) (*Series, error) {
	series, err := c.GetAllSeries()
	if err != nil {
		return nil, err
	}
	
	for _, s := range series {
		if s.TVDbID == tvdbID {
			return &s, nil
		}
	}
	
	return nil, fmt.Errorf("series with TVDb ID %d not found", tvdbID)
}

// SearchSeriesByTitle searches for series by title
func (c *Client) SearchSeriesByTitle(title string) ([]Series, error) {
	// First try to get all series and filter locally
	// This is more reliable than using Sonarr's search endpoint
	allSeries, err := c.GetAllSeries()
	if err != nil {
		return nil, err
	}
	
	var matches []Series
	titleLower := strings.ToLower(title)
	
	for _, series := range allSeries {
		if strings.Contains(strings.ToLower(series.Title), titleLower) ||
			strings.Contains(strings.ToLower(series.SortTitle), titleLower) {
			matches = append(matches, series)
			continue
		}
		
		// Check alternate titles
		for _, altTitle := range series.AlternateTitles {
			if strings.Contains(strings.ToLower(altTitle.Title), titleLower) {
				matches = append(matches, series)
				break
			}
		}
	}
	
	return matches, nil
}

// FindSeriesMatch attempts to find the best match for a series by title and year
func (c *Client) FindSeriesMatch(title string, year int) (*Series, error) {
	series, err := c.SearchSeriesByTitle(title)
	if err != nil {
		return nil, err
	}
	
	// First try exact title and year match
	titleLower := strings.ToLower(title)
	for _, s := range series {
		if strings.ToLower(s.Title) == titleLower && s.Year == year {
			return &s, nil
		}
	}
	
	// Then try year match with similar title
	for _, s := range series {
		if s.Year == year {
			return &s, nil
		}
	}
	
	// If still no match, try within 1 year range
	for _, s := range series {
		if s.Year >= year-1 && s.Year <= year+1 {
			return &s, nil
		}
	}
	
	// Return first match if any found
	if len(series) > 0 {
		return &series[0], nil
	}
	
	return nil, fmt.Errorf("no series match found for: %s (%d)", title, year)
}

// GetSystemStatus retrieves Sonarr system status (useful for testing connection)
func (c *Client) GetSystemStatus() (*SystemStatus, error) {
	resp, err := c.makeRequest("GET", "/api/v3/system/status", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	var status SystemStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("error decoding system status: %w", err)
	}
	
	return &status, nil
}

// TestConnection tests the connection to Sonarr
func (c *Client) TestConnection() error {
	_, err := c.GetSystemStatus()
	return err
}

// GetSeriesByIMDbID retrieves a series by its IMDb ID
func (c *Client) GetSeriesByIMDbID(imdbID string) (*Series, error) {
	// Normalize IMDb ID format
	if !strings.HasPrefix(imdbID, "tt") {
		imdbID = "tt" + imdbID
	}
	
	series, err := c.GetAllSeries()
	if err != nil {
		return nil, err
	}
	
	for _, s := range series {
		if s.IMDBID == imdbID {
			return &s, nil
		}
	}
	
	return nil, fmt.Errorf("series with IMDb ID %s not found", imdbID)
}

// GetSeriesByPath attempts to find a series by its file path
func (c *Client) GetSeriesByPath(filePath string) (*Series, error) {
	series, err := c.GetAllSeries()
	if err != nil {
		return nil, err
	}
	
	// Normalize the file path for comparison
	filePathLower := strings.ToLower(filePath)
	
	for _, s := range series {
		// Check if the file path is within the series' folder
		if s.Path != "" && strings.Contains(filePathLower, strings.ToLower(s.Path)) {
			return &s, nil
		}
	}
	
	return nil, fmt.Errorf("series not found for path: %s", filePath)
}

// GetTMDbIDFromSeries extracts the TMDb ID from a Sonarr series
func (c *Client) GetTMDbIDFromSeries(series *Series) string {
	if series.TMDBID > 0 {
		return strconv.Itoa(series.TMDBID)
	}
	return ""
}

// GetEpisodesBySeries gets all episodes for a series
func (c *Client) GetEpisodesBySeries(seriesID int) ([]Episode, error) {
	params := url.Values{}
	params.Set("seriesId", strconv.Itoa(seriesID))
	
	resp, err := c.makeRequest("GET", "/api/v3/episode", params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	var episodes []Episode
	if err := json.NewDecoder(resp.Body).Decode(&episodes); err != nil {
		return nil, fmt.Errorf("error decoding episodes: %w", err)
	}
	
	return episodes, nil
}