package media

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/nullable-eth/labelarr/internal/config"
	"github.com/nullable-eth/labelarr/internal/export"
	"github.com/nullable-eth/labelarr/internal/plex"
	"github.com/nullable-eth/labelarr/internal/radarr"
	"github.com/nullable-eth/labelarr/internal/sonarr"
	"github.com/nullable-eth/labelarr/internal/storage"
	"github.com/nullable-eth/labelarr/internal/tmdb"
	"github.com/nullable-eth/labelarr/internal/utils"
)

// MediaType represents the type of media being processed
type MediaType string

const (
	MediaTypeMovie MediaType = "movie"
	MediaTypeTV    MediaType = "tv"
)

// ProcessedItem is now imported from storage package

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
	config       *config.Config
	plexClient   *plex.Client
	tmdbClient   *tmdb.Client
	radarrClient *radarr.Client
	sonarrClient *sonarr.Client
	storage      *storage.Storage
	exporter     *export.Exporter
}

// NewProcessor creates a new generic media processor
func NewProcessor(cfg *config.Config, plexClient *plex.Client, tmdbClient *tmdb.Client, radarrClient *radarr.Client, sonarrClient *sonarr.Client) (*Processor, error) {
	// Initialize persistent storage
	stor, err := storage.NewStorage(cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	processor := &Processor{
		config:       cfg,
		plexClient:   plexClient,
		tmdbClient:   tmdbClient,
		radarrClient: radarrClient,
		sonarrClient: sonarrClient,
		storage:      stor,
	}

	// Initialize exporter if export is enabled
	if cfg.HasExportEnabled() {
		exporter, err := export.NewExporter(cfg.ExportLocation, cfg.ExportLabels, cfg.ExportMode)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize exporter: %w", err)
		}
		processor.exporter = exporter

		fmt.Printf("📤 Export enabled: Writing file paths for labels %v to %s\n", cfg.ExportLabels, cfg.ExportLocation)
	}

	// Log storage initialization
	count := stor.Count()
	if count > 0 {
		fmt.Printf("📁 Loaded %d previously processed items from storage\n", count)
	}

	return processor, nil
}

// GetExporter returns the exporter instance if export is enabled
func (p *Processor) GetExporter() *export.Exporter {
	return p.exporter
}

// ProcessAllItems processes all items in the specified library
func (p *Processor) ProcessAllItems(libraryID string, libraryName string, mediaType MediaType) error {
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

	// Set current library in exporter if export is enabled
	if p.exporter != nil {
		if err := p.exporter.SetCurrentLibrary(libraryName); err != nil {
			fmt.Printf("⚠️ Warning: Failed to set current library for export: %v\n", err)
		}
	}

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

	if p.config.ForceUpdate {
		fmt.Printf("🔄 FORCE UPDATE MODE: All items will be reprocessed regardless of previous processing\n")
	}

	if p.config.VerboseLogging {
		fmt.Printf("🔎 Starting detailed processing with verbose logging enabled...\n")
	} else {
		fmt.Printf("⏳ Processing %s... (enable VERBOSE_LOGGING=true for detailed lookup information)\n", displayName)
	}

	newItems := 0
	updatedItems := 0
	skippedItems := 0
	skippedAlreadyExist := 0

	// Progress tracking
	processedCount := 0
	lastProgressReport := 0

	for _, item := range items {
		processedCount++

		// Show progress for large libraries
		if totalCount > 100 {
			progress := (processedCount * 100) / totalCount
			if progress >= lastProgressReport+10 {
				fmt.Printf("📊 Progress: %d%% (%d/%d %s processed)\n", progress, processedCount, totalCount, displayName)
				lastProgressReport = progress
			}
		}
		processed, exists := p.storage.Get(item.GetRatingKey())
		if exists && processed.KeywordsSynced && processed.UpdateField == p.config.UpdateField && !p.config.ForceUpdate {
			skippedItems++
			skippedAlreadyExist++
			continue
		}

		// Silently check if we need to process this item
		tmdbID := p.extractTMDbID(item, mediaType)
		if tmdbID == "" {
			// Still try to export if export is enabled, even without TMDb ID
			if p.exporter != nil {
				details, err := p.getItemDetails(item.GetRatingKey(), mediaType)
				if err == nil {
					// Extract current labels for export
					currentLabels := p.extractCurrentValues(details)

					// Extract file paths and sizes
					fileInfos, err := p.extractFileInfos(details, mediaType)
					if err == nil && len(fileInfos) > 0 {
						// Accumulate the item for export
						if err := p.exporter.ExportItemWithSizes(item.GetTitle(), currentLabels, fileInfos); err == nil {
							if p.config.VerboseLogging {
								fmt.Printf("   📤 Accumulated %d file paths for %s (no TMDb ID)\n", len(fileInfos), item.GetTitle())
							}
						}
					}
				}
			}

			skippedItems++
			if p.config.VerboseLogging && skippedItems <= 10 {
				fmt.Printf("   ⏭️ Skipped %s: %s (%d) - No TMDb ID found\n", strings.TrimSuffix(displayName, "s"), item.GetTitle(), item.GetYear())
			}
			continue
		}

		// Silently fetch keywords and details to check if processing is needed
		keywords, err := p.getKeywords(tmdbID, mediaType)
		if err != nil {
			if p.config.VerboseLogging {
				fmt.Printf("   ❌ Error fetching keywords for TMDb ID %s: %v\n", tmdbID, err)
			}
			skippedItems++
			continue
		}

		if p.config.VerboseLogging {
			fmt.Printf("   📥 Fetched %d keywords from TMDb: %v\n", len(keywords), keywords)
		}

		details, err := p.getItemDetails(item.GetRatingKey(), mediaType)
		if err != nil {
			if p.config.VerboseLogging {
				fmt.Printf("   ❌ Error fetching item details: %v\n", err)
			}
			skippedItems++
			continue
		}

		currentValues := p.extractCurrentValues(details)
		if p.config.VerboseLogging {
			fmt.Printf("   📋 Current %ss in Plex: %v\n", p.config.UpdateField, currentValues)
		}

		currentValuesMap := make(map[string]bool)
		for _, val := range currentValues {
			currentValuesMap[strings.ToLower(val)] = true
		}

		allKeywordsExist := true
		var missingKeywords []string
		for _, keyword := range keywords {
			if !currentValuesMap[strings.ToLower(keyword)] {
				allKeywordsExist = false
				missingKeywords = append(missingKeywords, keyword)
			}
		}

		if allKeywordsExist && !p.config.ForceUpdate {
			// Silently skip - no verbose output
			if p.config.VerboseLogging {
				fmt.Printf("   ✨ Already has all keywords, skipping\n")
			}

			// Still export if export is enabled, even if no keyword updates are needed
			if p.exporter != nil {
				// Extract current labels for export
				currentLabels := p.extractCurrentValues(details)

				// Extract file paths and sizes
				fileInfos, err := p.extractFileInfos(details, mediaType)
				if err != nil {
					if p.config.VerboseLogging {
						fmt.Printf("   ⚠️ Warning: Could not extract file paths for export: %v\n", err)
					}
				} else if len(fileInfos) > 0 {
					// Accumulate the item for export
					if err := p.exporter.ExportItemWithSizes(item.GetTitle(), currentLabels, fileInfos); err != nil {
						if p.config.VerboseLogging {
							fmt.Printf("   ⚠️ Warning: Export accumulation failed for %s: %v\n", item.GetTitle(), err)
						}
					} else if p.config.VerboseLogging {
						fmt.Printf("   📤 Accumulated %d file paths for %s (already had keywords)\n", len(fileInfos), item.GetTitle())
					}
				}
			}

			skippedItems++
			skippedAlreadyExist++
			continue
		}

		if p.config.ForceUpdate && allKeywordsExist {
			if p.config.VerboseLogging {
				fmt.Printf("   🔄 Force update enabled - reprocessing item with existing keywords\n")
			}
		}

		if p.config.VerboseLogging {
			fmt.Printf("   🆕 Missing keywords to add: %v\n", missingKeywords)
		}

		// Only show verbose output for completely new items (never processed before)
		if !exists {
			fmt.Printf("\n%s Processing new %s: %s (%d)\n", emoji, strings.TrimSuffix(displayName, "s"), item.GetTitle(), item.GetYear())

			// Show source of TMDb ID
			source := p.getTMDbIDSource(item, mediaType, tmdbID)
			fmt.Printf("🔑 TMDb ID: %s (source: %s)\n", tmdbID, source)
			fmt.Printf("🏷️ Found %d TMDb keywords\n", len(keywords))
		}

		// Show when we're about to apply labels/genres
		if p.config.VerboseLogging || !exists {
			fmt.Printf("🔄 Applying %d keywords to %s field...\n", len(keywords), p.config.UpdateField)
			if p.config.VerboseLogging {
				fmt.Printf("   Current %ss: %v\n", p.config.UpdateField, currentValues)
				fmt.Printf("   New keywords to add: %v\n", keywords)
			}
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

		// Show success message when labels/genres are applied
		if p.config.VerboseLogging || !exists {
			fmt.Printf("✅ Successfully applied %d keywords to Plex %s field\n", len(keywords), p.config.UpdateField)
		}

		// Export file paths if export is enabled
		if p.exporter != nil {
			// Get updated item details to get current labels
			updatedDetails, err := p.getItemDetails(item.GetRatingKey(), mediaType)
			if err != nil {
				if p.config.VerboseLogging {
					fmt.Printf("   ⚠️ Warning: Could not get updated details for export: %v\n", err)
				}
			} else {
				// Extract current labels for export
				currentLabels := p.extractCurrentValues(updatedDetails)

				// Extract file paths and sizes
				fileInfos, err := p.extractFileInfos(updatedDetails, mediaType)
				if err != nil {
					if p.config.VerboseLogging {
						fmt.Printf("   ⚠️ Warning: Could not extract file paths for export: %v\n", err)
					}
				} else if len(fileInfos) > 0 {
					// Accumulate the item for export
					if err := p.exporter.ExportItemWithSizes(item.GetTitle(), currentLabels, fileInfos); err != nil {
						if p.config.VerboseLogging {
							fmt.Printf("   ⚠️ Warning: Export accumulation failed for %s: %v\n", item.GetTitle(), err)
						}
					} else if p.config.VerboseLogging {
						fmt.Printf("   📤 Accumulated %d file paths for %s\n", len(fileInfos), item.GetTitle())
					}
				}
			}
		}

		processedItem := &storage.ProcessedItem{
			RatingKey:      item.GetRatingKey(),
			Title:          item.GetTitle(),
			TMDbID:         tmdbID,
			LastProcessed:  time.Now(),
			KeywordsSynced: true,
			UpdateField:    p.config.UpdateField,
		}

		if err := p.storage.Set(processedItem); err != nil {
			fmt.Printf("⚠️ Warning: Failed to save processed item to storage: %v\n", err)
		}

		if exists {
			updatedItems++
		} else {
			newItems++
			fmt.Printf("✅ Successfully processed new %s: %s\n", strings.TrimSuffix(displayName, "s"), item.GetTitle())
		}

		time.Sleep(500 * time.Millisecond)
	}

	// Show verbose summary if items were skipped
	if p.config.VerboseLogging && skippedItems > 10 {
		fmt.Printf("   ... and %d more items skipped\n", skippedItems-10)
	}

	fmt.Printf("\n📊 Processing Summary:\n")
	fmt.Printf("  📈 Total %s in library: %d\n", displayName, totalCount)
	fmt.Printf("  🆕 New %s processed: %d\n", displayName, newItems)
	fmt.Printf("  🔄 Updated %s: %d\n", displayName, updatedItems)
	fmt.Printf("  ⏭️ Skipped %s: %d\n", displayName, skippedItems)
	if skippedAlreadyExist > 0 {
		fmt.Printf("  ✨ Already have all keywords: %d\n", skippedAlreadyExist)
	}

	// Show export summary if export is enabled
	if p.exporter != nil {
		librarySummary, err := p.exporter.GetLibraryExportSummary()
		if err != nil {
			fmt.Printf("  ⚠️ Export summary error: %v\n", err)
		} else {
			fmt.Printf("\n📤 Export Summary for %s:\n", libraryName)
			totalAccumulated := 0

			// Show current library summary
			currentLibrary := p.exporter.GetCurrentLibrary()
			if librarySummary[currentLibrary] != nil {
				for label, count := range librarySummary[currentLibrary] {
					fmt.Printf("  📁 %s: %d file paths accumulated\n", label, count)
					totalAccumulated += count
				}
			}

			fmt.Printf("📊 Total accumulated in this library: %d file paths\n", totalAccumulated)
		}
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
	// Clean duplicates: remove old unnormalized versions when normalized versions are present
	// This helps clean up cases like having both "sci-fi" and "Sci-Fi"
	cleanedValues := utils.CleanDuplicateKeywords(currentValues, keywords)

	if p.config.VerboseLogging && len(cleanedValues) != len(currentValues) {
		removedCount := len(currentValues) - len(cleanedValues) + len(keywords)
		fmt.Printf("   🧹 Cleaned %d duplicate/unnormalized keywords\n", removedCount)
	}

	return p.updateItemField(itemID, libraryID, cleanedValues, mediaType)
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
	if p.config.VerboseLogging {
		fmt.Printf("\n🔍 Starting TMDb ID lookup for movie: %s (%d)\n", item.GetTitle(), item.GetYear())
		fmt.Printf("   📋 Available Plex GUIDs:\n")
		for _, guid := range item.GetGuid() {
			fmt.Printf("      - %s\n", guid.ID)
		}
	}

	// First, try to get TMDb ID from Plex metadata
	for _, guid := range item.GetGuid() {
		if strings.Contains(guid.ID, "tmdb://") {
			parts := strings.Split(guid.ID, "//")
			if len(parts) > 1 {
				tmdbID := strings.Split(parts[1], "?")[0]
				if p.config.VerboseLogging {
					fmt.Printf("   ✅ Found TMDb ID in Plex metadata: %s\n", tmdbID)
				}
				return tmdbID
			}
		}
	}

	// If Radarr is enabled, try to match via Radarr
	if p.config.UseRadarr && p.radarrClient != nil {
		if p.config.VerboseLogging {
			fmt.Printf("   🎬 Checking Radarr for movie match...\n")
		}

		// Try to match by title and year first
		if p.config.VerboseLogging {
			fmt.Printf("      → Searching by title: \"%s\" year: %d\n", item.GetTitle(), item.GetYear())
		}
		movie, err := p.radarrClient.FindMovieMatch(item.GetTitle(), item.GetYear())
		if err == nil && movie != nil {
			tmdbID := p.radarrClient.GetTMDbIDFromMovie(movie)
			if p.config.VerboseLogging {
				fmt.Printf("      ✅ Found match in Radarr: %s (TMDb: %s)\n", movie.Title, tmdbID)
			}
			return tmdbID
		} else if p.config.VerboseLogging {
			fmt.Printf("      ❌ No match found by title/year\n")
		}

		// Try to match by file path
		if p.config.VerboseLogging {
			fmt.Printf("      → Searching by file path...\n")
		}
		for _, mediaItem := range item.GetMedia() {
			for _, part := range mediaItem.Part {
				if p.config.VerboseLogging {
					fmt.Printf("         - Checking: %s\n", part.File)
				}
				movie, err := p.radarrClient.GetMovieByPath(part.File)
				if err == nil && movie != nil {
					tmdbID := p.radarrClient.GetTMDbIDFromMovie(movie)
					if p.config.VerboseLogging {
						fmt.Printf("      ✅ Found match by file path: %s (TMDb: %s)\n", movie.Title, tmdbID)
					}
					return tmdbID
				}
			}
		}
		if p.config.VerboseLogging {
			fmt.Printf("      ❌ No match found by file path\n")
		}

		// Try to match by IMDb ID if available
		for _, guid := range item.GetGuid() {
			if strings.Contains(guid.ID, "imdb://") {
				imdbID := strings.TrimPrefix(guid.ID, "imdb://")
				if p.config.VerboseLogging {
					fmt.Printf("      → Searching by IMDb ID: %s\n", imdbID)
				}
				movie, err := p.radarrClient.GetMovieByIMDbID(imdbID)
				if err == nil && movie != nil {
					tmdbID := p.radarrClient.GetTMDbIDFromMovie(movie)
					if p.config.VerboseLogging {
						fmt.Printf("      ✅ Found match by IMDb ID: %s (TMDb: %s)\n", movie.Title, tmdbID)
					}
					return tmdbID
				} else if p.config.VerboseLogging {
					fmt.Printf("      ❌ No match found by IMDb ID\n")
				}
			}
		}
	}

	// If not found in Radarr or Radarr not enabled, try to extract from file paths
	if p.config.VerboseLogging {
		fmt.Printf("   📁 Checking file paths for TMDb ID pattern...\n")
	}
	for _, mediaItem := range item.GetMedia() {
		for _, part := range mediaItem.Part {
			if p.config.VerboseLogging {
				fmt.Printf("      - Checking: %s\n", part.File)
			}
			if tmdbID := ExtractTMDbIDFromPath(part.File); tmdbID != "" {
				if p.config.VerboseLogging {
					fmt.Printf("      ✅ Found TMDb ID in file path: %s\n", tmdbID)
				}
				return tmdbID
			}
		}
	}

	if p.config.VerboseLogging {
		fmt.Printf("   ❌ No TMDb ID found for movie: %s\n", item.GetTitle())
	}

	return ""
}

// extractTVShowTMDbID extracts TMDb ID from TV show metadata or episode file paths
func (p *Processor) extractTVShowTMDbID(item MediaItem) string {
	if p.config.VerboseLogging {
		fmt.Printf("\n🔍 Starting TMDb ID lookup for TV show: %s (%d)\n", item.GetTitle(), item.GetYear())
		fmt.Printf("   📋 Available Plex GUIDs:\n")
		for _, guid := range item.GetGuid() {
			fmt.Printf("      - %s\n", guid.ID)
		}
	}

	// First check if we have TMDb GUID in the TV show metadata
	for _, guid := range item.GetGuid() {
		if strings.HasPrefix(guid.ID, "tmdb://") {
			tmdbID := strings.TrimPrefix(guid.ID, "tmdb://")
			if p.config.VerboseLogging {
				fmt.Printf("   ✅ Found TMDb ID in Plex metadata: %s\n", tmdbID)
			}
			return tmdbID
		}
	}

	// If Sonarr is enabled, try to match via Sonarr
	if p.config.UseSonarr && p.sonarrClient != nil {
		if p.config.VerboseLogging {
			fmt.Printf("   📺 Checking Sonarr for series match...\n")
		}

		// Try to match by title and year first
		if p.config.VerboseLogging {
			fmt.Printf("      → Searching by title: \"%s\" year: %d\n", item.GetTitle(), item.GetYear())
		}
		series, err := p.sonarrClient.FindSeriesMatch(item.GetTitle(), item.GetYear())
		if err == nil && series != nil {
			tmdbID := p.sonarrClient.GetTMDbIDFromSeries(series)
			if p.config.VerboseLogging {
				fmt.Printf("      ✅ Found match in Sonarr: %s (TMDb: %s)\n", series.Title, tmdbID)
			}
			return tmdbID
		} else if p.config.VerboseLogging {
			fmt.Printf("      ❌ No match found by title/year\n")
		}

		// Try to match by TVDb ID if available
		for _, guid := range item.GetGuid() {
			if strings.Contains(guid.ID, "tvdb://") {
				tvdbIDStr := strings.TrimPrefix(guid.ID, "tvdb://")
				// Parse TVDb ID to int
				var tvdbID int
				if _, err := fmt.Sscanf(tvdbIDStr, "%d", &tvdbID); err == nil {
					if p.config.VerboseLogging {
						fmt.Printf("      → Searching by TVDb ID: %d\n", tvdbID)
					}
					series, err := p.sonarrClient.GetSeriesByTVDbID(tvdbID)
					if err == nil && series != nil {
						tmdbID := p.sonarrClient.GetTMDbIDFromSeries(series)
						if p.config.VerboseLogging {
							fmt.Printf("      ✅ Found match by TVDb ID: %s (TMDb: %s)\n", series.Title, tmdbID)
						}
						return tmdbID
					} else if p.config.VerboseLogging {
						fmt.Printf("      ❌ No match found by TVDb ID\n")
					}
				}
			}
		}

		// Try to match by IMDb ID if available
		for _, guid := range item.GetGuid() {
			if strings.Contains(guid.ID, "imdb://") {
				imdbID := strings.TrimPrefix(guid.ID, "imdb://")
				if p.config.VerboseLogging {
					fmt.Printf("      → Searching by IMDb ID: %s\n", imdbID)
				}
				series, err := p.sonarrClient.GetSeriesByIMDbID(imdbID)
				if err == nil && series != nil {
					tmdbID := p.sonarrClient.GetTMDbIDFromSeries(series)
					if p.config.VerboseLogging {
						fmt.Printf("      ✅ Found match by IMDb ID: %s (TMDb: %s)\n", series.Title, tmdbID)
					}
					return tmdbID
				} else if p.config.VerboseLogging {
					fmt.Printf("      ❌ No match found by IMDb ID\n")
				}
			}
		}

		// Try to match by file path from episodes
		if p.config.VerboseLogging {
			fmt.Printf("      → Searching by episode file paths...\n")
		}
		episodes, err := p.plexClient.GetTVShowEpisodes(item.GetRatingKey())
		if err == nil {
			episodeCount := 0
			for _, episode := range episodes {
				for _, mediaItem := range episode.Media {
					for _, part := range mediaItem.Part {
						episodeCount++
						if episodeCount <= 5 && p.config.VerboseLogging {
							fmt.Printf("         - Checking: %s\n", part.File)
						}
						series, err := p.sonarrClient.GetSeriesByPath(part.File)
						if err == nil && series != nil {
							tmdbID := p.sonarrClient.GetTMDbIDFromSeries(series)
							if p.config.VerboseLogging {
								fmt.Printf("      ✅ Found match by file path: %s (TMDb: %s)\n", series.Title, tmdbID)
							}
							return tmdbID
						}
					}
				}
			}
			if episodeCount > 5 && p.config.VerboseLogging {
				fmt.Printf("         ... and %d more episodes\n", episodeCount-5)
			}
			if p.config.VerboseLogging {
				fmt.Printf("      ❌ No match found by file path\n")
			}
		} else if p.config.VerboseLogging {
			fmt.Printf("      ⚠️ Could not fetch episodes: %v\n", err)
		}
	}

	// If no TMDb GUID found and Sonarr not enabled, get episodes and check their file paths
	if p.config.VerboseLogging {
		fmt.Printf("   📁 Checking episode file paths for TMDb ID pattern...\n")
	}
	episodes, err := p.plexClient.GetTVShowEpisodes(item.GetRatingKey())
	if err != nil {
		if p.config.VerboseLogging {
			fmt.Printf("   ⚠️ Error fetching episodes: %v\n", err)
		} else {
			fmt.Printf("⚠️ Error fetching episodes for %s: %v\n", item.GetTitle(), err)
		}
		return ""
	}

	// Check file paths in episodes for TMDb ID - stop at first match
	episodeCount := 0
	for _, episode := range episodes {
		for _, mediaItem := range episode.Media {
			for _, part := range mediaItem.Part {
				episodeCount++
				if episodeCount <= 5 && p.config.VerboseLogging {
					fmt.Printf("      - Checking: %s\n", part.File)
				}
				if tmdbID := ExtractTMDbIDFromPath(part.File); tmdbID != "" {
					if p.config.VerboseLogging {
						fmt.Printf("      ✅ Found TMDb ID in file path: %s\n", tmdbID)
					}
					return tmdbID
				}
			}
		}
	}

	if episodeCount > 5 && p.config.VerboseLogging {
		fmt.Printf("      ... and %d more episodes\n", episodeCount-5)
	}

	if p.config.VerboseLogging {
		fmt.Printf("   ❌ No TMDb ID found for TV show: %s\n", item.GetTitle())
	}

	return ""
}

// getTMDbIDSource determines the source of the TMDb ID
func (p *Processor) getTMDbIDSource(item MediaItem, mediaType MediaType, tmdbID string) string {
	// Check if it's from Plex metadata
	for _, guid := range item.GetGuid() {
		if strings.Contains(guid.ID, "tmdb://") {
			return "Plex metadata"
		}
	}

	// Check if it could be from Radarr/Sonarr
	if mediaType == MediaTypeMovie && p.config.UseRadarr && p.radarrClient != nil {
		// Quick check if movie exists in Radarr with this TMDb ID
		movie, err := p.radarrClient.FindMovieMatch(item.GetTitle(), item.GetYear())
		if err == nil && movie != nil && p.radarrClient.GetTMDbIDFromMovie(movie) == tmdbID {
			return "Radarr"
		}
	}

	if mediaType == MediaTypeTV && p.config.UseSonarr && p.sonarrClient != nil {
		// Quick check if series exists in Sonarr with this TMDb ID
		series, err := p.sonarrClient.FindSeriesMatch(item.GetTitle(), item.GetYear())
		if err == nil && series != nil && p.sonarrClient.GetTMDbIDFromSeries(series) == tmdbID {
			return "Sonarr"
		}
	}

	// Must be from file path
	return "file path"
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

// extractFilePaths extracts all file paths from a media item
func (p *Processor) extractFilePaths(item MediaItem, mediaType MediaType) ([]string, error) {
	fileInfos, err := p.extractFileInfos(item, mediaType)
	if err != nil {
		return nil, err
	}

	// Convert FileInfo back to paths for backwards compatibility
	var filePaths []string
	for _, fileInfo := range fileInfos {
		filePaths = append(filePaths, fileInfo.Path)
	}

	return filePaths, nil
}

// extractFileInfos extracts all file paths and sizes from a media item
func (p *Processor) extractFileInfos(item MediaItem, mediaType MediaType) ([]export.FileInfo, error) {
	var fileInfos []export.FileInfo

	switch mediaType {
	case MediaTypeMovie:
		// For movies, get file info directly from the media items
		for _, media := range item.GetMedia() {
			for _, part := range media.Part {
				if part.File != "" {
					fileInfos = append(fileInfos, export.FileInfo{
						Path: part.File,
						Size: part.Size,
					})
				}
			}
		}
	case MediaTypeTV:
		// For TV shows, get file info from all episodes (use GetAllTVShowEpisodes for export)
		episodes, err := p.plexClient.GetAllTVShowEpisodes(item.GetRatingKey())
		if err != nil {
			return nil, fmt.Errorf("failed to get all episodes for TV show %s: %w", item.GetTitle(), err)
		}

		for _, episode := range episodes {
			for _, media := range episode.Media {
				for _, part := range media.Part {
					if part.File != "" {
						fileInfos = append(fileInfos, export.FileInfo{
							Path: part.File,
							Size: part.Size,
						})
					}
				}
			}
		}
	}

	return fileInfos, nil
}
