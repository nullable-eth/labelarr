package radarr

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

type Client struct {
	baseURL     string
	apiKey      string
	httpClient  *http.Client
	retryClient *utils.RetryableHTTPClient
	movies      []Movie
	moviesMu    sync.Mutex
}

func NewClient(baseURL, apiKey string) *Client {
	return NewClientWithRetryConfig(baseURL, apiKey, nil)
}

// NewClientWithRetryConfig creates a new Radarr API client with custom retry configuration.
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

// makeRequest performs an API request to Radarr with exponential backoff retry.
// Query parameters are URL-encoded via url.Values.Encode() so values containing
// spaces, "+", "&", etc. are escaped correctly.
func (c *Client) makeRequest(method, endpoint string, params url.Values) (*http.Response, error) {
	fullURL := fmt.Sprintf("%s%s", c.baseURL, endpoint)

	if len(params) > 0 {
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
		return nil, fmt.Errorf("radarr API returned status %d", resp.StatusCode)
	}

	return resp, nil
}

// GetAllMovies fetches the full movie list from Radarr, caching the result
// for the lifetime of the client. Call ClearCache to refresh.
func (c *Client) GetAllMovies() ([]Movie, error) {
	c.moviesMu.Lock()
	defer c.moviesMu.Unlock()

	if c.movies != nil {
		return c.movies, nil
	}

	resp, err := c.makeRequest("GET", "/api/v3/movie", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var movies []Movie
	if err := json.NewDecoder(resp.Body).Decode(&movies); err != nil {
		return nil, fmt.Errorf("error decoding movies: %w", err)
	}

	c.movies = movies
	return movies, nil
}

// ClearCache forces the next GetAllMovies call to re-fetch from Radarr.
func (c *Client) ClearCache() {
	c.moviesMu.Lock()
	c.movies = nil
	c.moviesMu.Unlock()
}

// SearchMovieByTitle returns all movies whose title, original title, clean title,
// or alternate titles match the query. Matching is bidirectional (either contains
// the other) and also checks a cleaned/normalized form for punctuation-insensitive matching.
func (c *Client) SearchMovieByTitle(title string) ([]Movie, error) {
	allMovies, err := c.GetAllMovies()
	if err != nil {
		return nil, err
	}

	titleLower := strings.ToLower(title)
	titleClean := cleanTitle(title)

	var matches []Movie
	for _, movie := range allMovies {
		if titleMatches(titleLower, titleClean, movie) {
			matches = append(matches, movie)
		}
	}

	return matches, nil
}

func titleMatches(titleLower, titleClean string, movie Movie) bool {
	movieTitleLower := strings.ToLower(movie.Title)
	movieOrigLower := strings.ToLower(movie.OriginalTitle)

	// Bidirectional substring match on title and original title
	if containsEither(movieTitleLower, titleLower) || containsEither(movieOrigLower, titleLower) {
		return true
	}

	// CleanTitle match (Radarr provides this stripped of punctuation)
	if movie.CleanTitle != "" && (movie.CleanTitle == titleClean || strings.Contains(movie.CleanTitle, titleClean) || strings.Contains(titleClean, movie.CleanTitle)) {
		return true
	}

	// Cleaned form match against title (for cases where CleanTitle field is empty)
	if cleanTitle(movie.Title) == titleClean {
		return true
	}

	// Alternate titles
	for _, alt := range movie.AlternateTitles {
		altLower := strings.ToLower(alt.Title)
		if containsEither(altLower, titleLower) {
			return true
		}
		if alt.CleanTitle != "" && (alt.CleanTitle == titleClean || strings.Contains(alt.CleanTitle, titleClean) || strings.Contains(titleClean, alt.CleanTitle)) {
			return true
		}
	}

	return false
}

func containsEither(a, b string) bool {
	return strings.Contains(a, b) || strings.Contains(b, a)
}

// FindMovieMatch finds the best match for a movie by title and year.
func (c *Client) FindMovieMatch(title string, year int) (*Movie, error) {
	movies, err := c.SearchMovieByTitle(title)
	if err != nil {
		return nil, err
	}

	titleLower := strings.ToLower(title)
	titleClean := cleanTitle(title)

	// Exact title + year
	for i := range movies {
		if strings.ToLower(movies[i].Title) == titleLower && movies[i].Year == year {
			return &movies[i], nil
		}
	}

	// CleanTitle + year
	for i := range movies {
		if movies[i].Year == year && (movies[i].CleanTitle == titleClean || cleanTitle(movies[i].Title) == titleClean) {
			return &movies[i], nil
		}
	}

	// Year match with any title hit
	for i := range movies {
		if movies[i].Year == year {
			return &movies[i], nil
		}
	}

	// Within 1 year
	for i := range movies {
		if movies[i].Year >= year-1 && movies[i].Year <= year+1 {
			return &movies[i], nil
		}
	}

	if len(movies) > 0 {
		return &movies[0], nil
	}

	return nil, fmt.Errorf("no movie match found for: %s (%d)", title, year)
}

func (c *Client) GetMovieByIMDbID(imdbID string) (*Movie, error) {
	if !strings.HasPrefix(imdbID, "tt") {
		imdbID = "tt" + imdbID
	}

	movies, err := c.GetAllMovies()
	if err != nil {
		return nil, err
	}

	for i := range movies {
		if movies[i].IMDbID == imdbID {
			return &movies[i], nil
		}
	}

	return nil, fmt.Errorf("movie with IMDb ID %s not found", imdbID)
}

func (c *Client) GetMovieByPath(filePath string) (*Movie, error) {
	movies, err := c.GetAllMovies()
	if err != nil {
		return nil, err
	}

	filePathLower := strings.ToLower(filePath)

	for i := range movies {
		if movies[i].Path != "" && strings.Contains(filePathLower, strings.ToLower(movies[i].Path)) {
			return &movies[i], nil
		}
		if movies[i].HasFile && movies[i].MovieFile.Path != "" {
			if strings.EqualFold(movies[i].MovieFile.Path, filePath) ||
				strings.Contains(filePathLower, strings.ToLower(movies[i].MovieFile.Path)) {
				return &movies[i], nil
			}
		}
	}

	return nil, fmt.Errorf("movie not found for path: %s", filePath)
}

func (c *Client) GetTMDbIDFromMovie(movie *Movie) string {
	if movie.TMDbID > 0 {
		return strconv.Itoa(movie.TMDbID)
	}
	return ""
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
