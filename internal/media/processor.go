package media

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/nullable-eth/labelarr/internal/config"
	"github.com/nullable-eth/labelarr/internal/plex"
	"github.com/nullable-eth/labelarr/internal/tmdb"
)

// MediaType represents the type of media being processed
type MediaType string

const (
	MediaTypeMovie MediaType = "movie"
	MediaTypeTV    MediaType = "tv"
)

// ProcessedItem tracks processing state for any media item
type ProcessedItem struct {
	RatingKey      string
	Title          string
	TMDbID         string
	LastProcessed  time.Time
	KeywordsSynced bool
}

// MediaItem interface for common media operations
type MediaItem interface {
	GetRatingKey() string
	GetTitle() string
	GetYear() int
	GetGuid() []plex.Guid
	GetMedia() []plex.Media
	GetLabel() []plex.Label
	GetGenre() []plex.Genre
}

// Processor handles media processing operations for any media type
type Processor struct {
	config         *config.Config
	plexClient     *plex.Client
	tmdbClient     *tmdb.Client
	processedItems map[string]*ProcessedItem
}

// NewProcessor creates a new generic media processor
func NewProcessor(cfg *config.Config, plexClient *plex.Client, tmdbClient *tmdb.Client) *Processor {
	return &Processor{
		config:         cfg,
		plexClient:     plexClient,
		tmdbClient:     tmdbClient,
		processedItems: make(map[string]*ProcessedItem),
	}
}

// ProcessAllItems processes all items in the specified library
func (p *Processor) ProcessAllItems(libraryID string, mediaType MediaType) error {
	var displayName, emoji string
	switch mediaType {
	case MediaTypeMovie:
		displayName = "movies"
		emoji = "🎬"
	case MediaTypeTV:
		displayName = "tv shows"
		emoji = "📺"
	default:
		return fmt.Errorf("unsupported media type: %s", mediaType)
	}

	fmt.Printf("📋 Fetching all %s from library...\n", displayName)

	items, err := p.fetchItems(libraryID, mediaType)
	if err != nil {
		return fmt.Errorf("error fetching %s: %w", displayName, err)
	}

	if len(items) == 0 {
		fmt.Printf("❌ No %s found in library!\n", displayName)
		return nil
	}

	totalCount := len(items)
	fmt.Printf("✅ Found %d %s in library\n", totalCount, displayName)

	newItems := 0
	updatedItems := 0
	skippedItems := 0
	skippedAlreadyExist := 0

	for _, item := range items {
		processed, exists := p.processedItems[item.GetRatingKey()]
		if exists && processed.KeywordsSynced {
			skippedItems++
			skippedAlreadyExist++
			continue
		}

		// Silently check if we need to process this item
		tmdbID := p.extractTMDbID(item, mediaType)
		if tmdbID == "" {
			skippedItems++
			continue
		}

		// Silently fetch keywords and details to check if processing is needed
		keywords, err := p.getKeywords(tmdbID, mediaType)
		if err != nil {
			skippedItems++
			continue
		}

		details, err := p.getItemDetails(item.GetRatingKey(), mediaType)
		if err != nil {
			skippedItems++
			continue
		}

		currentValues := p.extractCurrentValues(details)

		currentValuesMap := make(map[string]bool)
		for _, val := range currentValues {
			currentValuesMap[strings.ToLower(val)] = true
		}

		allKeywordsExist := true
		for _, keyword := range keywords {
			if !currentValuesMap[strings.ToLower(keyword)] {
				allKeywordsExist = false
				break
			}
		}

		if allKeywordsExist {
			// Silently skip - no verbose output
			skippedItems++
			skippedAlreadyExist++
			continue
		}

		// Only show verbose output for completely new items (never processed before)
		if !exists {
			fmt.Printf("\n%s Processing new %s: %s (%d)\n", emoji, strings.TrimSuffix(displayName, "s"), item.GetTitle(), item.GetYear())
			fmt.Printf("🔑 TMDb ID: %s (%s)\n", tmdbID, item.GetTitle())
			fmt.Printf("🏷️ Found %d TMDb keywords\n", len(keywords))
		}

		err = p.syncFieldWithKeywords(item.GetRatingKey(), libraryID, currentValues, keywords, mediaType)
		if err != nil {
			// Show error even for existing items since it's important
			if exists {
				fmt.Printf("❌ Error syncing %s for %s: %v\n", p.config.UpdateField, item.GetTitle(), err)
			}
			skippedItems++
			continue
		}

		p.processedItems[item.GetRatingKey()] = &ProcessedItem{
			RatingKey:      item.GetRatingKey(),
			Title:          item.GetTitle(),
			TMDbID:         tmdbID,
			LastProcessed:  time.Now(),
			KeywordsSynced: true,
		}

		if exists {
			updatedItems++
		} else {
			newItems++
			fmt.Printf("✅ Successfully processed new %s: %s\n", strings.TrimSuffix(displayName, "s"), item.GetTitle())
		}

		time.Sleep(500 * time.Millisecond)
	}

	fmt.Printf("\n📊 Processing Summary:\n")
	fmt.Printf("  📈 Total %s in library: %d\n", displayName, totalCount)
	fmt.Printf("  🆕 New %s processed: %d\n", displayName, newItems)
	fmt.Printf("  🔄 Updated %s: %d\n", displayName, updatedItems)
	fmt.Printf("  ⏭️ Skipped %s: %d\n", displayName, skippedItems)
	if skippedAlreadyExist > 0 {
		fmt.Printf("  ✨ Already have all keywords: %d\n", skippedAlreadyExist)
	}

	return nil
}

// RemoveKeywordsFromItems removes TMDb keywords from all items in the specified library
func (p *Processor) RemoveKeywordsFromItems(libraryID string, mediaType MediaType) error {
	var displayName, emoji string
	switch mediaType {
	case MediaTypeMovie:
		displayName = "movies"
		emoji = "🎬"
	case MediaTypeTV:
		displayName = "tv shows"
		emoji = "📺"
	default:
		return fmt.Errorf("unsupported media type: %s", mediaType)
	}

	fmt.Printf("\n📋 Fetching all %s for keyword removal...\n", displayName)

	items, err := p.fetchItems(libraryID, mediaType)
	if err != nil {
		return fmt.Errorf("error fetching %s: %w", displayName, err)
	}

	if len(items) == 0 {
		fmt.Printf("❌ No %s found in library!\n", displayName)
		return nil
	}

	fmt.Printf("✅ Found %d %s in library\n", len(items), displayName)

	removedCount := 0
	skippedCount := 0
	totalKeywordsRemoved := 0

	for _, item := range items {
		tmdbID := p.extractTMDbID(item, mediaType)
		if tmdbID == "" {
			skippedCount++
			continue
		}

		details, err := p.getItemDetails(item.GetRatingKey(), mediaType)
		if err != nil {
			fmt.Printf("❌ Error fetching %s details for %s: %v\n", strings.TrimSuffix(displayName, "s"), item.GetTitle(), err)
			skippedCount++
			continue
		}

		currentValues := p.extractCurrentValues(details)

		if len(currentValues) == 0 {
			skippedCount++
			continue
		}

		keywords, err := p.getKeywords(tmdbID, mediaType)
		if err != nil {
			keywords = []string{}
		}

		keywordMap := make(map[string]bool)
		for _, keyword := range keywords {
			keywordMap[strings.ToLower(keyword)] = true
		}

		var valuesToRemove []string
		foundTMDbKeywords := false
		for _, value := range currentValues {
			if keywordMap[strings.ToLower(value)] {
				foundTMDbKeywords = true
				valuesToRemove = append(valuesToRemove, value)
			}
		}

		if !foundTMDbKeywords {
			skippedCount++
			continue
		}

		fmt.Printf("\n%s Processing %s: %s (%d)\n", emoji, strings.TrimSuffix(displayName, "s"), item.GetTitle(), item.GetYear())
		fmt.Printf("🔑 TMDb ID: %s\n", tmdbID)
		fmt.Printf("🗑️ Removing %d TMDb keywords from %s field\n", len(valuesToRemove), p.config.UpdateField)

		lockField := p.config.RemoveMode == "lock"
		err = p.removeItemFieldKeywords(item.GetRatingKey(), libraryID, valuesToRemove, lockField, mediaType)
		if err != nil {
			fmt.Printf("❌ Error removing keywords from %s: %v\n", item.GetTitle(), err)
			skippedCount++
			continue
		}

		totalKeywordsRemoved += len(valuesToRemove)
		removedCount++
		fmt.Printf("✅ Successfully removed keywords from %s\n", item.GetTitle())

		time.Sleep(500 * time.Millisecond)
	}

	fmt.Printf("\n📊 Removal Summary:\n")
	fmt.Printf("  📈 Total %s checked: %d\n", displayName, len(items))
	fmt.Printf("  🗑️ %s with keywords removed: %d\n", strings.Title(displayName), removedCount)
	fmt.Printf("  ⏭️ Skipped %s: %d\n", displayName, skippedCount)
	fmt.Printf("  🏷️ Total keywords removed: %d\n", totalKeywordsRemoved)

	return nil
}

// fetchItems gets all items from the library based on media type
func (p *Processor) fetchItems(libraryID string, mediaType MediaType) ([]MediaItem, error) {
	switch mediaType {
	case MediaTypeMovie:
		movies, err := p.plexClient.GetMoviesFromLibrary(libraryID)
		if err != nil {
			return nil, err
		}
		items := make([]MediaItem, len(movies))
		for i, movie := range movies {
			items[i] = movie
		}
		return items, nil

	case MediaTypeTV:
		tvShows, err := p.plexClient.GetTVShowsFromLibrary(libraryID)
		if err != nil {
			return nil, err
		}
		items := make([]MediaItem, len(tvShows))
		for i, tvShow := range tvShows {
			items[i] = tvShow
		}
		return items, nil

	default:
		return nil, fmt.Errorf("unsupported media type: %s", mediaType)
	}
}

// getItemDetails gets detailed information for an item based on media type
func (p *Processor) getItemDetails(ratingKey string, mediaType MediaType) (MediaItem, error) {
	switch mediaType {
	case MediaTypeMovie:
		movie, err := p.plexClient.GetMovieDetails(ratingKey)
		if err != nil {
			return nil, err
		}
		return *movie, nil

	case MediaTypeTV:
		tvShow, err := p.plexClient.GetTVShowDetails(ratingKey)
		if err != nil {
			return nil, err
		}
		return *tvShow, nil

	default:
		return nil, fmt.Errorf("unsupported media type: %s", mediaType)
	}
}

// getKeywords gets keywords from TMDb based on media type
func (p *Processor) getKeywords(tmdbID string, mediaType MediaType) ([]string, error) {
	switch mediaType {
	case MediaTypeMovie:
		return p.tmdbClient.GetMovieKeywords(tmdbID)
	case MediaTypeTV:
		return p.tmdbClient.GetTVShowKeywords(tmdbID)
	default:
		return nil, fmt.Errorf("unsupported media type: %s", mediaType)
	}
}

// syncFieldWithKeywords synchronizes the configured field with TMDb keywords
func (p *Processor) syncFieldWithKeywords(itemID, libraryID string, currentValues []string, keywords []string, mediaType MediaType) error {
	mergedValues := append(currentValues, keywords...)

	// Remove duplicates while preserving order
	seen := make(map[string]bool)
	var uniqueValues []string
	for _, value := range mergedValues {
		lowerValue := strings.ToLower(value)
		if !seen[lowerValue] {
			seen[lowerValue] = true
			uniqueValues = append(uniqueValues, value)
		}
	}

	return p.updateItemField(itemID, libraryID, uniqueValues, mediaType)
}

// toPlexMediaType converts MediaType to the string format expected by plex client
func (p *Processor) toPlexMediaType(mediaType MediaType) (string, error) {
	switch mediaType {
	case MediaTypeMovie:
		return "movie", nil
	case MediaTypeTV:
		return "show", nil
	default:
		return "", fmt.Errorf("unsupported media type: %s", mediaType)
	}
}

// updateItemField updates the configured field based on media type
func (p *Processor) updateItemField(itemID, libraryID string, keywords []string, mediaType MediaType) error {
	plexMediaType, err := p.toPlexMediaType(mediaType)
	if err != nil {
		return err
	}

	return p.plexClient.UpdateMediaField(itemID, libraryID, keywords, p.config.UpdateField, plexMediaType)
}

// removeItemFieldKeywords removes specific keywords from the configured field based on media type
func (p *Processor) removeItemFieldKeywords(itemID, libraryID string, valuesToRemove []string, lockField bool, mediaType MediaType) error {
	plexMediaType, err := p.toPlexMediaType(mediaType)
	if err != nil {
		return err
	}

	return p.plexClient.RemoveMediaFieldKeywords(itemID, libraryID, valuesToRemove, p.config.UpdateField, lockField, plexMediaType)
}

// extractCurrentValues extracts current values from the configured field
func (p *Processor) extractCurrentValues(item MediaItem) []string {
	switch strings.ToLower(p.config.UpdateField) {
	case "label":
		labels := item.GetLabel()
		values := make([]string, len(labels))
		for i, label := range labels {
			values[i] = label.Tag
		}
		return values
	case "genre":
		genres := item.GetGenre()
		values := make([]string, len(genres))
		for i, genre := range genres {
			values[i] = genre.Tag
		}
		return values
	default:
		return []string{}
	}
}

// extractTMDbID extracts TMDb ID using the appropriate strategy for each media type
func (p *Processor) extractTMDbID(item MediaItem, mediaType MediaType) string {
	switch mediaType {
	case MediaTypeMovie:
		return p.extractMovieTMDbID(item)
	case MediaTypeTV:
		return p.extractTVShowTMDbID(item)
	default:
		return ""
	}
}

// extractMovieTMDbID extracts TMDb ID from movie metadata or file paths
func (p *Processor) extractMovieTMDbID(item MediaItem) string {
	// First, try to get TMDb ID from Plex metadata
	for _, guid := range item.GetGuid() {
		if strings.Contains(guid.ID, "tmdb://") {
			parts := strings.Split(guid.ID, "//")
			if len(parts) > 1 {
				tmdbID := strings.Split(parts[1], "?")[0]
				return tmdbID
			}
		}
	}

	// If not found in metadata, try to extract from file paths
	for _, mediaItem := range item.GetMedia() {
		for _, part := range mediaItem.Part {
			if tmdbID := ExtractTMDbIDFromPath(part.File); tmdbID != "" {
				return tmdbID
			}
		}
	}

	return ""
}

// extractTVShowTMDbID extracts TMDb ID from TV show metadata or episode file paths
func (p *Processor) extractTVShowTMDbID(item MediaItem) string {
	// First check if we have TMDb GUID in the TV show metadata
	for _, guid := range item.GetGuid() {
		if strings.HasPrefix(guid.ID, "tmdb://") {
			return strings.TrimPrefix(guid.ID, "tmdb://")
		}
	}

	// If no TMDb GUID found, get episodes and check their file paths
	episodes, err := p.plexClient.GetTVShowEpisodes(item.GetRatingKey())
	if err != nil {
		fmt.Printf("⚠️ Error fetching episodes for %s: %v\n", item.GetTitle(), err)
		return ""
	}

	// Check file paths in episodes for TMDb ID - stop at first match
	for _, episode := range episodes {
		for _, mediaItem := range episode.Media {
			for _, part := range mediaItem.Part {
				if tmdbID := ExtractTMDbIDFromPath(part.File); tmdbID != "" {
					return tmdbID
				}
			}
		}
	}

	return ""
}

// ExtractTMDbIDFromPath extracts TMDb ID from file path using regex
func ExtractTMDbIDFromPath(filePath string) string {
	// Flexible regex pattern to match tmdb followed by digits with separators around the whole pattern
	// Matches: tmdb123, tmdb:123, {tmdb-456}, [tmdb=789], tmdb_012, etc.
	// Requires word boundaries or separators around the tmdb+digits pattern
	re := regexp.MustCompile(`(?i)(?:^|[^a-zA-Z0-9])tmdb[^a-zA-Z0-9]*(\d+)(?:[^a-zA-Z0-9]|$)`)
	matches := re.FindStringSubmatch(filePath)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}
