# Labelarr 🎬🏷️

**Automatically sync TMDb movie keywords as Plex labels**  
A lightweight Go application that bridges your Plex movie library with The Movie Database, adding relevant keywords as searchable labels.

## What it does

Labelarr continuously monitors your Plex movie library and automatically:

- 🔍 Detects TMDb IDs from Plex metadata or file paths (e.g. `{tmdb-12345}`, `[tmdb:12345]`, `(tmdb;12345)`, etc.)
- 📥 Fetches movie keywords from TMDb API
- 🏷️ Adds keywords as Plex labels (preserves existing labels)
- 📊 Tracks processed movies to avoid duplicates
- ⏰ Runs on a configurable timer (default: 5 minutes)

## Quick Start

```bash
docker run -d --name labelarr \
  -e PLEX_SERVER=localhost \
  -e PLEX_PORT=32400 \
  -e PLEX_REQUIRES_HTTPS=true \
  -e PLEX_TOKEN=your_plex_token_here \
  -e TMDB_READ_ACCESS_TOKEN=your_tmdb_read_access_token \
  nullableeth/labelarr:latest
```

## Required Environment Variables

| Variable                  | Description                        | How to Get                                                                 |
|---------------------------|------------------------------------|----------------------------------------------------------------------------|
| `PLEX_TOKEN`              | Plex authentication token          | Plex Web App → F12 → Network tab → Look for `X-Plex-Token` in headers      |
| `TMDB_READ_ACCESS_TOKEN`  | TMDb Read Access API Token         | [TMDb API Settings](https://www.themoviedb.org/settings/api)               |
| `PLEX_SERVER`             | Your Plex server IP/hostname       |                                                                            |
| `PLEX_PORT`               | Your Plex server port              |                                                                            |
| `PLEX_REQUIRES_HTTPS`     | Use HTTPS for Plex connection      | `true`/`false`                                                             |
| `PROCESS_TIMER`           | How often to scan (e.g., `5m`)     | `5m`, `10m`, `1h`, etc.                                                    |
| `LIBRARY_ID`              | Plex library ID (auto-detected if not set) | See Library Selection Logic below                                          |
| `PROCESS_ALL_MOVIE_LIBRARIES` | Process all movie libraries (set to `true` to enable) | `false` |

## Docker Compose Example

```yaml
version: '3.8'
services:
  labelarr:
    image: nullableeth/labelarr:latest
    container_name: labelarr
    restart: unless-stopped
    environment:
      - PLEX_SERVER=localhost
      - PLEX_PORT=32400
      - PLEX_REQUIRES_HTTPS=true
      - PLEX_TOKEN=your_plex_token_here
      - TMDB_READ_ACCESS_TOKEN=your_tmdb_read_access_token
      - PROCESS_TIMER=5m
```

## 🐳 Docker Compose: Ensuring Labelarr Waits for Plex

To prevent Labelarr from logging errors when Plex is not yet ready, use Docker Compose's `depends_on` with `condition: service_healthy` and add a healthcheck to your Plex service. This ensures Labelarr only starts after Plex is healthy.

Example:

```yaml
services:
  plex:
    image: plexinc/pms-docker:latest
    container_name: plex
    # ... other config ...
    healthcheck:
      test: curl --connect-timeout 15 --silent --show-error --fail http://localhost:32400/identity
      interval: 1m00s
      timeout: 15s
      retries: 3
      start_period: 1m00s
  labelarr:
    image: nullableeth/labelarr:latest
    container_name: labelarr
    depends_on:
      plex:
        condition: service_healthy
    # ... other config ...
```

This setup ensures Labelarr only starts after Plex is healthy, avoiding initial connection errors.

## TMDb ID Detection

Works with multiple sources:

- **Plex Metadata**: Standard TMDb agent data
- **File Names**: `/Movies/Movie (2023) {tmdb-12345}/movie.mkv`, `/Movies/Movie [tmdb:12345]/movie.mkv`
- **Directory Names**: `/Movies/Movie {tmdb-12345}/`, `/Movies/Movie [tmdb:12345]/`

Supports various separators and brackets: `{tmdb-12345}`, `[tmdb:12345]`, `(tmdb;12345)`, etc.

## Features

✅ **Non-destructive**: Never removes existing labels  
✅ **Smart detection**: Multiple TMDb ID sources  
✅ **Progress tracking**: Remembers processed movies  
✅ **Lightweight**: ~10MB Alpine-based container  
✅ **Secure**: Runs as non-root user  
✅ **Auto-retry**: Handles API rate limits gracefully  
✅ **Protocol flexibility**: Supports both HTTP and HTTPS Plex connections  

## Getting API Keys

### Plex Token

1. Open Plex Web App in browser
2. Press F12 → Network tab
3. Refresh page
4. Find any request with `X-Plex-Token` header
5. Copy the token value

### TMDb Read Access Token

1. Visit [TMDb API Settings](https://www.themoviedb.org/settings/api)
2. Create account if needed
3. Generate API key
4. Use the **(Read Access Token)** (not the v3 API key)

## 🏷️ Library Selection Logic

- **Default Behavior:**
  - If you do **not** specify a `LIBRARY_ID`, the application will automatically select the **first movie library** it finds on your Plex server.
- **Specifying a Library:**
  - You can specify a particular movie library by setting the `LIBRARY_ID` environment variable.
  - To find your library's ID, open your Plex web app, click on the desired movie library, and look for `source=` in the URL. For example:
    - `https://app.plex.tv/desktop/#!/media/xxxx/com.plexapp.plugins.library?source=1`
    - Here, the library ID is `1`.
- **Processing All Movie Libraries:**
  - If you set `PROCESS_ALL_MOVIE_LIBRARIES=true`, the application will process **all** movie libraries found on your Plex server, regardless of the `LIBRARY_ID` setting.

## Logs & Monitoring

View logs: `docker logs labelarr`

The application provides detailed logging including:

- Movie processing progress
- TMDb ID detection results  
- Label sync status
- API errors and retries

## Support

- **GitHub**: [https://github.com/nullable-eth/Labelarr](https://github.com/nullable-eth/Labelarr)
- **Issues**: Report bugs and feature requests
- **Logs**: Check container logs for troubleshooting

---

**Tags**: plex, tmdb, automation, movies, labels, docker, go, selfhosted

---
