package main

import (
	"fmt"
	"os"
	"time"

	"github.com/nullable-eth/labelarr/internal/config"
	"github.com/nullable-eth/labelarr/internal/media"
	"github.com/nullable-eth/labelarr/internal/plex"
	"github.com/nullable-eth/labelarr/internal/tmdb"
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
	tmdbClient := tmdb.NewClient(cfg)

	// Initialize single processor
	processor := media.NewProcessor(cfg, plexClient, tmdbClient)

	fmt.Println("🏷️ Starting Labelarr with TMDb Integration...")
	fmt.Printf("📡 Server: %s://%s:%s\n", cfg.Protocol, cfg.PlexServer, cfg.PlexPort)

	// Get and validate libraries
	movieLibraries, tvLibraries := getLibraries(cfg, plexClient)

	// Handle REMOVE mode - run once and exit
	if cfg.IsRemoveMode() {
		handleRemoveMode(cfg, processor, movieLibraries, tvLibraries)
		os.Exit(0)
	}

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

// handleRemoveMode processes keyword removal and exits
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
}

// handleNormalMode runs the periodic processing
func handleNormalMode(cfg *config.Config, processor *media.Processor, movieLibraries, tvLibraries []plex.Library) {
	displayLibrarySelection(cfg, movieLibraries, tvLibraries)
	fmt.Printf("🔄 Starting periodic processing interval: %v\n", cfg.ProcessTimer)

	processFunc := func() {
		// Process movie libraries
		if len(movieLibraries) > 0 {
			if cfg.MovieProcessAll {
				for _, lib := range movieLibraries {
					fmt.Printf("🎬 Processing library: %s (ID: %s)\n", lib.Title, lib.Key)
					if err := processor.ProcessAllItems(lib.Key, media.MediaTypeMovie); err != nil {
						fmt.Printf("❌ Error processing movies: %v\n", err)
					}
				}
			} else if cfg.MovieLibraryID != "" {
				if err := processor.ProcessAllItems(cfg.MovieLibraryID, media.MediaTypeMovie); err != nil {
					fmt.Printf("❌ Error processing movies: %v\n", err)
				}
			}
		}

		// Process TV libraries
		if cfg.ProcessTVShows() {
			if cfg.TVProcessAll {
				for _, lib := range tvLibraries {
					fmt.Printf("📺 Processing TV library: %s (ID: %s)\n", lib.Title, lib.Key)
					if err := processor.ProcessAllItems(lib.Key, media.MediaTypeTV); err != nil {
						fmt.Printf("❌ Error processing TV shows: %v\n", err)
					}
				}
			} else if cfg.TVLibraryID != "" {
				if err := processor.ProcessAllItems(cfg.TVLibraryID, media.MediaTypeTV); err != nil {
					fmt.Printf("❌ Error processing TV shows: %v\n", err)
				}
			}
		}
	}

	// Process immediately on start
	processFunc()

	// Set up timer for periodic processing
	ticker := time.NewTicker(cfg.ProcessTimer)
	defer ticker.Stop()

	for range ticker.C {
		fmt.Printf("\n⏰ Timer triggered - processing at %s\n", time.Now().Format("15:04:05"))
		processFunc()
	}
}
