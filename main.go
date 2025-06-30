package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

// Configuration struct
type Config struct {
	PlexServer        string
	PlexPort          string
	PlexToken         string
	LibraryID         string
	TMDbReadAccessToken string
	ProcessTimer      time.Duration
	PlexRequiresHTTPS bool
}

// Plex API response structures
type MediaContainer struct {
	Size     int     `json:"size"`
	Metadata []Movie `json:"Metadata"`
}

type Movie struct {
	RatingKey string       `json:"ratingKey"`
	Title     string       `json:"title"`
	Year      int          `json:"year"`
	Label     []Label      `json:"Label,omitempty"`
	Guid      FlexibleGuid `json:"Guid,omitempty"`
	Media     []Media      `json:"Media,omitempty"`
}

type Label struct {
	Tag string `json:"tag"`
}

type Guid struct {
	ID string `json:"id"`
}

type Media struct {
	Part []Part `json:"Part,omitempty"`
}

type Part struct {
	File string `json:"file,omitempty"`
}

// FlexibleGuid handles both string and array formats from Plex API
type FlexibleGuid []Guid

func (fg *FlexibleGuid) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as array first
	var guidArray []Guid
	if err := json.Unmarshal(data, &guidArray); err == nil {
		*fg = FlexibleGuid(guidArray)
		return nil
	}

	// If that fails, try as single string
	var guidString string
	if err := json.Unmarshal(data, &guidString); err == nil {
		*fg = FlexibleGuid([]Guid{{ID: guidString}})
		return nil
	}

	// If both fail, try as single Guid object
	var singleGuid Guid
	if err := json.Unmarshal(data, &singleGuid); err == nil {
		*fg = FlexibleGuid([]Guid{singleGuid})
		return nil
	}

	return fmt.Errorf("cannot unmarshal Guid field")
}

type PlexResponse struct {
	MediaContainer MediaContainer `json:"MediaContainer"`
}

// Library structures for getting all libraries
type LibraryContainer struct {
	Size      int       `json:"size"`
	Directory []Library `json:"Directory"`
}

type Library struct {
	Key   string `json:"key"`
	Type  string `json:"type"`
	Title string `json:"title"`
	Agent string `json:"agent"`
}

type LibraryResponse struct {
	MediaContainer LibraryContainer `json:"MediaContainer"`
}

// TMDb API structures
type TMDbMovie struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	Overview string `json:"overview"`
}

type TMDbKeyword struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type TMDbKeywordsResponse struct {
	ID       int           `json:"id"`
	Keywords []TMDbKeyword `json:"keywords"`
}

// Processing state
type ProcessedMovie struct {
	RatingKey      string
	Title          string
	TMDbID         string
	LastProcessed  time.Time
	KeywordsSynced bool
}

var processedMovies = make(map[string]*ProcessedMovie)
var totalMovieCount int

func main() {
	// Configuration from environment variables
	config := Config{
		PlexServer:        os.Getenv("PLEX_SERVER"),
		PlexPort:          os.Getenv("PLEX_PORT"),
		PlexToken:         os.Getenv("PLEX_TOKEN"),
		LibraryID:         os.Getenv("LIBRARY_ID"), // Will be auto-detected
		TMDbReadAccessToken: os.Getenv("TMDB_READ_ACCESS_TOKEN"),
		ProcessTimer:      getProcessTimerFromEnv(),
		PlexRequiresHTTPS: getBoolEnvWithDefault("PLEX_REQUIRES_HTTPS", true),
	}

	processAllMovieLibraries := getBoolEnvWithDefault("PROCESS_ALL_MOVIE_LIBRARIES", false)

	if config.PlexToken == "" {
		fmt.Println("❌ Please set PLEX_TOKEN environment variable")
		os.Exit(1)
	}

	if config.TMDbReadAccessToken == "" {
		fmt.Println("❌ Please set TMDB_READ_ACCESS_TOKEN environment variable")
		os.Exit(1)
	}

	protocol := "https"
	if !config.PlexRequiresHTTPS {
		protocol = "http"
	}

	fmt.Println("🏷️ Starting Labelarr with TMDb Integration...")
	fmt.Printf("📡 Server: %s://%s:%s\n", protocol, config.PlexServer, config.PlexPort)
	fmt.Printf("⏱️ Processing interval: %v\n", config.ProcessTimer)

	// Step 1: Get all libraries first
	fmt.Println("\n📚 Step 1: Fetching all libraries...")
	libraries, err := getAllLibraries(config)
	if err != nil {
		fmt.Printf("❌ Error fetching libraries: %v\n", err)
		os.Exit(1)
	}

	if len(libraries) == 0 {
		fmt.Println("❌ No libraries found!")
		os.Exit(1)
	}

	fmt.Printf("✅ Found %d libraries:\n", len(libraries))
	for _, lib := range libraries {
		fmt.Printf("  📁 ID: %s - %s (%s)\n", lib.Key, lib.Title, lib.Type)
	}

	var movieLibraries []Library
	for _, lib := range libraries {
		if lib.Type == "movie" {
			movieLibraries = append(movieLibraries, lib)
		}
	}

	if len(movieLibraries) == 0 {
		fmt.Println("❌ No movie library found!")
		os.Exit(1)
	}

	if processAllMovieLibraries {
		fmt.Printf("\n🎯 Processing all %d movie libraries\n", len(movieLibraries))
	} else {
		fmt.Printf("\n🎯 Using Movies library: %s (ID: %s)\n", movieLibraries[0].Title, movieLibraries[0].Key)
	}

	// Start the periodic processing
	fmt.Println("\n🔄 Starting periodic movie processing...")

	processFunc := func() {
		if processAllMovieLibraries {
			for _, lib := range movieLibraries {
				fmt.Printf("\n==============================\n")
				fmt.Printf("🎬 Processing library: %s (ID: %s)\n", lib.Title, lib.Key)
				libConfig := config
				libConfig.LibraryID = lib.Key
				processAllMovies(libConfig)
			}
		} else {
			config.LibraryID = movieLibraries[0].Key
			processAllMovies(config)
		}
	}

	// Process immediately on start
	processFunc()

	// Set up timer for periodic processing
	ticker := time.NewTicker(config.ProcessTimer)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			fmt.Printf("\n⏰ Timer triggered - processing movies at %s\n", time.Now().Format("15:04:05"))
			processFunc()
		}
	}
}

func processAllMovies(config Config) {
	fmt.Println("\n📋 Fetching all movies from library...")
	movies, err := getMoviesFromLibrary(config)
	if err != nil {
		fmt.Printf("❌ Error fetching movies: %v\n", err)
		return
	}

	if len(movies) == 0 {
		fmt.Println("❌ No movies found in library!")
		return
	}

	totalMovieCount = len(movies)
	fmt.Printf("✅ Found %d movies in library\n", totalMovieCount)

	newMovies := 0
	updatedMovies := 0
	skippedMovies := 0

	for i, movie := range movies {
		// Check if movie was already processed
		processed, exists := processedMovies[movie.RatingKey]
		if exists && processed.KeywordsSynced {
			skippedMovies++
			continue
		}

		// Extract TMDb ID from movie first (before any output)
		tmdbID := extractTMDbID(movie)
		if tmdbID == "" {
			fmt.Printf("⚠️ No TMDb ID found for movie: %s\n", movie.Title)
			continue
		}

		// Get TMDb keywords
		keywords, err := getTMDbKeywords(config, tmdbID)
		if err != nil {
			fmt.Printf("❌ Error fetching TMDb keywords for %s: %v\n", movie.Title, err)
			continue
		}

		// Fetch detailed movie information to get current labels
		movieDetails, err := getMovieDetails(config, movie.RatingKey)
		if err != nil {
			fmt.Printf("❌ Error fetching movie details for %s: %v\n", movie.Title, err)
			continue
		}

		// Get current movie labels from detailed fetch
		currentLabels := make([]string, len(movieDetails.Label))
		for j, label := range movieDetails.Label {
			currentLabels[j] = label.Tag
		}

		// Check if all keywords already exist as labels (case-insensitive)
		currentLabelsMap := make(map[string]bool)
		for _, label := range currentLabels {
			currentLabelsMap[strings.ToLower(label)] = true
		}

		allKeywordsExist := true
		for _, keyword := range keywords {
			if !currentLabelsMap[strings.ToLower(keyword)] {
				allKeywordsExist = false
				break
			}
		}

		// If all keywords already exist, skip silently
		if allKeywordsExist {
			skippedMovies++
			continue
		}

		// Only show processing output for movies that need updates
		fmt.Printf("\n🎬 Processing movie %d/%d: %s (%d)\n", i+1, len(movies), movie.Title, movie.Year)
		fmt.Printf("🔑 TMDb ID: %s (%s)\n", tmdbID, movie.Title)
		fmt.Printf("🏷️ Found %d TMDb keywords\n", len(keywords))

		// Sync labels with keywords
		err = syncMovieLabelsWithKeywords(config, movie.RatingKey, currentLabels, keywords)
		if err != nil {
			fmt.Printf("❌ Error syncing labels: %v\n", err)
			continue
		}

		// Update processed movies dictionary
		processedMovies[movie.RatingKey] = &ProcessedMovie{
			RatingKey:      movie.RatingKey,
			Title:          movie.Title,
			TMDbID:         tmdbID,
			LastProcessed:  time.Now(),
			KeywordsSynced: true,
		}

		if exists {
			updatedMovies++
		} else {
			newMovies++
		}

		fmt.Printf("✅ Successfully processed: %s\n", movie.Title)

		// Small delay to avoid overwhelming the APIs
		time.Sleep(500 * time.Millisecond)
	}

	fmt.Printf("\n📊 Processing Summary:\n")
	fmt.Printf("  📈 Total movies in library: %d\n", totalMovieCount)
	fmt.Printf("  🆕 New movies processed: %d\n", newMovies)
	fmt.Printf("  🔄 Updated movies: %d\n", updatedMovies)
	fmt.Printf("  ⏭️ Skipped movies: %d\n", skippedMovies)
	fmt.Printf("  📋 Total processed movies: %d\n", len(processedMovies))
}

func extractTMDbID(movie Movie) string {
	// First, look for TMDb ID in Guid array
	for _, guid := range movie.Guid {
		// TMDb IDs typically come in format like "tmdb://12345"
		if strings.Contains(guid.ID, "tmdb://") {
			return strings.TrimPrefix(guid.ID, "tmdb://")
		}
		// Sometimes it might be in format "com.plexapp.agents.themoviedb://12345"
		if strings.Contains(guid.ID, "themoviedb://") {
			return strings.TrimSuffix(strings.TrimPrefix(guid.ID, "com.plexapp.agents.themoviedb://"), "?lang=en")
		}
	}

	// If not found in Guid, try to extract from other patterns in Guid
	tmdbRegex := regexp.MustCompile(`tmdb-(\d+)`)
	for _, guid := range movie.Guid {
		if matches := tmdbRegex.FindStringSubmatch(guid.ID); len(matches) > 1 {
			return matches[1]
		}
	}

	// If still not found, try to extract from file paths
	// Look for patterns like {tmdb-12345} or [tmdb:12345] or (tmdb;12345) etc.
	// This regex will match:
	// 1. Any opening brace/bracket/parenthesis
	// 2. Optional whitespace
	// 3. "tmdb" (case insensitive)
	// 4. Any non-digit characters (separators)
	// 5. One or more digits (the ID)
	// 6. Any closing brace/bracket/parenthesis
	filePathRegex := regexp.MustCompile(`[\[\{\(\<]?\s*tmdb\D+?(\d+)[\]\}\)\>]?`)

	for _, media := range movie.Media {
		for _, part := range media.Part {
			if part.File != "" {
				// Convert backslashes to forward slashes for consistency
				normalizedPath := strings.ReplaceAll(part.File, "\\", "/")

				// Check both the full path and individual path components
				if matches := filePathRegex.FindStringSubmatch(normalizedPath); len(matches) > 1 {
					return matches[1]
				}

				// Split path and check each component
				pathComponents := strings.Split(normalizedPath, "/")
				for _, component := range pathComponents {
					if matches := filePathRegex.FindStringSubmatch(component); len(matches) > 1 {
						return matches[1]
					}
				}
			}
		}
	}

	return ""
}

func getTMDbKeywords(config Config, tmdbID string) ([]string, error) {
	keywordsURL := fmt.Sprintf("https://api.themoviedb.org/3/movie/%s/keywords", tmdbID)

	req, err := http.NewRequest("GET", keywordsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating TMDb request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.TMDbReadAccessToken))

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making TMDb request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("TMDb API HTTP %d: %s - Response: %s", resp.StatusCode, resp.Status, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading TMDb response: %w", err)
	}

	var keywordsResponse TMDbKeywordsResponse
	if err := json.Unmarshal(body, &keywordsResponse); err != nil {
		return nil, fmt.Errorf("parsing TMDb JSON: %w", err)
	}

	keywords := make([]string, len(keywordsResponse.Keywords))
	for i, keyword := range keywordsResponse.Keywords {
		keywords[i] = keyword.Name
	}

	return keywords, nil
}

func syncMovieLabelsWithKeywords(config Config, movieID string, currentLabels []string, keywords []string) error {
	// Convert current labels to a map for easy lookup (case-insensitive)
	currentLabelsMap := make(map[string]bool)
	for _, label := range currentLabels {
		currentLabelsMap[strings.ToLower(label)] = true
	}

	// Find keywords to add (keywords not in current labels, case-insensitive comparison)
	labelsToAdd := make([]string, 0)
	for _, keyword := range keywords {
		if !currentLabelsMap[strings.ToLower(keyword)] {
			labelsToAdd = append(labelsToAdd, keyword)
		}
	}

	fmt.Printf("  📝 Labels to add: %v\n", labelsToAdd)
	fmt.Printf("  🏷️ Existing labels: %v\n", currentLabels)

	// Merge existing labels with new keywords
	allLabels := make([]string, 0, len(currentLabels)+len(labelsToAdd))
	allLabels = append(allLabels, currentLabels...)
	allLabels = append(allLabels, labelsToAdd...)

	// Update movie labels with combined list
	return updateMovieLabelsWithKeywords(config, movieID, allLabels)
}

func updateMovieLabelsWithKeywords(config Config, movieID string, keywords []string) error {
	// Build the base URL path
	basePath := fmt.Sprintf("/library/sections/%s/all?type=1&id=%s&includeExternalMedia=1", config.LibraryID, movieID)

	// Add each keyword as a label parameter
	for i, keyword := range keywords {
		encodedKeyword := url.QueryEscape(keyword)
		basePath += fmt.Sprintf("&label%%5B%d%%5D.tag.tag=%s", i, encodedKeyword)
	}

	// Add label lock and token
	basePath += fmt.Sprintf("&label.locked=1&X-Plex-Token=%s", config.PlexToken)

	// Build the complete URL
	updateURL := buildPlexURL(config, basePath)

	fmt.Printf("  📤 Updating movie labels...\n")

	req, err := http.NewRequest("PUT", updateURL, nil)
	if err != nil {
		return fmt.Errorf("creating update request: %w", err)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("making update request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s - Response: %s", resp.StatusCode, resp.Status, string(body))
	}

	return nil
}

func getAllLibraries(config Config) ([]Library, error) {
	librariesURL := buildPlexURL(config, fmt.Sprintf("/library/sections/?X-Plex-Token=%s", config.PlexToken))

	fmt.Printf("🔗 Attempting to connect to: %s\n", librariesURL)

	req, err := http.NewRequest("GET", librariesURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	client := &http.Client{
		Timeout: 30 * time.Second, // Add 30 second timeout
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	fmt.Println("📡 Making request to Plex server...")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making request to %s: %w", librariesURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s - Response: %s", resp.StatusCode, resp.Status, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	var libraryResponse LibraryResponse
	if err := json.Unmarshal(body, &libraryResponse); err != nil {
		return nil, fmt.Errorf("parsing JSON response: %w", err)
	}

	return libraryResponse.MediaContainer.Directory, nil
}

func getMovieDetails(config Config, ratingKey string) (*Movie, error) {
	// Use the individual metadata endpoint which includes labels by default
	movieURL := buildPlexURL(config, fmt.Sprintf("/library/metadata/%s?X-Plex-Token=%s", ratingKey, config.PlexToken))

	req, err := http.NewRequest("GET", movieURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making request to %s: %w", movieURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s - Response: %s", resp.StatusCode, resp.Status, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	var plexResponse PlexResponse
	if err := json.Unmarshal(body, &plexResponse); err != nil {
		return nil, fmt.Errorf("parsing JSON response: %w", err)
	}

	if len(plexResponse.MediaContainer.Metadata) == 0 {
		return nil, fmt.Errorf("no movie found with ratingKey %s", ratingKey)
	}

	return &plexResponse.MediaContainer.Metadata[0], nil
}

func getMoviesFromLibrary(config Config) ([]Movie, error) {
	moviesURL := buildPlexURL(config, fmt.Sprintf("/library/sections/%s/all?X-Plex-Token=%s", config.LibraryID, config.PlexToken))

	req, err := http.NewRequest("GET", moviesURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making request to %s: %w", moviesURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s - Response: %s", resp.StatusCode, resp.Status, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	var plexResponse PlexResponse
	if err := json.Unmarshal(body, &plexResponse); err != nil {
		return nil, fmt.Errorf("parsing JSON response: %w", err)
	}

	return plexResponse.MediaContainer.Metadata, nil
}

func getEnvWithDefault(envVar, defaultValue string) string {
	if value, exists := os.LookupEnv(envVar); exists {
		return value
	}
	return defaultValue
}

func getProcessTimerFromEnv() time.Duration {
	if value, exists := os.LookupEnv("PROCESS_TIMER"); exists {
		duration, err := time.ParseDuration(value)
		if err == nil {
			return duration
		}
	}
	return 5 * time.Minute
}

func getBoolEnvWithDefault(envVar string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(envVar); exists {
		return value == "true"
	}
	return defaultValue
}

func buildPlexURL(config Config, path string) string {
	protocol := "https"
	if !config.PlexRequiresHTTPS {
		protocol = "http"
	}
	return fmt.Sprintf("%s://%s:%s%s", protocol, config.PlexServer, config.PlexPort, path)
}
