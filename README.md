# Labelarr

[![GitHub Release](https://img.shields.io/github/v/release/nullable-eth/labelarr?style=flat-square)](https://github.com/nullable-eth/labelarr/releases/latest)
[![Docker Image](https://img.shields.io/badge/docker-ghcr.io-blue?style=flat-square&logo=docker)](https://github.com/nullable-eth/labelarr/pkgs/container/labelarr)
[![Go Version](https://img.shields.io/github/go-mod/go-version/nullable-eth/labelarr?style=flat-square)](https://golang.org/)

Syncs TMDb keywords to Plex as labels or genres. Runs as a Docker container on a timer, or reacts to Plex webhooks in real time.

## Upgrade notes (breaking changes)

If you're upgrading from an earlier release, two configuration changes are worth a quick check before you redeploy:

- **Plex TLS certificate verification is now enabled by default.** Previously, Labelarr silently trusted any Plex certificate. If your Plex server uses a self-signed cert (the default for most homelab installs and Plex's built-in `*.plex.direct` certs accessed by raw IP), set `PLEX_INSECURE_SKIP_VERIFY=true` to restore the previous behavior. A `[WARN]` line is logged at startup whenever the flag is on. See [CHANGELOG 1.3.1](CHANGELOG.md) for the full rationale (CodeQL `go/disabled-certificate-check`).
- **`BATCH_DELAY_SECONDS` has been renamed to `BATCH_DELAY`** and now accepts Go duration strings instead of an integer. Migrate `BATCH_DELAY_SECONDS=10` to `BATCH_DELAY=10s`. Similarly, `ITEM_DELAY` is now a duration (default `500ms`). The integer-seconds form is no longer accepted.

## Table of Contents

- [Upgrade notes (breaking changes)](#upgrade-notes-breaking-changes)
- [Quick Start](#quick-start)
- [How It Works](#how-it-works)
- [Environment Variables](#environment-variables)
- [Radarr/Sonarr Integration](#radarrsonarr-integration)
- [Webhook Support](#webhook-support)
- [Batch Processing](#batch-processing)
- [Keyword Prefix](#keyword-prefix)
- [Keyword Normalization](#keyword-normalization)
- [Export Functionality](#export-functionality)
- [TMDb ID Detection](#tmdb-id-detection)
- [Removing Keywords](#removing-keywords)
- [Field Locking](#field-locking)
- [Force Update Mode](#force-update-mode)
- [Verbose Logging](#verbose-logging)
- [Persistent Storage](#persistent-storage)
- [Getting API Keys](#getting-api-keys)
- [Troubleshooting](#troubleshooting)
- [Local Development](#local-development)

## Quick Start

```yaml
services:
  labelarr:
    image: ghcr.io/nullable-eth/labelarr:latest
    container_name: labelarr
    restart: unless-stopped
    volumes:
      - ./labelarr-data:/data
    environment:
      - PLEX_TOKEN=your_plex_token_here
      - TMDB_READ_ACCESS_TOKEN=your_tmdb_read_access_token
      - PLEX_SERVER=plex
      - PLEX_PORT=32400
      - PLEX_REQUIRES_HTTPS=true
      - MOVIE_PROCESS_ALL=true
      - TV_PROCESS_ALL=true
```

Run `docker-compose up -d`. Labelarr processes your libraries immediately on startup, then repeats every hour.

![Labels](example/labels.png) ![Dynamic Filters](example/dynamic_filter.png) ![Filter](example/filter.png)

## How It Works

1. Fetches all movies/shows from your Plex libraries
2. Finds the TMDb ID for each item (from Plex metadata, Radarr/Sonarr, or file paths)
3. Pulls keywords from the TMDb API
4. Normalizes keyword formatting (capitalization, acronyms, known patterns)
5. Adds keywords as Plex labels or genres -- never removes existing values
6. Tracks what has been processed to skip it next time

Runs on a configurable timer (default 1h). With webhooks enabled, also processes immediately when Plex adds new media.

## Environment Variables

### Required

| Variable | Description |
|----------|-------------|
| `PLEX_TOKEN` | Plex authentication token |
| `TMDB_READ_ACCESS_TOKEN` | TMDb API read access token |
| `PLEX_SERVER` | Plex server hostname or IP |
| `PLEX_PORT` | Plex server port (usually 32400) |

### Library Selection

Pick one approach per media type:

| Variable | Description |
|----------|-------------|
| `MOVIE_PROCESS_ALL=true` | Process all movie libraries |
| `MOVIE_LIBRARY_ID=1` | Process a specific movie library by ID |
| `TV_PROCESS_ALL=true` | Process all TV show libraries |
| `TV_LIBRARY_ID=2` | Process a specific TV library by ID |

Optionally narrow what gets processed within those libraries:

| Variable | Default | Description |
|----------|---------|-------------|
| `MOVIE_LIBRARY_EXCLUDE` | (empty) | Comma-separated Plex library **IDs** to skip when `MOVIE_PROCESS_ALL=true` (e.g. `MOVIE_LIBRARY_EXCLUDE=8,12`). Useful for keeping a "Home Videos" library out of the scan. |
| `TV_LIBRARY_EXCLUDE` | (empty) | Same as above for TV libraries. |
| `EXCLUDE_LABELS` | (empty) | Comma-separated **per-item opt-out** label list. Any Plex item carrying one of these labels is skipped on both apply and removal paths. Case-insensitive. Example: `EXCLUDE_LABELS=labelarr:skip,home video`. Tag the offending items in Plex (Edit -> Tags -> Labels) and labelarr will leave them alone. |

### Optional

| Variable | Default | Description |
|----------|---------|-------------|
| `PLEX_REQUIRES_HTTPS` | `false` | Use HTTPS for Plex connection |
| `PLEX_INSECURE_SKIP_VERIFY` | `false` | Skip TLS certificate verification for Plex. Only takes effect when `PLEX_REQUIRES_HTTPS=true`. Enable only for self-signed certs; a `[WARN]` line is logged at startup. |
| `UPDATE_FIELD` | `label` | Field to update: `label` or `genre` |
| `PROCESS_TIMER` | `1h` | How often to run (e.g. `30m`, `2h`, `24h`) |
| `VERBOSE_LOGGING` | `false` | Show detailed lookup and matching info |
| `DATA_DIR` | _(none)_ | Directory for persistent storage; ephemeral if unset |
| `FORCE_UPDATE` | `false` | Reprocess all items regardless of storage state |
| `REMOVE` | _(none)_ | Removal mode: `lock` or `unlock` (runs once and exits) |

### Batch Processing

| Variable | Default | Description |
|----------|---------|-------------|
| `BATCH_SIZE` | `100` | Items per batch |
| `BATCH_DELAY` | `10s` | Pause between batches |
| `ITEM_DELAY` | `500ms` | Pause between individual items |

### Keyword Prefix

| Variable | Default | Description |
|----------|---------|-------------|
| `KEYWORD_PREFIX` | _(none)_ | String prepended to each keyword (e.g. `"- "`) |

### Webhook

| Variable | Default | Description |
|----------|---------|-------------|
| `WEBHOOK_ENABLED` | `false` | Start the webhook HTTP server |
| `WEBHOOK_PORT` | `9090` | Port for the webhook listener |
| `WEBHOOK_DEBOUNCE` | `30s` | Debounce window for rapid events |

### Radarr/Sonarr

| Variable | Default | Description |
|----------|---------|-------------|
| `USE_RADARR` | `false` | Enable Radarr integration |
| `RADARR_URL` | _(none)_ | Radarr base URL (e.g. `http://radarr:7878`) |
| `RADARR_API_KEY` | _(none)_ | Radarr API key |
| `USE_SONARR` | `false` | Enable Sonarr integration |
| `SONARR_URL` | _(none)_ | Sonarr base URL (e.g. `http://sonarr:8989`) |
| `SONARR_API_KEY` | _(none)_ | Sonarr API key |

### Export

| Variable | Default | Description |
|----------|---------|-------------|
| `EXPORT_LABELS` | _(none)_ | Comma-separated labels to export file paths for |
| `EXPORT_LOCATION` | _(none)_ | Directory for export output |
| `EXPORT_MODE` | `txt` | Export format: `txt` or `json` |

## Radarr/Sonarr Integration

If your file paths don't contain TMDb IDs, Labelarr can look them up through Radarr and Sonarr's APIs. The lookup chain is:

1. Plex metadata (fastest)
2. Radarr/Sonarr API (title/year match, then IMDb/TVDb ID, then file path)
3. File path regex (fallback)

This means you don't need to rename any files. Enable it by setting `USE_RADARR=true` and/or `USE_SONARR=true` with the corresponding URL and API key.

File path detection is faster than API calls. If your filenames already include TMDb IDs (e.g. `{tmdb-603}`), you don't need this.

API keys: Radarr/Sonarr Settings > General > Security > API Key.

## Webhook Support

**Requires Plex Pass.** Instead of waiting for the next timer tick, Labelarr can react to Plex webhook events immediately.

```yaml
environment:
  - WEBHOOK_ENABLED=true
  - WEBHOOK_PORT=9090
  - WEBHOOK_DEBOUNCE=30s
ports:
  - "9090:9090"
```

Configure Plex to send webhooks to `http://labelarr:9090/webhook` (Settings > Webhooks in Plex). Labelarr listens for `library.new` events. Other Plex webhook events (`library.on.deck`, `media.play`, etc.) are accepted but intentionally ignored, since they fire repeatedly for items that don't need to be reprocessed.

When multiple events arrive for the same library in quick succession (common during bulk imports), the debounce window coalesces them into a single processing run.

The webhook server runs alongside the existing timer. Both can be active at the same time.

A health check is available at `/health`.

### Manual Scan Trigger

`POST /scan` on the webhook server kicks off a scan cycle without waiting for the timer. Useful when operating in `WEBHOOK_ONLY=true` mode or after extended downtime.

```bash
# Full scan of all non-excluded libraries
curl -X POST http://labelarr:9090/scan

# Scan a single library by Plex section ID
curl -X POST "http://labelarr:9090/scan?library=22"

# Scan a single library by name (case-insensitive)
curl -X POST "http://labelarr:9090/scan?library=Movies"
```

Responses:
- `202 Accepted` — scan started in the background
- `409 Conflict` — a scan is already in progress
- `404 Not Found` — `library` param did not match any configured library
- `405 Method Not Allowed` — non-POST request

> **Network exposure note:** The webhook server has no built-in authentication. `POST /scan` triggers potentially long-running work, and `POST /webhook` accepts any well-formed Plex payload. Bind the port to a trusted network (e.g. a docker bridge with Plex, or behind a reverse proxy that does auth) -- don't expose it to the open internet.

## Batch Processing

Large libraries (4000+ items) can overwhelm Radarr/Sonarr APIs with thousands of requests. Batch processing breaks the work into chunks with pauses between them.

```yaml
environment:
  - BATCH_SIZE=100
  - BATCH_DELAY=10s
  - ITEM_DELAY=500ms
```

With 4000 items and a batch size of 100, Labelarr processes 100 items, pauses 10 seconds, processes the next 100, and so on. The per-item delay (default 500ms) paces individual API calls within each batch.

## Keyword Prefix

When using `UPDATE_FIELD=genre`, TMDb keywords get mixed in with real Plex genres in the filter dropdown. A prefix separates them visually:

```yaml
environment:
  - UPDATE_FIELD=genre
  - KEYWORD_PREFIX="- "
```

This turns `Sci-Fi` into `- Sci-Fi` in the genre list, so real genres sort to the top and keyword-derived genres cluster at the bottom.

The prefix is applied consistently during both add and remove operations.

## Keyword Normalization

TMDb keywords come in inconsistent formats. Labelarr normalizes them before applying:

- Title casing with proper article/preposition handling
- Acronym detection: `fbi` -> `FBI`, `cia` -> `CIA`
- Known replacements: `sci-fi` / `scifi` / `sci fi` -> `Sci-Fi`, `romcom` -> `Romantic Comedy`
- Relationship patterns: `father daughter` -> `Father Daughter Relationship`
- Century formatting: `5th century bc` -> `5th Century BC`
- Location formatting: `san francisco, california` -> `San Francisco, California`
- Credit stingers: `duringcreditsstinger` -> `During Credits Stinger`

When a normalized keyword replaces an old unnormalized version, the old one is automatically removed from Plex.

90+ test cases cover the normalization rules.

## Export Functionality

Generate file path lists for media matching specific labels. Useful for syncing specific genres to other devices or creating targeted backups.

```yaml
environment:
  - EXPORT_LABELS=action,comedy,thriller
  - EXPORT_LOCATION=/data/exports
  - EXPORT_MODE=txt
volumes:
  - ./exports:/data/exports
```

### Text mode (default)

Creates per-library subdirectories with one file per label:

```
/data/exports/
  summary.txt
  Movies/
    action.txt
    comedy.txt
  TV Shows/
    action.txt
    comedy.txt
```

Each file lists the full file paths of matching media.

### JSON mode

Creates a single `export.json` with structured data including file sizes and statistics.

Label matching is case-insensitive. Items with multiple matching labels appear in each corresponding file. Exported paths reflect Plex's internal filesystem, so you may need to translate container paths to host paths.

## TMDb ID Detection

Labelarr looks for TMDb IDs in file and folder names using a flexible regex. All of these work:

```
/movies/The Matrix (1999) {tmdb-603}/file.mkv
/movies/Inception (2010) [tmdb:27205]/file.mkv
/movies/Avatar (2009) tmdb19995/file.mkv
/movies/Interstellar (2014) (tmdb=157336)/file.mkv
/movies/The Dark Knight (2008) TMDB_155/file.mkv
```

Separators (`-`, `:`, `_`, `=`, space) and bracket styles (`{}`, `[]`, `()`) all work. Case-insensitive.

Will not match: `mytmdb12345` (preceded by letters), `tmdb` (no digits), `tmdb12345abc` (followed by letters).

### Radarr naming format

To include TMDb IDs in Radarr-managed files, set the folder format to:

```
{Movie CleanTitle} ({Release Year}) {tmdb-{TmdbId}}
```

For existing libraries, use Radarr's mass rename feature to apply the new format.

## Removing Keywords

`REMOVE=lock` or `REMOVE=unlock` runs a single pass that removes TMDb keywords from the configured field, then exits.

- `lock`: removes keywords, keeps the field locked (Plex can't overwrite)
- `unlock`: removes keywords, unlocks the field (Plex can refresh it)

Only TMDb-sourced keywords are removed. Custom labels you added manually are preserved.

```bash
docker run --rm \
  -e PLEX_TOKEN=... -e TMDB_READ_ACCESS_TOKEN=... \
  -e REMOVE=lock -e UPDATE_FIELD=label \
  -e MOVIE_PROCESS_ALL=true \
  ghcr.io/nullable-eth/labelarr:latest
```

## Field Locking

Labelarr locks the label/genre field after writing to prevent Plex from overwriting keywords during metadata refreshes. Locked fields show a lock icon in the Plex UI.

You can still edit locked fields manually in Plex. External tools (including Labelarr) can also modify them.

![Example of locked genre field](example/genre.png)

## Force Update Mode

Set `FORCE_UPDATE=true` to reprocess every item regardless of whether it was already processed. Useful after:

- Enabling keyword normalization on an existing library
- Switching between label and genre modes
- Wanting to refresh all keywords from TMDb

This bypasses both the storage check and the "already has all keywords" check.

## Verbose Logging

`VERBOSE_LOGGING=true` shows the full TMDb ID lookup chain for each item: which Plex GUIDs are available, Radarr/Sonarr lookup attempts, file path matching, and the source of the final match.

Useful for debugging why specific items aren't being matched.

## Persistent Storage

When `DATA_DIR` is set (e.g. `/data`), Labelarr saves processed items to a JSON file so it can skip them on restart. Without `DATA_DIR`, it runs in ephemeral mode and reprocesses everything each cycle.

Mount a volume to persist across container restarts:

```yaml
volumes:
  - ./labelarr-data:/data
environment:
  - DATA_DIR=/data
```

## Getting API Keys

**Plex Token:** Open Plex Web, press F12, go to Network tab, refresh the page, and look for `X-Plex-Token` in any request header.

**TMDb:** Create an account at [themoviedb.org](https://www.themoviedb.org/settings/api) and generate a Read Access Token.

**Radarr/Sonarr:** Settings > General > Security > API Key.

## Troubleshooting

**401 from Plex** -- Check your token. Try `PLEX_REQUIRES_HTTPS=false` for local servers.

**401 from TMDb** -- Make sure you're using the Read Access Token, not the API key.

**No TMDb ID found** -- Enable `VERBOSE_LOGGING=true` to see where the lookup fails. Either add TMDb IDs to your file paths, enable Radarr/Sonarr integration, or make sure Plex is using the TMDb agent.

**Container permission errors** -- If you see "mkdir /data: permission denied", either set `DATA_DIR` to a writable path with a mounted volume, or leave `DATA_DIR` unset to run in ephemeral mode.

**Large library crashes** -- Set `BATCH_SIZE` and `BATCH_DELAY` to reduce API pressure. The defaults (100 items, 10s pause) work for most setups.

## Local Development

```bash
git clone https://github.com/nullable-eth/labelarr.git
cd labelarr
go mod tidy
go build -o labelarr ./cmd/labelarr

# Set required env vars and run
./labelarr
```

Build for Docker: `CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o labelarr ./cmd/labelarr`

Run tests: `go test ./...`

## License

MIT. See [LICENSE](LICENSE).
