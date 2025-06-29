# Labelarr 🎬🏷️

[![Docker Image](https://img.shields.io/docker/v/nullableeth/labelarr?style=flat-square)](https://hub.docker.com/r/nullableeth/labelarr)
[![Docker Pulls](https://img.shields.io/docker/pulls/nullableeth/labelarr?style=flat-square)](https://hub.docker.com/r/nullableeth/labelarr)
[![Go Version](https://img.shields.io/github/go-mod/go-version/nullable-eth/labelarr?style=flat-square)](https://golang.org/)

Automatically sync TMDb movie keywords as Plex labels - A lightweight Go application that bridges your Plex movie library with The Movie Database, adding relevant keywords as searchable labels.

## 🚀 Features

- 🔄 **Periodic Processing**: Automatically processes movies on a configurable timer
- 🏷️ **Smart Label Management**: Adds TMDb keywords as Plex labels without removing existing labels
- 🔍 **Flexible TMDb ID Detection**: Extracts TMDb IDs from Plex metadata or file paths
- 📊 **Progress Tracking**: Maintains a dictionary of processed movies to avoid duplicates
- 🐳 **Docker Ready**: Containerized for easy deployment
- ⚙️ **Environment Configuration**: Fully configurable via environment variables
- 🔒 **Protocol Flexibility**: Supports both HTTP and HTTPS Plex connections
- Allows you to have TMBDB keywords as labels in Plex:
![1](https://github.com/user-attachments/assets/914c5d32-1a90-4378-be3c-38679bc6263c)
- Create custom dynamic filters for multiple labels that will update automatically when new movies are labeled:
![2](https://github.com/user-attachments/assets/23ab5d2c-9300-4560-a626-31ed836c583c)
- Filter on the fly by a label:
 
![3](https://github.com/user-attachments/assets/886df494-83c5-4fff-862d-8f51152bd68c)


## 📋 Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `PLEX_SERVER` | Plex server IP/hostname | - | **Yes** |
| `PLEX_PORT` | Plex server port | - | **Yes** |
| `PLEX_REQUIRES_HTTPS` | Use HTTPS for Plex connection | `true` | No |
| `PLEX_TOKEN` | Plex authentication token | - | **Yes** |
| `TMDB_READ_ACCESS_TOKEN` | TMDb API Bearer token | - | **Yes** |
| `PROCESS_TIMER` | Processing interval (e.g., `5m`, `1h`) | `5m` | No |
| `LIBRARY_ID` | Plex library ID (auto-detected if not set) | - | No |

## 🔑 Getting API Keys

### Plex Token

1. Open Plex Web App in browser
2. Press F12 → Network tab
3. Refresh the page
4. Find any request with `X-Plex-Token` in headers
5. Copy the token value

### TMDb API Key

1. Visit [TMDb API Settings](https://www.themoviedb.org/settings/api)
2. Create account and generate API key
3. Use the **Bearer Token** (not the API key)

## 🐳 Docker Deployment

### Quick Start

```bash
docker run -d --name labelarr \
  -e PLEX_SERVER=192.168.1.12 \
  -e PLEX_PORT=32400 \
  -e PLEX_REQUIRES_HTTPS=true \
  -e PLEX_TOKEN=your_plex_token_here \
  -e TMDB_READ_ACCESS_TOKEN=your_tmdb_read_access_token \
  -e PROCESS_TIMER=5m \
  nullableeth/labelarr:latest
```

### Docker Compose

1. Download the `docker-compose.yml` file from this repository
2. Update environment variables with your credentials:

```yaml
version: '3.8'

services:
  labelarr:
    image: nullableeth/labelarr:latest
    container_name: labelarr
    restart: unless-stopped
    environment:
      - PLEX_SERVER=192.168.1.12
      - PLEX_PORT=32400
      - PLEX_REQUIRES_HTTPS=true
      - PLEX_TOKEN=your_plex_token_here
      - TMDB_READ_ACCESS_TOKEN=your_tmdb_read_access_token
      - PROCESS_TIMER=5m
```

3. Run: `docker-compose up -d`

## 🛠️ Local Development

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
export PLEX_SERVER=192.168.1.12
export PLEX_PORT=32400
export PLEX_TOKEN=your_plex_token
export TMDB_READ_ACCESS_TOKEN=your_tmdb_read_access_token

# Run the application
go run main.go
```

### Build Binary

```bash
# Build for current platform
go build -olabelarr main.go

# Build for Linux (Docker)
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o labelarr main.go
```

## 📖 How It Works

1. **Library Discovery**: Automatically finds your movie library
2. **Movie Processing**: Iterates through all movies in the library
3. **TMDb ID Extraction**: Gets TMDb IDs from:
   - Plex metadata Guid field
   - File/folder names with `{tmdb-12345}` format
4. **Keyword Fetching**: Retrieves keywords from TMDb API
5. **Label Synchronization**: Adds new keywords as labels (preserves existing labels)
6. **Progress Tracking**: Remembers processed movies to avoid re-processing

## 🔍 TMDb ID Detection

The application can find TMDb IDs from multiple sources and supports flexible formats:

- **Plex Metadata**: Standard TMDb agent IDs
- **File Paths**: `{tmdb-12345}` in filenames or directory names
- **Flexible Formats**: The TMDb ID can be detected in a variety of patterns, not just `{tmdb-12345}`. Supported patterns include:
  - `{tmdb-12345}` (curly braces, anywhere in the folder or file name)
  - `[tmdb-12345]` (square brackets)
  - `(tmdb-12345)` (parentheses)
  - `tmdb-12345` (standalone, with or without delimiters)
  - Case-insensitive: `TMDB-12345`, `Tmdb-12345`, etc.
  - The TMDb ID can appear in either the directory or file name, and can be surrounded by spaces or other characters.
  - **Delimiters**: The TMDb ID pattern supports all common delimiters (such as `:`, `;`, `-`, `_`, etc.) between `tmdb` and the ID. For example:
    - `tmdb:15448`
    - `tmdb;15448`
    - `tmdb-15448`
    - `tmdb_15448`
    - `tmdb: 15448`, `tmdb- 15448`, etc.
    - These can appear in any of the supported bracket/brace/parenthesis formats or standalone.
    - The pattern will **not** match `tmdb15448` (no separator).

Example file paths:

```
/data/Movies/Zeitgeist - Moving Forward (2011) {tmdb-54293}/movie.mp4
/movies/The Matrix (1999) [tmdb-603]/The Matrix.mkv
/movies/Inception (2010) (tmdb:27205)/Inception.mkv
/movies/Avatar (2009) tmdb;19995/Avatar.mkv
/movies/Interstellar (2014) TMDB_157336/Interstellar.mkv
/movies/Edge Case - {tmdb-12345}/file.mkv
/movies/Colon: [tmdb:54321]/file.mkv
/movies/Semicolon; (tmdb;67890)/file.mkv
/movies/Underscore_tmdb_11111/file.mkv
/movies/ExtraSuffix tmdb-22222_extra/file.mkv
```

## 📊 Monitoring

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

## ⚙️ Configuration Examples

### For HTTP-only Plex servers

```bash
-e PLEX_REQUIRES_HTTPS=false
```

### For frequent processing

```bash
-e PROCESS_TIMER=2m
```

### For specific library

```bash
-e LIBRARY_ID=1
```

## 🔧 Troubleshooting

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

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Plex](https://www.plex.tv/) for the amazing media server
- [The Movie Database (TMDb)](https://www.themoviedb.org/) for the comprehensive movie data
- [Go](https://golang.org/) for the excellent programming language

## 📞 Support

- **Issues**: [GitHub Issues](https://github.com/nullable-eth/labelarr/issues)
- **Docker Hub**: [nullableeth/labelarr](https://hub.docker.com/r/nullableeth/labelarr)
- **Documentation**: This README and inline code comments

---

⭐ **If you find this project helpful, please consider giving it a star!**
