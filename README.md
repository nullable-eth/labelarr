# Labelarr 🎬📺🏷️

[![GitHub Release](https://img.shields.io/github/v/release/nullable-eth/labelarr?style=flat-square)](https://github.com/nullable-eth/labelarr/releases/latest)
[![Docker Image](https://img.shields.io/badge/docker-ghcr.io-blue?style=flat-square&logo=docker)](https://github.com/nullable-eth/labelarr/pkgs/container/labelarr)
[![Go Version](https://img.shields.io/github/go-mod/go-version/nullable-eth/labelarr?style=flat-square)](https://golang.org/)
[![GitHub Actions](https://img.shields.io/github/actions/workflow/status/nullable-eth/labelarr/docker-publish.yml?branch=main&style=flat-square)](https://github.com/nullable-eth/labelarr/actions)

**Automatically sync TMDb keywords as Plex labels or genres for movies and TV shows**  
Lightweight Docker container that bridges Plex with The Movie Database, adding searchable keywords to your media.

## 🚀 Quick Start

### Docker Compose (Recommended)

```yaml
version: '3.8'

services:
  labelarr:
    image: ghcr.io/nullable-eth/labelarr:latest
    container_name: labelarr
    restart: unless-stopped
    environment:
      # Required - Get from Plex Web (F12 → Network → X-Plex-Token)
      - PLEX_TOKEN=your_plex_token_here
      # Required - Get from https://www.themoviedb.org/settings/api
      - TMDB_READ_ACCESS_TOKEN=your_tmdb_read_access_token
      # Required - Your Plex server details
      - PLEX_SERVER=localhost
      - PLEX_PORT=32400
      - PLEX_REQUIRES_HTTPS=true
      # Process all libraries (recommended for first-time users)
      - MOVIE_PROCESS_ALL=true
      - TV_PROCESS_ALL=true
      # Optional settings
      - PROCESS_TIMER=1h
      - UPDATE_FIELD=label  # or 'genre'
      # Optional Radarr/Sonarr integration
      # - USE_RADARR=true
      # - RADARR_URL=http://radarr:7878
      # - RADARR_API_KEY=your_radarr_api_key
      # - USE_SONARR=true
      # - SONARR_URL=http://sonarr:8989
      # - SONARR_API_KEY=your_sonarr_api_key
```

**Run:** `docker-compose up -d`

### What it does

✅ **Detects TMDb IDs** from Plex metadata, Radarr/Sonarr APIs, or file paths (e.g., `{tmdb-12345}`)  
✅ **Fetches keywords** from TMDb API for movies and TV shows  
✅ **Normalizes keywords** with proper capitalization and spelling  
✅ **Adds as Plex labels/genres** - never removes existing values  
✅ **Runs automatically** on configurable timer (default: 1 hour)  
✅ **Multi-architecture** support (AMD64 + ARM64)  

### 🎉 New Features in This Fork

- **🚀 Radarr/Sonarr Integration** - Automatically detect TMDb IDs from your media managers
- **💾 Persistent Storage** - Tracks processed items across container restarts
- **🔍 Verbose Logging** - Detailed debugging information for troubleshooting
- **📝 Keyword Normalization** - Intelligent formatting with pattern recognition
- **🔄 Force Update Mode** - Reprocess all items regardless of previous processing status
- **🧹 Smart Duplicate Cleaning** - Automatically removes old unnormalized keywords when adding normalized versions
- **🔒 Enhanced Error Handling** - Better authentication and connection testing

---

<details id="examples-in-plex">
<summary><h3 style="margin: 0; display: inline;">📸 Examples in Plex</h3></summary>

![Labels](example/labels.png) ![Dynamic Filters](example/dynamic_filter.png) ![Filter](example/filter.png)

</details>

<details id="docker-run-command">
<summary><h3 style="margin: 0; display: inline;">🐳 Alternative: Docker Run Command</h3></summary>

```bash
docker run -d --name labelarr \
  -e PLEX_TOKEN=your_plex_token_here \
  -e TMDB_READ_ACCESS_TOKEN=your_tmdb_read_access_token \
  -e PLEX_SERVER=localhost -e PLEX_PORT=32400 -e PLEX_REQUIRES_HTTPS=true \
  -e MOVIE_PROCESS_ALL=true -e TV_PROCESS_ALL=true \
  ghcr.io/nullable-eth/labelarr:latest
```

</details>

<details id="plex-container-setup">
<summary><h3 style="margin: 0; display: inline;">🐳 Advanced: Running with Plex Container Ensuring Labelarr Waits for Plex</h3></summary>
To avoid Labelarr startup errors when Plex is not yet ready, use Docker Compose's depends_on with condition: service_healthy and add a healthcheck to your Plex service. This ensures Labelarr only starts after Plex is healthy.

```yaml
version: '3.8'
services:
  plex:
    image: plexinc/pms-docker:latest
    container_name: plex
    # ... your plex configuration ...
    healthcheck:
      test: curl --connect-timeout 15 --silent --show-error --fail http://localhost:32400/identity
      interval: 1m00s
      timeout: 15s
      retries: 3
      start_period: 1m00s

  labelarr:
    image: ghcr.io/nullable-eth/labelarr:latest
    container_name: labelarr
    restart: unless-stopped
    depends_on:
      plex:
        condition: service_healthy
    environment:
      - PLEX_SERVER=localhost
      - PLEX_PORT=32400
      - PLEX_REQUIRES_HTTPS=false
      - PLEX_TOKEN=your_plex_token_here
      - TMDB_READ_ACCESS_TOKEN=your_tmdb_read_access_token
      - MOVIE_PROCESS_ALL=true
      - TV_PROCESS_ALL=true
```

</details>

<details id="environment-variables">
<summary><h3 style="margin: 0; display: inline;">📋 Environment Variables</h3></summary>

**Required Settings:**

- `PLEX_TOKEN` - Get from Plex Web (F12 → Network → X-Plex-Token)
- `TMDB_READ_ACCESS_TOKEN` - Get from [TMDb API Settings](https://www.themoviedb.org/settings/api)
- `PLEX_SERVER` - Your Plex server address (e.g., `localhost`)
- `PLEX_PORT` - Usually `32400`

**Library Selection** (choose one approach):

- `MOVIE_PROCESS_ALL=true` + `TV_PROCESS_ALL=true` - Process all libraries (recommended)
- `MOVIE_LIBRARY_ID=1` + `TV_LIBRARY_ID=2` - Process specific libraries only

**Optional Settings:**

- `PLEX_REQUIRES_HTTPS=true` - Use HTTPS (default: `true`)
- `UPDATE_FIELD=label` - Field to update: `label` or `genre` (default: `label`)
- `PROCESS_TIMER=1h` - How often to run 24h, 5m, 2h30m etc. (default: `1h`)
- `REMOVE=lock` - Clean mode: `lock` or `unlock` (runs once and exits)
- `VERBOSE_LOGGING=true` - Enable detailed lookup information (default: `false`)
- `DATA_DIR=/data` - Directory for persistent storage (default: `/data`)
- `FORCE_UPDATE=true` - Force reprocess all items regardless of previous processing (default: `false`)

**Radarr Integration (Optional):**

- `USE_RADARR=true` - Enable Radarr integration (default: `false`)
- `RADARR_URL=http://localhost:7878` - Your Radarr instance URL
- `RADARR_API_KEY=your_api_key` - Your Radarr API key

**Sonarr Integration (Optional):**

- `USE_SONARR=true` - Enable Sonarr integration (default: `false`)
- `SONARR_URL=http://localhost:8989` - Your Sonarr instance URL
- `SONARR_API_KEY=your_api_key` - Your Sonarr API key

</details>

<details id="how-it-works">
<summary><h3 style="margin: 0; display: inline;">📖 How It Works</h3></summary>

1. **Movie Processing**: Iterates through all movies in the library
2. **TMDb ID Extraction**: Gets TMDb IDs from:
   - Plex metadata Guid field
   - File/folder names with `{tmdb-12345}` format
3. **Keyword Fetching**: Retrieves keywords from TMDb API
4. **Label Synchronization**: Adds new keywords as labels (preserves existing labels)
5. **Progress Tracking**: Remembers processed movies to avoid re-processing

</details>

<details id="radarr-sonarr-integration">
<summary><h3 style="margin: 0; display: inline;">🚀 Radarr/Sonarr Integration</h3></summary>

Labelarr now supports automatic TMDb ID detection through Radarr and Sonarr APIs, eliminating the need for TMDb IDs in file paths!

### Benefits

- ✅ **No file renaming required** - Works with your existing file structure
- ✅ **Multiple matching methods** - Title, year, IMDb ID, TVDb ID, file path
- ✅ **Automatic fallback** - If Radarr/Sonarr doesn't have the item, falls back to file path detection
- ✅ **Optional integration** - Enable only if you use Radarr/Sonarr

### How It Works

1. **For Movies (Radarr)**:
   - Matches by title and year
   - Falls back to IMDb ID from Plex
   - Checks file paths against Radarr's database
   - Extracts TMDb ID from matched movie

2. **For TV Shows (Sonarr)**:
   - Matches by title and year
   - Uses TVDb ID from Plex if available
   - Falls back to IMDb ID
   - Checks episode file paths against Sonarr's database
   - Extracts TMDb ID from matched series

### Configuration Example

```yaml
services:
  labelarr:
    image: ghcr.io/nullable-eth/labelarr:latest
    environment:
      # ... other config ...
      
      # Enable Radarr integration
      - USE_RADARR=true
      - RADARR_URL=http://radarr:7878
      - RADARR_API_KEY=your_radarr_api_key
      
      # Enable Sonarr integration
      - USE_SONARR=true
      - SONARR_URL=http://sonarr:8989
      - SONARR_API_KEY=your_sonarr_api_key
```

### Finding Your API Keys

**Radarr**: Settings → General → Security → API Key  
**Sonarr**: Settings → General → Security → API Key

</details>

<details id="tmdb-id-detection">
<summary><h3 style="margin: 0; display: inline;">🔍 TMDb ID Detection</h3></summary>

The application can find TMDb IDs from multiple sources and supports flexible formats:

- **Plex Metadata**: Standard TMDb agent IDs
- **Radarr/Sonarr APIs**: Automatic matching (when enabled)
- **File Paths**: Flexible TMDb ID detection in filenames or directory names

### ✅ **Supported Patterns** (Case-Insensitive)

The TMDb ID detection is very flexible and supports various formats:

**Direct Concatenation:**

- `/movies/The Matrix (1999) tmdb603/file.mkv`
- `/movies/Inception (2010) TMDB27205/file.mkv`
- `/movies/Avatar (2009) Tmdb19995/file.mkv`

**With Separators:**

- `/movies/Interstellar (2014) tmdb:157336/file.mkv`
- `/movies/The Dark Knight (2008) tmdb-155/file.mkv`
- `/movies/Pulp Fiction (1994) tmdb_680/file.mkv`
- `/movies/Fight Club (1999) tmdb=550/file.mkv`
- `/movies/The Shawshank Redemption (1994) tmdb 278/file.mkv`

**With Brackets/Braces:**

- `/movies/Goodfellas (1990) {tmdb634}/file.mkv`
- `/movies/Forrest Gump (1994) [tmdb-13]/file.mkv`
- `/movies/The Godfather (1972) (tmdb:238)/file.mkv`
- `/movies/Taxi Driver (1976) {tmdb=103}/file.mkv`
- `/movies/Casablanca (1942) (tmdb 289)/file.mkv`

**Mixed Examples:**

- `/movies/Citizen Kane (1941) something tmdb: 15678 extra/file.mkv`
- `/movies/Vertigo (1958) {tmdb=194884}/file.mkv`
- `/movies/Psycho (1960) [ tmdb-539 ]/file.mkv`

### ❌ **Will NOT Match**

- `mytmdb12345` (preceded by alphanumeric characters)
- `tmdb12345abc` (followed by alphanumeric characters)  
- `tmdb` (no digits following)

### 📁 **Example File Paths**

```
/movies/The Matrix (1999) [tmdb-603]/The Matrix.mkv
/movies/Inception (2010) (tmdb:27205)/Inception.mkv
/movies/Avatar (2009) tmdb19995/Avatar.mkv
/movies/Interstellar (2014) TMDB_157336/Interstellar.mkv
/movies/Edge Case - {tmdb=12345}/file.mkv
/movies/Colon: [tmdb:54321]/file.mkv
/movies/Semicolon; (tmdb;67890)/file.mkv
/movies/Underscore_tmdb_11111/file.mkv
/movies/ExtraSuffix tmdb-22222_extra/file.mkv
/movies/Direct tmdb194884 format/file.mkv
```

</details>

<details id="advanced-configuration">
<summary><h3 style="margin: 0; display: inline;">🔧 Advanced Configuration</h3></summary>

<details id="finding-library-ids" style="margin-left: 20px;">
<summary><strong>🔍 Finding Library IDs</strong></summary>

To find your library's ID, open your Plex web app, click on the desired library, and look for `source=` in the URL:

- `https://app.plex.tv/desktop/#!/media/xxxx/com.plexapp.plugins.library?source=1`
- Here, the library ID is `1`

**⚠️ Note**: Starting with this version, explicit library configuration is required. The application will **NOT** auto-select libraries by default.

- `MOVIE_LIBRARY_ID=1` - Process only specific movie library
- `MOVIE_PROCESS_ALL=true` - Process all movie libraries (recommended)
- Neither set: Movies are **NOT** processed

</details>

<details id="labels-vs-genres" style="margin-left: 20px;">
<summary><strong>🏷️ Labels vs Genres (UPDATE_FIELD)</strong></summary>

Control whether TMDb keywords are synced as Plex **labels** (default) or **genres**:

- `UPDATE_FIELD=label` (default): Syncs keywords as Plex labels
- `UPDATE_FIELD=genre`: Syncs keywords as Plex genres

The chosen field will be **locked** after update to prevent Plex from overwriting it.

![Example of genres updated and locked by Labelarr](example/genre.png)

</details>

<details id="removing-keywords" style="margin-left: 20px;">
<summary><strong>🗑️ Removing Keywords (REMOVE)</strong></summary>

Remove **only** TMDb keywords while preserving custom labels/genres:

- `REMOVE=lock`: Removes TMDb keywords and **locks** the field
- `REMOVE=unlock`: Removes TMDb keywords and **unlocks** the field for Plex to update

**Use lock when**: You manually manage labels/genres  
**Use unlock when**: You want Plex to refresh metadata naturally

```bash
# Example: Remove TMDb keywords from labels and lock field
docker run --rm \
  -e PLEX_TOKEN=... -e TMDB_READ_ACCESS_TOKEN=... \
  -e REMOVE=lock -e UPDATE_FIELD=label \
  -e MOVIE_PROCESS_ALL=true -e TV_PROCESS_ALL=true \
  ghcr.io/nullable-eth/labelarr:latest
```

</details>

<details id="field-locking-metadata" style="margin-left: 20px;">
<summary><strong>🔒 Field Locking & Plex Metadata</strong></summary>

**Locked fields** in Plex are protected from automatic updates:

- ✅ Labelarr can still modify them
- ✅ Manual edits in Plex UI still work
- ❌ Plex cannot overwrite during metadata refresh
- 🔒 Lock icon appears in Plex UI

**Unlocked fields** can be updated by Plex during metadata refreshes.

**Labelarr's behavior:**

- **Adding keywords**: Always locks the field
- **Remove with lock**: Keeps field locked after removing keywords
- **Remove with unlock**: Unlocks field for Plex to manage

</details>

<details id="migration" style="margin-left: 20px;">
<summary><strong>🔄 Migration from Previous Version</strong></summary>

**⚠️ Breaking Changes**: This version requires explicit library configuration.

**Old behavior**: Auto-selected first movie library  
**New behavior**: Must specify which libraries to process

**Migration steps:**

```bash
# Before (auto-selected movies)
-e LIBRARY_ID=1

# After (explicit selection)
-e MOVIE_LIBRARY_ID=1  # Specific library
# OR
-e MOVIE_PROCESS_ALL=true  # All movie libraries
-e TV_PROCESS_ALL=true     # All TV libraries
```

**New Features:**

- 📺 TV show support
- 🔇 Reduced verbose output
- 📊 Better progress tracking
- 🛡️ Enhanced error handling

</details>

</details>

<details id="field-locking">
<summary><h3 style="margin: 0; display: inline;">🔒 Understanding Field Locking & Plex Metadata</h3></summary>

Field locking is a crucial concept in Plex that determines whether Plex can automatically update metadata fields during library scans and metadata refreshes. Understanding how this works with Labelarr is essential for managing your media library effectively.

<details id="what-is-field-locking" style="margin-left: 20px;">
<summary><strong>🔐 What is Field Locking?</strong></summary>

When a field is **locked** in Plex:

- ✅ The field value is **protected** from automatic changes
- ✅ Plex **cannot** overwrite the field during metadata refresh
- ✅ Manual edits in Plex UI are still possible
- ✅ External tools (like Labelarr) can still modify the field
- 🔒 A **lock icon** appears next to the field in Plex UI

When a field is **unlocked** in Plex:

- 🔄 Plex **can** update the field during metadata refresh
- 🔄 New metadata agents can overwrite existing values
- 🔄 "Refresh Metadata" will update the field with fresh data
- 🔓 **No lock icon** appears in Plex UI

</details>

<details id="labelarr-locking-behavior" style="margin-left: 20px;">
<summary><strong>🎯 Labelarr's Field Locking Behavior</strong></summary>

#### **During Normal Operation (Adding Keywords)**

Labelarr **always locks** the field after adding TMDb keywords to prevent Plex from accidentally removing them during future metadata refreshes.

#### **During Remove Operation**

- `REMOVE=lock`: Removes TMDb keywords but **keeps the field locked**
- `REMOVE=unlock`: Removes TMDb keywords and **unlocks the field**

</details>

<details id="practical-examples" style="margin-left: 20px;">
<summary><strong>📋 Practical Examples</strong></summary>

#### **Scenario 1: Mixed Content Management**

You have movies with:

- 🏷️ TMDb keywords: `action`, `thriller`, `heist`  
- 🏷️ Custom labels: `watched`, `favorites`, `4k-remaster`

**Using `REMOVE=lock`:**

- ✅ Removes only: `action`, `thriller`, `heist`
- ✅ Keeps: `watched`, `favorites`, `4k-remaster`
- 🔒 Field remains **locked** - Plex won't add new genres
- 💡 **Best for**: Users who manually manage labels alongside TMDb keywords

**Using `REMOVE=unlock`:**

- ✅ Removes only: `action`, `thriller`, `heist`  
- ✅ Keeps: `watched`, `favorites`, `4k-remaster`
- 🔓 Field becomes **unlocked** - Plex can add new metadata
- 💡 **Best for**: Users who want Plex to manage metadata going forward

#### **Scenario 2: Complete Reset**

You want to completely reset your library's metadata:

1. **Step 1**: `REMOVE=unlock` - Removes TMDb keywords and unlocks fields
2. **Step 2**: Use Plex's "Refresh All Metadata" to restore original metadata
3. **Result**: Clean slate with Plex's default metadata

</details>

<details id="best-practices" style="margin-left: 20px;">
<summary><strong>🛡️ Best Practices</strong></summary>

#### **Use Locking When:**

- ✅ You manually curate labels/genres
- ✅ You use labels for organization (playlists, collections, etc.)
- ✅ You want to prevent accidental metadata overwrites
- ✅ You share your library and need consistent metadata

#### **Use Unlocking When:**

- ✅ You want to return to Plex's default metadata behavior
- ✅ You're switching to a different metadata agent
- ✅ You want Plex to automatically update metadata in the future
- ✅ You're troubleshooting metadata issues

</details>

<details id="visual-indicators" style="margin-left: 20px;">
<summary><strong>🔍 Visual Indicators</strong></summary>

In Plex Web UI, you'll see:

- 🔒 **Lock icon** = Field is locked (protected from automatic updates)
- 🔓 **No lock icon** = Field is unlocked (can be updated by Plex)

![Example of locked genre field in Plex](example/genre.png)

*The lock icon indicates this genre field is protected from automatic changes*

</details>

</details>

<details id="verbose-logging">
<summary><h3 style="margin: 0; display: inline;">🔍 Verbose Logging</h3></summary>

Enable verbose logging to see detailed information about TMDb ID lookups and matching attempts.

### What it shows

When `VERBOSE_LOGGING=true`, you'll see:

- 📋 All available Plex GUIDs for each item
- 🎬 Radarr lookup attempts (title, file path, IMDb ID)
- 📺 Sonarr lookup attempts (title, TVDb ID, IMDb ID, file paths)
- 📁 File path pattern matching attempts
- ✅ Successful matches with source information
- ❌ Failed lookup attempts with reasons

### Example Output

```
🔍 Starting TMDb ID lookup for movie: The Matrix (1999)
   📋 Available Plex GUIDs:
      - imdb://tt0133093
      - tmdb://603
   ✅ Found TMDb ID in Plex metadata: 603

🔍 Starting TMDb ID lookup for movie: Inception (2010)
   📋 Available Plex GUIDs:
      - imdb://tt1375666
   🎬 Checking Radarr for movie match...
      → Searching by title: "Inception" year: 2010
      ✅ Found match in Radarr: Inception (TMDb: 27205)

🔍 Starting TMDb ID lookup for TV show: Breaking Bad (2008)
   📋 Available Plex GUIDs:
      - tvdb://81189
      - imdb://tt0903747
   📺 Checking Sonarr for series match...
      → Searching by title: "Breaking Bad" year: 2008
      ❌ No match found by title/year
      → Searching by TVDb ID: 81189
      ✅ Found match by TVDb ID: Breaking Bad (TMDb: 1396)
```

### Configuration

```yaml
environment:
  - VERBOSE_LOGGING=true
```

This is especially useful for:

- Troubleshooting why certain items aren't being matched
- Understanding which data source provided the TMDb ID
- Debugging Radarr/Sonarr integration issues

</details>

<details id="keyword-normalization">
<summary><h3 style="margin: 0; display: inline;">📝 Keyword Normalization</h3></summary>

Labelarr automatically normalizes keywords from TMDb using intelligent pattern recognition and proper capitalization rules.

### How it works

- **Smart Title Casing**: Proper capitalization with article/preposition handling
- **Acronym Recognition**: Automatically detects "fbi" → "FBI", "usa" → "USA"
- **Pattern-Based Rules**: Dynamic handling of common patterns without hardcoding every keyword
- **Critical Replacements**: Known abbreviations like "sci-fi" → "Sci-Fi", "romcom" → "Romantic Comedy"
- **Intelligent Patterns**: Recognizes relationships, locations, decades, and compound terms
- **Duplicate Removal**: Removes duplicates after normalization

### Examples

**Before normalization:**

```
sci-fi, action, fbi, based on novel, time travel, woman in peril
```

**After normalization:**

```
Sci-Fi, Action, FBI, Based on Novel, Time Travel, Woman in Peril
```

### Pattern Recognition Examples

- **Critical Replacements**: `sci-fi`, `scifi`, `sci fi` → `Sci-Fi`
- **Relationships**: `father daughter` → `Father Daughter Relationship`
- **Locations**: `san francisco, california` → `San Francisco, California`
- **Versus Patterns**: `man vs nature` → `Man vs Nature`
- **Based On**: `based on novel` → `Based on Novel`
- **Decades**: `1940s` → `1940s` (preserved)
- **Ethnicity**: `african american lead` → `African American Lead`
- **General Terms**: Any multi-word keyword gets proper title casing

### Smart Duplicate Cleaning

Labelarr automatically cleans up duplicate keywords when applying normalization:

- **Removes old versions**: If you have "sci-fi" and we add "Sci-Fi", the old version is removed
- **Preserves manual keywords**: Custom tags you've added manually are always kept
- **Handles complex patterns**: Works with all normalization patterns (agencies, centuries, etc.)

### Verbose Logging

With `VERBOSE_LOGGING=true`, you'll see normalization and cleaning in action:

```
📝 Normalized: "sci-fi" → "Sci-Fi"
📝 Normalized: "fbi" → "FBI"
📝 Normalized: "based on novel" → "Based on Novel"
🧹 Cleaned 2 duplicate/unnormalized keywords
```

</details>

<details id="force-update">
<summary><h3 style="margin: 0; display: inline;">🔄 Force Update Mode</h3></summary>

Use force update mode to reprocess all items in your library, regardless of whether they've been processed before. This is especially useful after implementing keyword normalization or when you want to refresh all metadata.

### When to use Force Update

- **After enabling keyword normalization** - Update existing keywords with proper formatting
- **Configuration changes** - When switching between label/genre fields
- **Keyword cleanup** - Refresh all TMDb keywords with latest data
- **Initial migration** - When moving from another labeling system

### Configuration

```yaml
environment:
  - FORCE_UPDATE=true
```

### What it does

When `FORCE_UPDATE=true`:

- ✅ Processes all items regardless of previous processing status
- ✅ Reapplies keywords even if they already exist
- ✅ Updates storage with latest processing information
- ✅ Shows "FORCE UPDATE MODE" message in logs

### Example Output

```
✅ Found 1250 movies in library
🔄 FORCE UPDATE MODE: All items will be reprocessed regardless of previous processing
⏳ Processing movies...
```

**⚠️ Note**: Force update will reprocess your entire library, which may take time for large collections. Consider running with `VERBOSE_LOGGING=true` to monitor progress.

</details>

<details id="getting-api-keys">
<summary><h3 style="margin: 0; display: inline;">🔑 Getting API Keys</h3></summary>

### Plex Token

1. Open Plex Web App in browser
2. Press F12 → Network tab
3. Refresh the page
4. Find any request with `X-Plex-Token` in headers
5. Copy the token value

### TMDb API Key

1. Visit [TMDb API Settings](https://www.themoviedb.org/settings/api)
2. Create account and generate API key
3. Use the Read Access Token (not the API key)

</details>

<details id="troubleshooting">
<summary><h3 style="margin: 0; display: inline;">🔧 Troubleshooting</h3></summary>

### Common Issues

**401 Unauthorized from Plex**

- Verify your Plex token is correct
- Check if your Plex server requires HTTPS

**401 Unauthorized from TMDb**

- Ensure you're using a valid API token.

**No TMDb ID found**

- Check if your movies have TMDb metadata
- Verify file naming includes `{tmdb-12345}` format
- Ensure TMDb agent is used in Plex

**Connection refused**

- Check PLEX_SERVER and PLEX_PORT values
- Try setting PLEX_REQUIRES_HTTPS=false for local servers

### 🎬 Radarr Users: Ensuring TMDb ID in File Paths

If you're using Radarr to manage your movie collection, follow these steps to ensure Labelarr can detect TMDb IDs from your file paths:

#### **Configure Radarr Naming to Include TMDb ID**

Radarr can automatically include TMDb IDs in your movie file and folder names. Update your naming scheme in Radarr settings:

**Recommended Settings:**

1. **Movie Folder Format**:

   ```
   {Movie CleanTitle} ({Release Year}) {tmdb-{TmdbId}}
   ```

   *Example*: `The Matrix (1999) {tmdb-603}`

2. **Movie File Format**:

   ```
   {Movie CleanTitle} ({Release Year}) {tmdb-{TmdbId}} - {[Quality Full]}{[MediaInfo VideoDynamicRangeType]}{[Mediainfo AudioCodec}{ Mediainfo AudioChannels]}{[MediaInfo VideoCodec]}{-Release Group}
   ```

   *Example*: `The Matrix (1999) {tmdb-603} - [Bluray-1080p][x264][DTS 5.1]-GROUP`

#### **Alternative Radarr Naming Options**

If you prefer different bracket styles, these formats also work with Labelarr:

- **Square brackets**: `{Movie CleanTitle} ({Release Year}) [tmdb-{TmdbId}]`
- **Parentheses**: `{Movie CleanTitle} ({Release Year}) (tmdb-{TmdbId})`
- **Different delimiters**: `{Movie CleanTitle} ({Release Year}) {tmdb:{TmdbId}}` or `{Movie CleanTitle} ({Release Year}) {tmdb;{TmdbId}}`

#### **Common Radarr Configuration Pitfalls**

❌ **Avoid these common mistakes:**

1. **Missing TMDb ID in paths**: Default Radarr naming like `{Movie CleanTitle} ({Release Year})` doesn't include TMDb IDs
2. **Using only IMDb IDs**: `{imdb-{ImdbId}}` won't work - Labelarr specifically needs TMDb IDs
3. **Folder vs. file naming**: Ensure TMDb ID is in at least one location (folder name OR file name)

#### **Verifying Your Configuration**

After updating Radarr naming:

1. **For new movies**: TMDb IDs will be included automatically
2. **For existing movies**: Use Radarr's "Rename Files" feature:
   - Go to Movies → Select movies → Mass Editor
   - Choose your root folder and click "Yes, move files"
   - This will rename existing files to match your new naming scheme

#### **Plex Agent Compatibility**

- **New Plex Movie Agent**: Works with any naming scheme above
- **Legacy Plex Movie Agent**: May require specific TMDb ID placement for optimal matching
- **Best practice**: Include TMDb ID in folder names for maximum compatibility

#### **Example Directory Structure**

```
/movies/
├── The Matrix (1999) {tmdb-603}/
│   └── The Matrix (1999) {tmdb-603} - [Bluray-1080p].mkv
├── Inception (2010) [tmdb-27205]/
│   └── Inception (2010) [tmdb-27205] - [WEBDL-1080p].mkv
└── Avatar (2009) (tmdb:19995)/
    └── Avatar (2009) (tmdb:19995) - [Bluray-2160p].mkv
```

#### **Migration from Existing Libraries**

If you have an existing movie library without TMDb IDs in file paths:

1. **Update Radarr naming scheme** as shown above
2. **Use Radarr's mass rename feature** to update existing files
3. **Wait for Plex to detect the changes** (or manually scan library)
4. **Run Labelarr** - it will now detect TMDb IDs from the updated file paths

**⚠️ Note**: Large libraries may take time to rename. Consider doing this in batches during low-usage periods.

### 📺 Sonarr Users: Renaming Existing Folders to Include TMDb ID

If you're using Sonarr to manage your TV show collection and want to apply new folder naming that includes TMDb IDs, here's how to rename existing folders:

#### **🔄 Apply the New Folder Names**

To actually rename existing folders:

1. **Go to the Series tab**

2. **Click the Mass Editor** (three sliders icon)

3. **Select the shows** you want to rename

4. **At the bottom, click "Edit"**

5. **In the popup:**
   - Set the **Root Folder** to the same one it's already using (e.g., `/mnt/user/TV`)
   - Click **"Save"**

6. **Sonarr will interpret this as a move** and apply the new folder naming format without physically moving the files—just renaming the folders.

#### **Example Result**

After applying the new naming format, your TV show folders will include TMDb IDs:

```
/tv/Batman [tmdb-2287]/Season 3/Batman - S03E17 - The Joke's on Catwoman Bluray-1080p [tmdb-2287].mkv
```

**💡 Pro Tip**: This method works for renaming folders without actually moving files, making it safe and efficient for large TV libraries.

</details>

<details id="local-development">
<summary><h3 style="margin: 0; display: inline;">🛠️ Local Development</h3></summary>

### Prerequisites

- Go 1.23+
- Git

### Build and Run

```bash
# Clone the repository
git clone https://github.com/nullable-eth/labelarr.git
cd labelarr

# Initialize Go modules
go mod tidy

# Set environment variables
export PLEX_SERVER=localhost
export PLEX_PORT=32400
export PLEX_TOKEN=your_plex_token
export TMDB_READ_ACCESS_TOKEN=your_tmdb_read_access_token
export MOVIE_PROCESS_ALL=true
export TV_PROCESS_ALL=true

# Run the application
go run main.go
```

### Build Binary

```bash
# Build for current platform
go build -o labelarr main.go

# Build for Linux (Docker)
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o labelarr main.go
```

</details>

<details id="monitoring">
<summary><h3 style="margin: 0; display: inline;">📊 Monitoring</h3></summary>

### View Logs

```bash
# Docker logs
docker logs labelarr

# Follow logs
docker logs -f labelarr
```

### Log Output Includes

- Processing progress with movie counts
- TMDb ID detection results
- Label synchronization status
- API error handling and retries
- Detailed processing summaries

</details>

<details id="contributing">
<summary><h3 style="margin: 0; display: inline;">🤝 Contributing</h3></summary>

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

</details>

<details id="support">
<summary><h3 style="margin: 0; display: inline;">📞 Support</h3></summary>

- **GitHub**: [https://github.com/nullable-eth/labelarr](https://github.com/nullable-eth/labelarr)
- **Issues**: Report bugs and feature requests
- **Logs**: Check container logs for troubleshooting with `docker logs labelarr`

</details>

<details id="license">
<summary><h3 style="margin: 0; display: inline;">📄 License</h3></summary>

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

</details>

---

**Tags**: plex, tmdb, automation, movies, tv shows, labels, genres, docker, go, selfhosted, media management

---

⭐ **If you find this project helpful, please consider giving it a star!**
