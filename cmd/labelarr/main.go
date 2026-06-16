package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nullable-eth/labelarr/internal/config"
	"github.com/nullable-eth/labelarr/internal/media"
	"github.com/nullable-eth/labelarr/internal/plex"
	"github.com/nullable-eth/labelarr/internal/radarr"
	"github.com/nullable-eth/labelarr/internal/sonarr"
	"github.com/nullable-eth/labelarr/internal/tmdb"
	"github.com/nullable-eth/labelarr/internal/utils"
	"github.com/nullable-eth/labelarr/internal/version"
	"github.com/nullable-eth/labelarr/internal/webhook"
)

func main() {
	fmt.Printf("[INFO] Labelarr v%s\n", version.Version)

	cfg := config.Load()

	if err := cfg.Validate(); err != nil {
		fmt.Printf("[ERROR] Configuration error: %v\n", err)
		os.Exit(1)
	}

	plexClient := plex.NewClient(cfg)
	tmdbClient := tmdb.NewClient(cfg)

	if err := tmdbClient.TestConnection(); err != nil {
		fmt.Printf("[ERROR] Failed to connect to TMDb: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("[OK] Successfully connected to TMDb")

	var radarrClient *radarr.Client
	if cfg.UseRadarr {
		radarrClient = radarr.NewClient(cfg.RadarrURL, cfg.RadarrAPIKey)
		if err := radarrClient.TestConnection(); err != nil {
			fmt.Printf("[ERROR] Failed to connect to Radarr: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("[OK] Successfully connected to Radarr")
	}

	var sonarrClient *sonarr.Client
	if cfg.UseSonarr {
		sonarrClient = sonarr.NewClient(cfg.SonarrURL, cfg.SonarrAPIKey)
		if err := sonarrClient.TestConnection(); err != nil {
			fmt.Printf("[ERROR] Failed to connect to Sonarr: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("[OK] Successfully connected to Sonarr")
	}

	processor, err := media.NewProcessor(cfg, media.Clients{
		Plex:   plexClient,
		TMDb:   tmdbClient,
		Radarr: radarrClient,
		Sonarr: sonarrClient,
	})
	if err != nil {
		fmt.Printf("[ERROR] Failed to initialize processor: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("[INFO] Starting Labelarr with TMDb Integration...")
	fmt.Printf("[NET] Server: %s://%s:%s\n", cfg.Protocol, cfg.PlexServer, cfg.PlexPort)

	movieLibraries, tvLibraries := getLibraries(cfg, plexClient)

	if cfg.IsRemoveMode() {
		handleRemoveMode(cfg, processor, movieLibraries, tvLibraries)
		os.Exit(0)
	}

	handleNormalMode(cfg, processor, movieLibraries, tvLibraries)
}

func getLibraries(cfg *config.Config, plexClient *plex.Client) ([]plex.Library, []plex.Library) {
	fmt.Println("[INFO] Fetching all libraries...")
	libraries, err := plexClient.GetAllLibraries()
	if err != nil {
		fmt.Printf("[ERROR] Error fetching libraries: %v\n", err)
		os.Exit(1)
	}

	if len(libraries) == 0 {
		fmt.Println("[ERROR] No libraries found!")
		os.Exit(1)
	}

	fmt.Printf("[OK] Found %d libraries:\n", len(libraries))
	for _, lib := range libraries {
		fmt.Printf("  ID: %s - %s (%s)\n", lib.Key, lib.Title, lib.Type)
	}

	var movieLibraries, tvLibraries []plex.Library
	for _, lib := range libraries {
		switch lib.Type {
		case "movie":
			movieLibraries = append(movieLibraries, lib)
		case "show":
			tvLibraries = append(tvLibraries, lib)
		}
	}
	movieLibraries = filterExcluded(movieLibraries, utils.StringSet(cfg.MovieLibraryExclude), "movie")
	tvLibraries = filterExcluded(tvLibraries, utils.StringSet(cfg.TVLibraryExclude), "TV")

	if len(movieLibraries) == 0 && !cfg.ProcessTVShows() {
		fmt.Println("[ERROR] No movie library found!")
		os.Exit(1)
	}

	if cfg.ProcessTVShows() && len(tvLibraries) == 0 {
		fmt.Println("[ERROR] No TV show library found!")
		os.Exit(1)
	}

	return movieLibraries, tvLibraries
}

func filterExcluded(libs []plex.Library, exclude map[string]bool, kind string) []plex.Library {
	if len(exclude) == 0 {
		return libs
	}
	kept := libs[:0]
	for _, lib := range libs {
		if exclude[lib.Key] {
			fmt.Printf("[INFO] Excluding %s library: %s (ID: %s)\n", kind, lib.Title, lib.Key)
			continue
		}
		kept = append(kept, lib)
	}
	return kept
}

// findLibraryName returns the library title for the given ID, or the fallback if not found.
func findLibraryName(libraries []plex.Library, id, fallback string) string {
	for _, lib := range libraries {
		if lib.Key == id {
			return lib.Title
		}
	}
	return fallback
}

// forEachLibrary calls fn for the relevant libraries based on config (processAll vs specific ID).
func forEachLibrary(processAll bool, specificID string, libraries []plex.Library, fallbackName string, fn func(id, name string)) {
	if processAll {
		for _, lib := range libraries {
			fn(lib.Key, lib.Title)
		}
	} else if specificID != "" {
		fn(specificID, findLibraryName(libraries, specificID, fallbackName))
	}
}

func displayLibrarySelection(cfg *config.Config, movieLibraries, tvLibraries []plex.Library) {
	if cfg.ProcessMovies() {
		if cfg.MovieProcessAll {
			fmt.Printf("[INFO] Processing all %d movie libraries\n", len(movieLibraries))
		} else if cfg.MovieLibraryID != "" {
			name := findLibraryName(movieLibraries, cfg.MovieLibraryID, "")
			if name == "" {
				fmt.Printf("[ERROR] Movie library with ID %s not found!\n", cfg.MovieLibraryID)
				os.Exit(1)
			}
			fmt.Printf("[INFO] Using specified movie library: %s (ID: %s)\n", name, cfg.MovieLibraryID)
		}
	}
	if cfg.ProcessTVShows() {
		if cfg.TVProcessAll {
			fmt.Printf("[INFO] Processing all %d TV show libraries\n", len(tvLibraries))
		} else if cfg.TVLibraryID != "" {
			name := findLibraryName(tvLibraries, cfg.TVLibraryID, "")
			if name == "" {
				fmt.Printf("[ERROR] TV library with ID %s not found!\n", cfg.TVLibraryID)
				os.Exit(1)
			}
			fmt.Printf("[INFO] Using specified TV library: %s (ID: %s)\n", name, cfg.TVLibraryID)
		} else {
			fmt.Printf("[INFO] Using TV library: %s (ID: %s)\n", tvLibraries[0].Title, tvLibraries[0].Key)
		}
	}
}

func handleRemoveMode(cfg *config.Config, processor *media.Processor, movieLibraries, tvLibraries []plex.Library) {
	displayLibrarySelection(cfg, movieLibraries, tvLibraries)
	fmt.Printf("\n[REMOVE] Starting keyword removal (field: %s, lock: %s)...\n", cfg.UpdateField, cfg.RemoveMode)

	if cfg.ProcessMovies() {
		forEachLibrary(cfg.MovieProcessAll, cfg.MovieLibraryID, movieLibraries, "Movies", func(id, name string) {
			fmt.Printf("[MOVIE] Removing keywords from library: %s (ID: %s)\n", name, id)
			if err := processor.RemoveKeywordsFromItems(id, media.MediaTypeMovie); err != nil {
				fmt.Printf("[ERROR] Error removing keywords from movies: %v\n", err)
			}
		})
	}
	if cfg.ProcessTVShows() {
		forEachLibrary(cfg.TVProcessAll, cfg.TVLibraryID, tvLibraries, "TV Shows", func(id, name string) {
			fmt.Printf("[TV] Removing keywords from library: %s (ID: %s)\n", name, id)
			if err := processor.RemoveKeywordsFromItems(id, media.MediaTypeTV); err != nil {
				fmt.Printf("[ERROR] Error removing keywords from TV shows: %v\n", err)
			}
		})
	}
	fmt.Println("\n[OK] Keyword removal completed. Exiting.")
}

func handleNormalMode(cfg *config.Config, processor *media.Processor, movieLibraries, tvLibraries []plex.Library) {
	displayLibrarySelection(cfg, movieLibraries, tvLibraries)

	scanner := &scanRunner{cfg: cfg, processor: processor, movieLibs: movieLibraries, tvLibs: tvLibraries}

	var webhookServer *webhook.Server
	if cfg.WebhookEnabled {
		webhookServer = webhook.NewServer(cfg, processor, movieLibraries, tvLibraries, scanner)
		if err := webhookServer.Start(); err != nil {
			fmt.Printf("[ERROR] %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("[OK] Webhook server listening on port %d\n", cfg.WebhookPort)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		fmt.Printf("\n[INFO] Received %s, shutting down...\n", sig)
		if webhookServer != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = webhookServer.Stop(ctx)
		}
		os.Exit(0)
	}()

	if cfg.WebhookOnly {
		fmt.Println("[INFO] WEBHOOK_ONLY=true: skipping startup full scan and periodic timer; webhook server is the only trigger")
		select {}
	}

	fmt.Printf("[INFO] Starting periodic processing interval: %v\n", cfg.ProcessTimer)

	scanner.RunAll()

	ticker := time.NewTicker(cfg.ProcessTimer)
	defer ticker.Stop()

	for range ticker.C {
		fmt.Printf("\n[TIMER] Timer triggered - processing at %s\n", time.Now().Format("15:04:05"))
		scanner.RunAll()
	}
}

// scanRunner implements webhook.Scanner. Used by both the periodic timer and
// the /scan HTTP endpoint.
type scanRunner struct {
	cfg       *config.Config
	processor *media.Processor
	movieLibs []plex.Library
	tvLibs    []plex.Library
}

func (r *scanRunner) RunAll() {
	r.processor.ClearCaches()

	if len(r.movieLibs) > 0 {
		forEachLibrary(r.cfg.MovieProcessAll, r.cfg.MovieLibraryID, r.movieLibs, "Movies", func(id, name string) {
			fmt.Printf("[MOVIE] Processing library: %s (ID: %s)\n", name, id)
			if err := r.processor.ProcessAllItems(id, name, media.MediaTypeMovie); err != nil {
				fmt.Printf("[ERROR] Error processing movies: %v\n", err)
			}
		})
	}

	if r.cfg.ProcessTVShows() {
		forEachLibrary(r.cfg.TVProcessAll, r.cfg.TVLibraryID, r.tvLibs, "TV Shows", func(id, name string) {
			fmt.Printf("[TV] Processing TV library: %s (ID: %s)\n", name, id)
			if err := r.processor.ProcessAllItems(id, name, media.MediaTypeTV); err != nil {
				fmt.Printf("[ERROR] Error processing TV shows: %v\n", err)
			}
		})
	}

	if r.cfg.HasExportEnabled() {
		writeExportFiles(r.cfg, r.processor)
	}
}

func (r *scanRunner) RunLibrary(libraryID, libraryName string, mediaType media.MediaType) error {
	r.processor.ClearCaches()
	tag := "[MOVIE]"
	if mediaType == media.MediaTypeTV {
		tag = "[TV]"
	}
	fmt.Printf("%s Processing library: %s (ID: %s)\n", tag, libraryName, libraryID)
	if err := r.processor.ProcessAllItems(libraryID, libraryName, mediaType); err != nil {
		return err
	}
	if r.cfg.HasExportEnabled() {
		writeExportFiles(r.cfg, r.processor)
	}
	return nil
}

func writeExportFiles(cfg *config.Config, processor *media.Processor) {
	fmt.Printf("\n[EXPORT] Writing export files to %s...\n", cfg.ExportLocation)
	exporter := processor.GetExporter()
	if exporter == nil {
		return
	}

	totalSummary, err := exporter.GetExportSummary()
	if err != nil {
		fmt.Printf("[ERROR] Error getting export summary: %v\n", err)
		return
	}

	totalAccumulated := 0
	for label, count := range totalSummary {
		if count > 0 {
			fmt.Printf("  %s: %d total file paths\n", label, count)
		}
		totalAccumulated += count
	}

	if totalAccumulated > 0 {
		fmt.Printf("[INFO] Writing %d total file paths across all libraries...\n", totalAccumulated)
	} else {
		fmt.Printf("[INFO] No matching items found for export labels\n")
	}

	if err := exporter.FlushAll(); err != nil {
		fmt.Printf("[ERROR] Failed to write export files: %v\n", err)
		return
	}

	if cfg.ExportMode == "json" {
		fmt.Printf("[OK] Successfully wrote export data to export.json\n")
	} else {
		fmt.Printf("[OK] Successfully wrote export files to library subdirectories\n")
	}
}
