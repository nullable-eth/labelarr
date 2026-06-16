package sonarr

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nullable-eth/labelarr/internal/utils"
)

var nonAlphanumeric = regexp.MustCompile(`[^a-z0-9]`)

func cleanTitle(s string) string {
	return nonAlphanumeric.ReplaceAllString(strings.ToLower(s), "")
}

func containsEither(a, b string) bool {
	return strings.Contains(a, b) || strings.Contains(b, a)
}

type Client struct {
	baseURL     string
	apiKey      string
	httpClient  *http.Client
	retryClient *utils.RetryableHTTPClient
	series      []Series
	seriesMu    sync.Mutex
}

func NewClient(baseURL, apiKey string) *Client {
	return NewClientWithRetryConfig(baseURL, apiKey, nil)
}

// NewClientWithRetryConfig creates a new Sonarr API client with custom retry configuration.
// Pass nil for retryConfig to use the default exponential backoff configuration.
func NewClientWithRetryConfig(baseURL, apiKey string, retryConfig *utils.RetryConfig) *Client {
	baseURL = strings.TrimRight(baseURL, "/")
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}
	return &Client{
		baseURL:     baseURL,
		apiKey:      apiKey,
		httpClient:  httpClient,
		retryClient: utils.NewRetryableHTTPClient(httpClient, retryConfig),
	}
}

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

	resp, err := c.retryClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("sonarr API returned status %d", resp.StatusCode)
	}

	return resp, nil
}

// GetAllSeries fetches the full series list from Sonarr, caching the result
// for the lifetime of the client. Call ClearCache to refresh.
func (c *Client) GetAllSeries() ([]Series, error) {
	c.seriesMu.Lock()
	defer c.seriesMu.Unlock()

	if c.series != nil {
		return c.series, nil
	}

	resp, err := c.makeRequest("GET", "/api/v3/series", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var series []Series
	if err := json.NewDecoder(resp.Body).Decode(&series); err != nil {
		return nil, fmt.Errorf("error decoding series: %w", err)
	}

	c.series = series
	return series, nil
}

// ClearCache forces the next GetAllSeries call to re-fetch from Sonarr.
func (c *Client) ClearCache() {
	c.seriesMu.Lock()
	c.series = nil
	c.seriesMu.Unlock()
}

// SearchSeriesByTitle returns all series whose title, sort title, clean title,
// or alternate titles match the query. Matching is bidirectional and also checks
// a cleaned/normalized form for punctuation-insensitive matching.
func (c *Client) SearchSeriesByTitle(title string) ([]Series, error) {
	allSeries, err := c.GetAllSeries()
	if err != nil {
		return nil, err
	}

	titleLower := strings.ToLower(title)
	titleClean := cleanTitle(title)

	var matches []Series
	for _, s := range allSeries {
		if seriesTitleMatches(titleLower, titleClean, s) {
			matches = append(matches, s)
		}
	}

	return matches, nil
}

func seriesTitleMatches(titleLower, titleClean string, s Series) bool {
	sTitleLower := strings.ToLower(s.Title)
	sSortLower := strings.ToLower(s.SortTitle)

	if containsEither(sTitleLower, titleLower) || containsEither(sSortLower, titleLower) {
		return true
	}

	if s.CleanTitle != "" && (s.CleanTitle == titleClean || strings.Contains(s.CleanTitle, titleClean) || strings.Contains(titleClean, s.CleanTitle)) {
		return true
	}

	if cleanTitle(s.Title) == titleClean {
		return true
	}

	for _, alt := range s.AlternateTitles {
		altLower := strings.ToLower(alt.Title)
		if containsEither(altLower, titleLower) {
			return true
		}
	}

	return false
}

// FindSeriesMatch finds the best match for a series by title and year.
func (c *Client) FindSeriesMatch(title string, year int) (*Series, error) {
	series, err := c.SearchSeriesByTitle(title)
	if err != nil {
		return nil, err
	}

	titleLower := strings.ToLower(title)
	titleClean := cleanTitle(title)

	// Exact title + year
	for i := range series {
		if strings.ToLower(series[i].Title) == titleLower && series[i].Year == year {
			return &series[i], nil
		}
	}

	// CleanTitle + year
	for i := range series {
		if series[i].Year == year && (series[i].CleanTitle == titleClean || cleanTitle(series[i].Title) == titleClean) {
			return &series[i], nil
		}
	}

	// Year match with any title hit
	for i := range series {
		if series[i].Year == year {
			return &series[i], nil
		}
	}

	// Within 1 year
	for i := range series {
		if series[i].Year >= year-1 && series[i].Year <= year+1 {
			return &series[i], nil
		}
	}

	if len(series) > 0 {
		return &series[0], nil
	}

	return nil, fmt.Errorf("no series match found for: %s (%d)", title, year)
}

func (c *Client) GetSeriesByTMDbID(tmdbID int) (*Series, error) {
	series, err := c.GetAllSeries()
	if err != nil {
		return nil, err
	}

	for i := range series {
		if series[i].TMDBID == tmdbID {
			return &series[i], nil
		}
	}

	return nil, fmt.Errorf("series with TMDb ID %d not found", tmdbID)
}

func (c *Client) GetSeriesByTVDbID(tvdbID int) (*Series, error) {
	series, err := c.GetAllSeries()
	if err != nil {
		return nil, err
	}

	for i := range series {
		if series[i].TVDbID == tvdbID {
			return &series[i], nil
		}
	}

	return nil, fmt.Errorf("series with TVDb ID %d not found", tvdbID)
}

func (c *Client) GetSeriesByIMDbID(imdbID string) (*Series, error) {
	if !strings.HasPrefix(imdbID, "tt") {
		imdbID = "tt" + imdbID
	}

	series, err := c.GetAllSeries()
	if err != nil {
		return nil, err
	}

	for i := range series {
		if series[i].IMDBID == imdbID {
			return &series[i], nil
		}
	}

	return nil, fmt.Errorf("series with IMDb ID %s not found", imdbID)
}

func (c *Client) GetSeriesByPath(filePath string) (*Series, error) {
	series, err := c.GetAllSeries()
	if err != nil {
		return nil, err
	}

	filePathLower := strings.ToLower(filePath)

	for i := range series {
		if series[i].Path != "" && strings.Contains(filePathLower, strings.ToLower(series[i].Path)) {
			return &series[i], nil
		}
	}

	return nil, fmt.Errorf("series not found for path: %s", filePath)
}

func (c *Client) GetTMDbIDFromSeries(series *Series) string {
	if series.TMDBID > 0 {
		return strconv.Itoa(series.TMDBID)
	}
	return ""
}

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

func (c *Client) TestConnection() error {
	_, err := c.GetSystemStatus()
	return err
}
