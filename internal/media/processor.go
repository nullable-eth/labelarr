package media

import (
	"fmt"
	//"regexp"
	"strings"
	"time"

	"github.com/nullable-eth/labelarr/internal/config"
	"github.com/nullable-eth/labelarr/internal/export"
	"github.com/nullable-eth/labelarr/internal/plex"
	"github.com/nullable-eth/labelarr/internal/radarr"
	"github.com/nullable-eth/labelarr/internal/sonarr"
	"github.com/nullable-eth/labelarr/internal/storage"
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
	radarrClient *radarr.Client
	sonarrClient *sonarr.Client
	storage      *storage.Storage
	exporter     *export.Exporter
}

// NewProcessor creates a new generic media processor
func NewProcessor(cfg *config.Config, plexClient *plex.Client, radarrClient *radarr.Client, sonarrClient *sonarr.Client) (*Processor, error) {
	// Initialize persistent storage only if DATA_DIR is set
	var stor *storage.Storage
	if cfg.DataDir != "" {
		var err error
		stor, err = storage.NewStorage(cfg.DataDir)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize storage: %w", err)
		}
	}

	processor := &Processor{
		config:       cfg,
		plexClient:   plexClient,
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
	if stor != nil {
		count := stor.Count()
		if count > 0 {
			fmt.Printf("📁 Loaded %d previously processed items from storage\n", count)
		}
	} else {
		fmt.Printf("🔄 Running in ephemeral mode - no persistent storage (set DATA_DIR to enable)\n")
	}

	return processor, nil
}

// GetExporter returns the exporter instance if export is enabled
func (p *Processor) GetExporter() *export.Exporter {
	return p.exporter
}

// ProcessAllItems processes all items in the specified library
func (p *Processor) ProcessAllItems(libraryID string, libraryName string, mediaType MediaType) error {
	var displayName string //, emoji string
	switch mediaType {
	case MediaTypeMovie:
		displayName = "movies"
		//emoji = "🎬"
	case MediaTypeTV:
		displayName = "tv shows"
		//emoji = "📺"
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
		// Check if already processed (only if storage is enabled)
		var exists bool
		if p.storage != nil {
			processed, storageExists := p.storage.Get(item.GetRatingKey())
			if storageExists && processed.KeywordsSynced && processed.UpdateField == p.config.UpdateField && !p.config.ForceUpdate {
				// Still try to export if export is enabled, even if already processed
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
									fmt.Printf("   📤 Accumulated %d file paths for %s (already processed)\n", len(fileInfos), item.GetTitle())
								}
							}
						}
					}
				}

				skippedItems++
				skippedAlreadyExist++
				continue
			}
			exists = storageExists
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

		// Save processed item (only if storage is enabled)
		if p.storage != nil {
			processedItem := &storage.ProcessedItem{
				RatingKey:      item.GetRatingKey(),
				Title:          item.GetTitle(),
				LastProcessed:  time.Now(),
				KeywordsSynced: true,
				UpdateField:    p.config.UpdateField,
			}

			if err := p.storage.Set(processedItem); err != nil {
				fmt.Printf("⚠️ Warning: Failed to save processed item to storage: %v\n", err)
			}
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
