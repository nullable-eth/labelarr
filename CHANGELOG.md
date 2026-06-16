# Changelog

## [Unreleased]

### Added
- `EXCLUDE_LABELS` environment variable (default empty): comma-separated list of Plex labels that mark items as opted-out of labelarr. Items carrying any of these labels are skipped during both apply and removal passes. Case-insensitive; surrounding whitespace and empty values in the CSV are ignored. Logged at startup when active (`[INFO] EXCLUDE_LABELS active - items tagged with any of [...] will be skipped`) and per skipped item under `VERBOSE_LOGGING=true`.

### Documentation
- README `Library Selection` section now documents `MOVIE_LIBRARY_EXCLUDE` and `TV_LIBRARY_EXCLUDE`, which were added in 1.3.0 but only appeared in the changelog.

## [1.3.2] - 2026-04-21

### Security
- Plex client error messages now redact `X-Plex-Token`, `apikey`, and `api_key` query-string values. Previously a transport failure (e.g. TLS cert rejection) would bubble up Go's `*url.Error`, which embeds the full request URL including the Plex token, leaking it into logs/Loki. All `c.httpClient.Do` calls are now routed through a `safeDo` wrapper that scrubs the error string before returning.

## [1.3.1] - 2026-04-21

### Security
- Plex TLS certificate verification is now enabled by default. Previously, `InsecureSkipVerify` was hardcoded to `true`, which silently trusted any server certificate. Set `PLEX_INSECURE_SKIP_VERIFY=true` to opt back into skip-verify (e.g., for self-signed Plex certs). A `[WARN]` line is logged at startup when the flag is on. Resolves CodeQL `go/disabled-certificate-check`.
- Export paths are now validated to stay within `EXPORT_LOCATION`. Library names that resolve to `.`, `..`, or empty after sanitization are rejected, and a `safeJoin` helper verifies every export path stays inside the export root before writing. Resolves CodeQL `go/path-injection`.
- `.github/workflows/release.yml` now declares a least-privilege top-level `permissions: contents: read`. Per-job elevated permissions on `create-release`, `build-binaries`, and `publish-docker` are unchanged. Resolves CodeQL `actions/missing-workflow-permissions`.

### Added
- `PLEX_INSECURE_SKIP_VERIFY` environment variable (default `false`).

### Changed
- Bumped Docker build base images to `golang:1.26-alpine` (builder) and `alpine:3.22` (runtime) to pick up patched Go stdlib and Alpine packages. Clears 1 CRITICAL + 8 HIGH CVEs surfaced by Trivy on the previous `golang:1.23-alpine` / `alpine:3.21` base (CVE-2025-68121, CVE-2025-58183, CVE-2025-61726/28/29, CVE-2026-25679/32280/32281/32283).
- `go.mod` now requires Go 1.26; `release.yml` `actions/setup-go` pinned to `1.26`.
- `.dockerignore` hardened to exclude `.env*`, `*.pem`, `*.key`, `.claude`, `.github`, `CLAUDE.md`, `PR_DESCRIPTION.md`, `tmp/`.

## [1.3.0] - 2026-04-12

### Added
- `POST /scan` endpoint on the webhook server for manual scan triggers.
  - `POST /scan` runs a full scan across all non-excluded libraries.
  - `POST /scan?library=<id|name>` scans a single library; `library`
    accepts the numeric Plex section ID or a case-insensitive library
    title.
  - Returns `202 Accepted` immediately; the scan runs in the
    background. Returns `409 Conflict` if a scan is already in
    progress, `404 Not Found` if the library param does not match, and
    `405 Method Not Allowed` for non-POST requests.
  - Works in `WEBHOOK_ONLY=true` mode — enables ad-hoc catchup scans
    without toggling environment variables.

## [1.2.3] - 2026-04-12

### Fixed
- Every Plex webhook delivery was being rejected with `HTTP 400` because
  `PlexWebhookPayload.Metadata.GUID` was typed as `string`, while Plex
  sends it as an array of objects (`[{id:"imdb://..."}, ...]`) for
  multi-provider items. `json.Unmarshal` failed before reaching the
  event handler. The unused field has been removed; unknown JSON fields
  are ignored by the decoder, so the shape no longer matters.

## [1.2.2] - 2026-04-12

### Changed
- Webhook 400 responses now log the specific failure reason (parse error,
  missing payload field, or JSON unmarshal error) along with
  `Content-Type`, form/file part keys, and a payload snippet. Previously
  all three paths returned 400 with no log, making Plex delivery failures
  invisible.

## [1.2.1] - 2026-04-12

### Added
- `MOVIE_LIBRARY_EXCLUDE` / `TV_LIBRARY_EXCLUDE`: comma-separated library
  IDs to skip when `*_PROCESS_ALL=true`. Excluded libraries are filtered
  from both timer-driven processing and webhook routing.
- `WEBHOOK_ONLY=true`: skips the startup full scan and the periodic timer
  entirely, leaving the webhook server as the only trigger. Requires
  `WEBHOOK_ENABLED=true`.

### Fixed
- Webhook items are no longer silently dropped when a full library scan is
  in progress. `ProcessSingleItem` now waits for the per-library slot to
  free up (polling every 5s, bounded to a 2-hour deadline) instead of
  logging a fake "queuing for next cycle" and returning early.

## [1.2.0] - 2026-04-10

### Added

#### Plex Webhook Support
- Webhook listener for real-time processing (WEBHOOK_ENABLED, WEBHOOK_PORT)
- Handles library.new events (other Plex events are accepted but ignored to avoid reprocessing items that don't need it)
- Configurable debounce window (WEBHOOK_DEBOUNCE, default 30s)
- Prevents concurrent processing of the same library
- Health check endpoint at /health
- Runs alongside the existing timer

#### Keyword Prefix
- KEYWORD_PREFIX env var to prepend text to keywords (e.g. "- ")
- Useful when UPDATE_FIELD=genre to separate TMDb keywords from real genres

#### Batch Processing
- BATCH_SIZE (default 100) and BATCH_DELAY (default 10s) env vars
- Prevents API flooding on large libraries (4000+ items)
- ITEM_DELAY (default 500ms) controls per-item pacing

#### Version Tracking
- Version constant in internal/version/version.go
- Logs version on startup

### Changed
- Removed all emoji from log output; replaced with bracketed tags
- Extracted Clients struct for processor initialization
- Added keyword cache by TMDb ID to avoid redundant API calls
- Eliminated redundant Plex API call after keyword sync for export

## 2025-07-05

### Added

#### Radarr/Sonarr Integration
- Radarr API client (internal/radarr/) -- movie lookup by title, year, TMDb ID, IMDb ID, file path
- Sonarr API client (internal/sonarr/) -- series lookup by title, year, TMDb ID, TVDb ID, IMDb ID, file path
- USE_RADARR, USE_SONARR, RADARR_URL, RADARR_API_KEY, SONARR_URL, SONARR_API_KEY env vars
- TMDb ID extraction chain: Plex metadata -> Radarr/Sonarr -> file path regex
- Connection testing on startup for all enabled services

#### Verbose Logging
- VERBOSE_LOGGING env var (default false)
- Shows TMDb ID lookup source, Plex GUIDs, matching attempts
- Progress percentage for libraries over 100 items

#### Persistent Storage
- JSON file storage for processed items (DATA_DIR env var)
- Tracks rating key, TMDb ID, update field, last processed time
- Skips already-processed items unless FORCE_UPDATE=true
- Runs in ephemeral mode when DATA_DIR is not set

#### Keyword Normalization
- Pattern-based normalization: sci-fi -> Sci-Fi, romcom -> Romantic Comedy
- Acronym detection (FBI, CIA, DEA, etc.)
- Century formatting (5th century bc -> 5th Century BC)
- City/state, relationship, and credit stinger patterns
- 90+ test cases
- Duplicate cleaning: removes old unnormalized keywords when normalized versions are added

#### Force Update Mode
- FORCE_UPDATE env var (default false)
- Reprocesses all items regardless of storage state

#### Export Functionality
- EXPORT_LABELS, EXPORT_LOCATION, EXPORT_MODE (txt/json) env vars
- Generates file lists per label per library
- JSON mode outputs a single structured export.json
- TXT mode creates per-library subdirectories with summary.txt

### Changed
- NewProcessor accepts optional Radarr/Sonarr clients and returns error
- TMDb client normalizes keywords before returning them
- Removal delay reduced from 500ms to configurable ITEM_DELAY

### Technical Notes
- Radarr/Sonarr use API v3
- All new features are optional and backward compatible
- No breaking changes to existing configuration
