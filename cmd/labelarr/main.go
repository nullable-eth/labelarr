package main

import (
	"fmt"
	"os"
	"time"

	"github.com/nullable-eth/labelarr/internal/config"
	"github.com/nullable-eth/labelarr/internal/media"
	"github.com/nullable-eth/labelarr/internal/plex"
	"github.com/nullable-eth/labelarr/internal/radarr"
	"github.com/nullable-eth/labelarr/internal/sonarr"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		fmt.Printf("❌ Configuration error: %v\n", err)
		os.Exit(1)
	}

	// Initialize clients
	plexClient := plex.NewClient(cfg)

	// Initialize Radarr client if enabled
	var radarrClient *radarr.Client
	if cfg.UseRadarr {
		radarrClient = radarr.NewClient(cfg.RadarrURL, cfg.RadarrAPIKey)
		if err := radarrClient.TestConnection(); err != nil {
			fmt.Printf("❌ Failed to connect to Radarr: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✅ Successfully connected to Radarr")
	}

	// Initialize Sonarr client if enabled
	var sonarrClient *sonarr.Client
	if cfg.UseSonarr {
		sonarrClient = sonarr.NewClient(cfg.SonarrURL, cfg.SonarrAPIKey)
		if err := sonarrClient.TestConnection(); err != nil {
			fmt.Printf("❌ Failed to connect to Sonarr: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✅ Successfully connected to Sonarr")
	}

	// Initialize single processor
	processor, err := media.NewProcessor(cfg, plexClient, radarrClient, sonarrClient)
	if err != nil {
		fmt.Printf("❌ Failed to initialize processor: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("🏷️ Starting Labelarr withOUT TMDb Integration...")
	fmt.Printf("📡 Server: %s://%s:%s\n", cfg.Protocol, cfg.PlexServer, cfg.PlexPort)

	// Get and validate libraries
	movieLibraries, tvLibraries := getLibraries(cfg, plexClient)

	/* // Handle REMOVE mode - run once and exit
	if cfg.IsRemoveMode() {
		handleRemoveMode(cfg, processor, movieLibraries, tvLibraries)
		os.Exit(0)
	} */

	// Handle normal processing mode
	handleNormalMode(cfg, processor, movieLibraries, tvLibraries)
}

// getLibraries fetches, separates, and validates libraries from Plex
func getLibraries(cfg *config.Config, plexClient *plex.Client) ([]plex.Library, []plex.Library) {
	// Get all libraries
	fmt.Println("📚 Fetching all libraries...")
	libraries, err := plexClient.GetAllLibraries()
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

	// Separate libraries by type
	var movieLibraries []plex.Library
	var tvLibraries []plex.Library
	for _, lib := range libraries {
		switch lib.Type {
		case "movie":
			movieLibraries = append(movieLibraries, lib)
		case "show":
			tvLibraries = append(tvLibraries, lib)
		}
	}

	// Validate libraries exist
	if len(movieLibraries) == 0 && !cfg.ProcessTVShows() {
		fmt.Println("❌ No movie library found!")
		os.Exit(1)
	}

	if cfg.ProcessTVShows() && len(tvLibraries) == 0 {
		fmt.Println("❌ No TV show library found!")
		os.Exit(1)
	}

	return movieLibraries, tvLibraries
}

// displayLibrarySelection shows which libraries will be processed
func displayLibrarySelection(cfg *config.Config, movieLibraries, tvLibraries []plex.Library) {
	// Movie library selection
	if cfg.ProcessMovies() {
		if cfg.MovieProcessAll {
			fmt.Printf("🎯 Processing all %d movie libraries\n", len(movieLibraries))
		} else if cfg.MovieLibraryID != "" {
			found := false
			for _, lib := range movieLibraries {
				if lib.Key == cfg.MovieLibraryID {
					fmt.Printf("\n🎯 Using specified movie library: %s (ID: %s)\n", lib.Title, lib.Key)
					found = true
					break
				}
			}
			if !found {
				fmt.Printf("❌ Movie library with ID %s not found!\n", cfg.MovieLibraryID)
				os.Exit(1)
			}
		}
	}
	// TV library selection
	if cfg.ProcessTVShows() {
		if cfg.TVProcessAll {
			fmt.Printf("📺 Processing all %d TV show libraries\n", len(tvLibraries))
		} else if cfg.TVLibraryID != "" {
			found := false
			for _, lib := range tvLibraries {
				if lib.Key == cfg.TVLibraryID {
					fmt.Printf("\n📺 Using specified TV library: %s (ID: %s)\n", lib.Title, lib.Key)
					found = true
					break
				}
			}
			if !found {
				fmt.Printf("❌ TV library with ID %s not found!\n", cfg.TVLibraryID)
				os.Exit(1)
			}
		} else {
			fmt.Printf("\n📺 Using TV library: %s (ID: %s)\n", tvLibraries[0].Title, tvLibraries[0].Key)
		}
	}
}

/* // handleRemoveMode processes keyword removal and exits
func handleRemoveMode(cfg *config.Config, processor *media.Processor, movieLibraries, tvLibraries []plex.Library) {
	// Display selected libraries
	displayLibrarySelection(cfg, movieLibraries, tvLibraries)
	fmt.Printf("\n🗑️ Starting keyword removal from (field: %s, lock: %s)...\n", cfg.UpdateField, cfg.RemoveMode)

	if cfg.ProcessMovies() {
		// Process movie libraries
		if cfg.MovieProcessAll {
			for _, lib := range movieLibraries {
				fmt.Printf("🎬 Processing library: %s (ID: %s)\n", lib.Title, lib.Key)
				if err := processor.RemoveKeywordsFromItems(lib.Key, media.MediaTypeMovie); err != nil {
					fmt.Printf("❌ Error removing keywords from movies: %v\n", err)
				}
			}
		} else if cfg.MovieLibraryID != "" {
			if err := processor.RemoveKeywordsFromItems(cfg.MovieLibraryID, media.MediaTypeMovie); err != nil {
				fmt.Printf("❌ Error removing keywords from movies: %v\n", err)
			}
		}
	}
	// Process TV libraries
	if cfg.ProcessTVShows() {
		if cfg.TVProcessAll {
			for _, lib := range tvLibraries {
				fmt.Printf("📺 Processing TV library: %s (ID: %s)\n", lib.Title, lib.Key)
				if err := processor.RemoveKeywordsFromItems(lib.Key, media.MediaTypeTV); err != nil {
					fmt.Printf("❌ Error removing keywords from TV shows: %v\n", err)
				}
			}
		} else if cfg.TVLibraryID != "" {
			if err := processor.RemoveKeywordsFromItems(cfg.TVLibraryID, media.MediaTypeTV); err != nil {
				fmt.Printf("❌ Error removing keywords from TV shows: %v\n", err)
			}
		}
	}
	fmt.Println("\n✅ Keyword removal completed. Exiting.")
} */

// handleNormalMode runs the periodic processing
func handleNormalMode(cfg *config.Config, processor *media.Processor, movieLibraries, tvLibraries []plex.Library) {
	displayLibrarySelection(cfg, movieLibraries, tvLibraries)
	fmt.Printf("🔄 Starting periodic processing interval: %v\n", cfg.ProcessTimer)

	processFunc := func() {
		// Process movie libraries
		fmt.Printf("✅ Process movie libraries - start\n")
		if len(movieLibraries) > 0 {
			if cfg.MovieProcessAll {
				for _, lib := range movieLibraries {
					fmt.Printf("🎬 Processing library: %s (ID: %s)\n", lib.Title, lib.Key)
					if err := processor.ProcessAllItems(lib.Key, lib.Title, media.MediaTypeMovie); err != nil {
						fmt.Printf("❌ Error processing movies: %v\n", err)
					}
				}
			} else if cfg.MovieLibraryID != "" {
				// Find the library name for the specified ID
				libraryName := "Movies" // Default fallback
				for _, lib := range movieLibraries {
					if lib.Key == cfg.MovieLibraryID {
						libraryName = lib.Title
						break
					}
				}
				if err := processor.ProcessAllItems(cfg.MovieLibraryID, libraryName, media.MediaTypeMovie); err != nil {
					fmt.Printf("❌ Error processing movies: %v\n", err)
				}
			}
		}

		// Process TV libraries
		fmt.Printf("✅ Process TV libraries - start\n")
		if cfg.ProcessTVShows() {
			if cfg.TVProcessAll {
				for _, lib := range tvLibraries {
					fmt.Printf("📺 Processing TV library: %s (ID: %s)\n", lib.Title, lib.Key)
					if err := processor.ProcessAllItems(lib.Key, lib.Title, media.MediaTypeTV); err != nil {
						fmt.Printf("❌ Error processing TV shows: %v\n", err)
					}
				}
			} else if cfg.TVLibraryID != "" {
				// Find the library name for the specified ID
				libraryName := "TV Shows" // Default fallback
				for _, lib := range tvLibraries {
					if lib.Key == cfg.TVLibraryID {
						libraryName = lib.Title
						break
					}
				}
				if err := processor.ProcessAllItems(cfg.TVLibraryID, libraryName, media.MediaTypeTV); err != nil {
					fmt.Printf("❌ Error processing TV shows: %v\n", err)
				}
			}
		}

		// Write all accumulated export files after processing all libraries
		fmt.Printf("✅ Write all accumulated export files after processing all libraries - start\n")
		if cfg.HasExportEnabled() {
			fmt.Printf("\n📤 Writing export files to %s...\n", cfg.ExportLocation)
			if exporter := processor.GetExporter(); exporter != nil {
				totalSummary, err := exporter.GetExportSummary()
				if err != nil {
					fmt.Printf("❌ Error getting export summary: %v\n", err)
				} else {
					totalAccumulated := 0
					for label, count := range totalSummary {
						if count > 0 {
							fmt.Printf("  📁 %s: %d total file paths\n", label, count)
						}
						totalAccumulated += count
					}

					if totalAccumulated > 0 {
						fmt.Printf("📝 Writing %d total file paths across all libraries...\n", totalAccumulated)
						if err := exporter.FlushAll(); err != nil {
							fmt.Printf("❌ Failed to write export files: %v\n", err)
						} else {
							if cfg.ExportMode == "json" {
								fmt.Printf("✅ Successfully wrote export data to export.json\n")
							} else {
								fmt.Printf("✅ Successfully wrote export files to library subdirectories\n")
								fmt.Printf("📊 Generated summary.txt with detailed statistics and file sizes\n")
							}
						}
					} else {
						fmt.Printf("📭 No matching items found for export labels\n")
						// Still create empty files for each label in each library
						if err := exporter.FlushAll(); err != nil {
							fmt.Printf("❌ Failed to create export files: %v\n", err)
						} else {
							if cfg.ExportMode == "json" {
								fmt.Printf("✅ Created empty export.json file\n")
							} else {
								fmt.Printf("✅ Created empty export files in library subdirectories\n")
								fmt.Printf("📊 Generated summary.txt with export statistics\n")
							}
						}
					}
				}
			}
		}
	}

	// Process immediately on start
	processFunc()
	fmt.Printf("✅ processFunc - end\n")

	// Set up timer for periodic processing
	ticker := time.NewTicker(cfg.ProcessTimer)
	defer ticker.Stop()

	for range ticker.C {
		fmt.Printf("\n⏰ Timer triggered - processing at %s\n", time.Now().Format("15:04:05"))
		processFunc()
	}
}
