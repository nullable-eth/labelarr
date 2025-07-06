package radarr

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Client represents a Radarr API client
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a new Radarr API client
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

// makeRequest performs an API request to Radarr
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
		return nil, fmt.Errorf("radarr API returned status %d", resp.StatusCode)
	}
	
	return resp, nil
}

// GetAllMovies retrieves all movies from Radarr
func (c *Client) GetAllMovies() ([]Movie, error) {
	resp, err := c.makeRequest("GET", "/api/v3/movie", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	var movies []Movie
	if err := json.NewDecoder(resp.Body).Decode(&movies); err != nil {
		return nil, fmt.Errorf("error decoding movies: %w", err)
	}
	
	return movies, nil
}

// GetMovieByTMDbID retrieves a movie by its TMDb ID
func (c *Client) GetMovieByTMDbID(tmdbID int) (*Movie, error) {
	movies, err := c.GetAllMovies()
	if err != nil {
		return nil, err
	}
	
	for _, movie := range movies {
		if movie.TMDbID == tmdbID {
			return &movie, nil
		}
	}
	
	return nil, fmt.Errorf("movie with TMDb ID %d not found", tmdbID)
}

// SearchMovieByTitle searches for movies by title
func (c *Client) SearchMovieByTitle(title string) ([]Movie, error) {
	// First try to get all movies and filter locally
	// This is more reliable than using Radarr's search endpoint
	allMovies, err := c.GetAllMovies()
	if err != nil {
		return nil, err
	}
	
	var matches []Movie
	titleLower := strings.ToLower(title)
	
	for _, movie := range allMovies {
		if strings.Contains(strings.ToLower(movie.Title), titleLower) ||
			strings.Contains(strings.ToLower(movie.OriginalTitle), titleLower) {
			matches = append(matches, movie)
			continue
		}
		
		// Check alternate titles
		for _, altTitle := range movie.AlternateTitles {
			if strings.Contains(strings.ToLower(altTitle.Title), titleLower) {
				matches = append(matches, movie)
				break
			}
		}
	}
	
	return matches, nil
}

// FindMovieMatch attempts to find the best match for a movie by title and year
func (c *Client) FindMovieMatch(title string, year int) (*Movie, error) {
	movies, err := c.SearchMovieByTitle(title)
	if err != nil {
		return nil, err
	}
	
	// First try exact title and year match
	titleLower := strings.ToLower(title)
	for _, movie := range movies {
		if strings.ToLower(movie.Title) == titleLower && movie.Year == year {
			return &movie, nil
		}
	}
	
	// Then try year match with similar title
	for _, movie := range movies {
		if movie.Year == year {
			return &movie, nil
		}
	}
	
	// If still no match, try within 1 year range
	for _, movie := range movies {
		if movie.Year >= year-1 && movie.Year <= year+1 {
			return &movie, nil
		}
	}
	
	// Return first match if any found
	if len(movies) > 0 {
		return &movies[0], nil
	}
	
	return nil, fmt.Errorf("no movie match found for: %s (%d)", title, year)
}

// GetSystemStatus retrieves Radarr system status (useful for testing connection)
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

// TestConnection tests the connection to Radarr
func (c *Client) TestConnection() error {
	_, err := c.GetSystemStatus()
	return err
}

// GetMovieByIMDbID retrieves a movie by its IMDb ID
func (c *Client) GetMovieByIMDbID(imdbID string) (*Movie, error) {
	// Normalize IMDb ID format
	if !strings.HasPrefix(imdbID, "tt") {
		imdbID = "tt" + imdbID
	}
	
	movies, err := c.GetAllMovies()
	if err != nil {
		return nil, err
	}
	
	for _, movie := range movies {
		if movie.IMDbID == imdbID {
			return &movie, nil
		}
	}
	
	return nil, fmt.Errorf("movie with IMDb ID %s not found", imdbID)
}

// GetMovieByPath attempts to find a movie by its file path
func (c *Client) GetMovieByPath(filePath string) (*Movie, error) {
	movies, err := c.GetAllMovies()
	if err != nil {
		return nil, err
	}
	
	// Normalize the file path for comparison
	filePathLower := strings.ToLower(filePath)
	
	for _, movie := range movies {
		// Check if the file path is within the movie's folder
		if movie.Path != "" && strings.Contains(filePathLower, strings.ToLower(movie.Path)) {
			return &movie, nil
		}
		
		// Also check against the movie file path if available
		if movie.HasFile && movie.MovieFile.Path != "" {
			if strings.EqualFold(movie.MovieFile.Path, filePath) ||
				strings.Contains(filePathLower, strings.ToLower(movie.MovieFile.Path)) {
				return &movie, nil
			}
		}
	}
	
	return nil, fmt.Errorf("movie not found for path: %s", filePath)
}

// GetTMDbIDFromMovie extracts the TMDb ID from a Radarr movie
func (c *Client) GetTMDbIDFromMovie(movie *Movie) string {
	if movie.TMDbID > 0 {
		return strconv.Itoa(movie.TMDbID)
	}
	return ""
}