# Changelog

## [Unreleased] - 2025-07-05

### Added

#### Radarr/Sonarr Integration

- ✅ Created Radarr API client module (`internal/radarr/`) with full API support
  - Movie search by title, year, TMDb ID, IMDb ID, and file path
  - Automatic TMDb ID extraction from Radarr database
  - Connection testing and system status endpoints
- ✅ Created Sonarr API client module (`internal/sonarr/`) with full API support

  - TV series search by title, year, TMDb ID, TVDb ID, IMDb ID, and file path
  - Episode fetching and file path matching
  - Connection testing and system status endpoints

- ✅ Updated configuration system to support Radarr/Sonarr

  - Added `USE_RADARR` and `USE_SONARR` environment variables
  - Added `RADARR_URL`, `RADARR_API_KEY`, `SONARR_URL`, `SONARR_API_KEY` configuration
  - Added validation for Radarr/Sonarr settings when enabled

- ✅ Enhanced TMDb ID extraction to use multiple sources

  - Primary: Plex metadata (existing functionality preserved)
  - Secondary: Radarr/Sonarr API matching (new)
  - Fallback: File path regex matching (existing functionality preserved)
  - Added source tracking to show where TMDb ID was found

- ✅ Updated media processor to integrate with Radarr/Sonarr

  - Modified `extractMovieTMDbID` to query Radarr when enabled
  - Modified `extractTVShowTMDbID` to query Sonarr when enabled
  - Added multiple matching strategies: title/year, IMDb ID, TVDb ID, file path

- ✅ Updated main application to initialize Radarr/Sonarr clients

  - Added connection testing on startup
  - Graceful handling when Radarr/Sonarr are not configured

- ✅ Created comprehensive docker-compose.yml example
  - Includes all existing configuration options
  - Added Radarr/Sonarr configuration examples with defaults

#### Verbose Logging & Debugging

- ✅ Added verbose logging feature

  - New `VERBOSE_LOGGING` environment variable (default: false)
  - Shows detailed TMDb ID lookup process for each item
  - Displays all available Plex GUIDs (IMDb, TMDb, TVDb)
  - Shows Radarr lookup attempts with title, file path, and IMDb ID matching
  - Shows Sonarr lookup attempts with title, TVDb ID, IMDb ID, and file path matching
  - Indicates source of successful TMDb ID matches
  - Helps troubleshoot matching issues

- ✅ Added progress tracking for large libraries

  - Shows percentage progress for libraries with >100 items
  - Displays current processing status
  - Shows summary of skipped items in verbose mode

- ✅ Enhanced label/genre application logging
  - Shows when keywords are being applied to Plex
  - Displays Plex API call timing in verbose mode
  - Shows current and new keywords being merged
  - Confirms successful application to Plex

#### Batch Processing

- ✅ Added batch processing system to prevent API overwhelming

  - New `BATCH_SIZE` environment variable (default: 100 items per batch)
  - New `BATCH_DELAY_SECONDS` environment variable (default: 10 seconds between batches)
  - **Fully backward compatible** - existing configurations work without changes
  - Processes large libraries in manageable chunks instead of continuous loop
  - Prevents Radarr/Sonarr SQLite database locks and API flooding
  - Reduces memory usage and prevents container crashes
  - Configurable batch size and delay for different system capabilities

- ✅ Enhanced processing flow for large libraries

  - Sequential batch processing with progress reporting
  - Batch-specific progress indicators (e.g., "Processing batch 5/42")
  - Configurable delays between batches to allow system recovery
  - Maintains existing per-item processing logic within batches
  - No delay after final batch for efficiency
  - **Smart messaging** - verbose batch logs only for large libraries or custom settings

- ✅ Added configuration validation and guidelines
  - Validates positive batch sizes and non-negative delays
  - Comprehensive documentation for different library sizes
  - Performance tuning recommendations based on library size
  - Troubleshooting guide for optimal batch configuration
  - **Zero migration required** - defaults provide immediate stability improvements

#### Persistent Storage

- ✅ Added persistent storage for processed items
  - Prevents reprocessing items after container restarts
  - JSON file-based storage with atomic writes
  - Tracks which field (label/genre) was updated for each item
  - Configurable data directory via DATA_DIR environment variable
  - Docker volume support for data persistence
  - Storage directory defaults to `/data` in container

#### Error Handling & Connection Testing

- ✅ Added TMDb API connection testing on startup

  - Validates API token before processing begins
  - Provides clear error messages for authentication failures
  - Shows detailed error responses for debugging

- ✅ Improved error handling throughout
  - Better error messages for TMDb API failures
  - Clear indication of authentication vs other errors
  - Verbose mode shows why items are skipped

### Changed

- Modified `NewProcessor` to accept optional Radarr/Sonarr clients and return error
- Enhanced TMDb ID detection to show source (Plex metadata, Radarr, Sonarr, or file path)
- Processor initialization now includes persistent storage setup
- Main application now tests all API connections on startup

### Documentation

- ✅ Updated README.md with comprehensive documentation

  - Added Radarr/Sonarr Integration section with benefits and configuration
  - Added Verbose Logging section with examples
  - Updated environment variables documentation
  - Added persistent storage information
  - Updated docker-compose examples

- ✅ Created detailed CHANGELOG.md
  - Comprehensive list of all changes
  - Organized by feature area
  - Technical implementation details

#### Keyword Normalization

- ✅ Added intelligent keyword normalization feature

  - Automatically normalizes TMDb keywords for consistent formatting
  - Pattern-based recognition for dynamic handling without hardcoding
  - Smart title casing with proper article and preposition handling
  - Automatic duplicate removal after normalization

- ✅ Pattern Recognition Features

  - **Critical Replacements**: Known abbreviations (sci-fi → Sci-Fi, romcom → Romantic Comedy)
  - **Acronym Detection**: Automatically uppercases known acronyms (FBI, CIA, DEA, etc.)
  - **Agency Patterns**: Detects agency roles (dea agent → DEA Agent)
  - **Parenthetical Acronyms**: Handles acronyms in parentheses (central intelligence agency (cia) → Central Intelligence Agency (CIA))
  - **Century Patterns**: Properly formats centuries (5th century bc → 5th Century BC)
  - **City/State Patterns**: Handles location formatting (san francisco, california → San Francisco, California)
  - **Relationship Patterns**: Adds "Relationship" where appropriate (father daughter → Father Daughter Relationship)
  - **Credit Stinger Terms**: Expands compound terms (duringcreditsstinger → During Credits Stinger)

- ✅ Added comprehensive test suite

  - 90+ test cases covering various normalization scenarios
  - Tests for edge cases, mixed case preservation, and pattern matching
  - Ensures consistent behavior across different keyword types

- ✅ Smart duplicate cleaning functionality
  - Automatically removes old unnormalized keywords when adding normalized versions
  - Preserves manually set keywords in Plex
  - Prevents accumulation of duplicate keywords (e.g., both "sci-fi" and "Sci-Fi")
  - Shows cleaning activity in verbose logging mode

#### Force Update Mode

- ✅ Added force update functionality
  - New `FORCE_UPDATE` environment variable (default: false)
  - Reprocesses all items regardless of previous processing status
  - Useful for applying keyword normalization to existing libraries
  - Shows clear indication when force update mode is active
  - Bypasses both storage checks and "already has keywords" logic

### Changed

- Modified `NewProcessor` to accept optional Radarr/Sonarr clients and return error
- Enhanced TMDb ID detection to show source (Plex metadata, Radarr, Sonarr, or file path)
- Processor initialization now includes persistent storage setup
- Main application now tests all API connections on startup
- TMDb client now normalizes all keywords before returning them
- Updated keyword display to show normalization in verbose mode
- Enhanced keyword synchronization with smart duplicate cleaning
- Force update mode bypasses all previous processing checks

### Documentation

- ✅ Updated README.md with comprehensive documentation

  - Added Radarr/Sonarr Integration section with benefits and configuration
  - Added Verbose Logging section with examples
  - Added Keyword Normalization section with pattern examples
  - Added Force Update Mode section with use cases and examples
  - Added Smart Duplicate Cleaning documentation
  - Updated environment variables documentation
  - Added persistent storage information
  - Updated docker-compose examples

- ✅ Created detailed CHANGELOG.md
  - Comprehensive list of all changes
  - Organized by feature area
  - Technical implementation details

### Technical Details

- Radarr/Sonarr clients use API v3 endpoints
- Implemented robust error handling and fallback mechanisms
- No breaking changes - Radarr/Sonarr integration is fully optional
- Maintains backward compatibility with existing file path matching
- Verbose logging provides detailed insights without affecting normal operation
- Keyword normalization uses regex patterns for scalability
- All features are designed to be non-breaking and backward compatible
