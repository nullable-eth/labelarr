package radarr

// Movie represents a movie in Radarr
type Movie struct {
	ID               int               `json:"id"`
	Title            string            `json:"title"`
	OriginalTitle    string            `json:"originalTitle,omitempty"`
	AlternateTitles  []AlternateTitle  `json:"alternateTitles,omitempty"`
	Year             int               `json:"year"`
	TMDbID           int               `json:"tmdbId"`
	IMDbID           string            `json:"imdbId,omitempty"`
	Images           []Image           `json:"images,omitempty"`
	Status           string            `json:"status"`
	Path             string            `json:"path"`
	FolderName       string            `json:"folderName,omitempty"`
	MovieFile        MovieFile         `json:"movieFile,omitempty"`
	HasFile          bool              `json:"hasFile"`
	Monitored        bool              `json:"monitored"`
	MinimumAvailability string         `json:"minimumAvailability"`
	IsAvailable      bool              `json:"isAvailable"`
	ProfileID        int               `json:"profileId"`
	Runtime          int               `json:"runtime"`
	CleanTitle       string            `json:"cleanTitle"`
	TitleSlug        string            `json:"titleSlug"`
}

// AlternateTitle represents alternate titles for a movie
type AlternateTitle struct {
	SourceType string `json:"sourceType"`
	MovieID    int    `json:"movieId"`
	Title      string `json:"title"`
	CleanTitle string `json:"cleanTitle"`
}

// Image represents movie artwork
type Image struct {
	CoverType string `json:"coverType"`
	URL       string `json:"url"`
	RemoteURL string `json:"remoteUrl,omitempty"`
}

// MovieFile represents the actual file for a movie
type MovieFile struct {
	ID               int    `json:"id"`
	MovieID          int    `json:"movieId"`
	RelativePath     string `json:"relativePath"`
	Path             string `json:"path"`
	Size             int64  `json:"size"`
	DateAdded        string `json:"dateAdded"`
}

// SearchResult represents a movie search result
type SearchResult struct {
	Movie
}

// SystemStatus represents Radarr system status
type SystemStatus struct {
	Version          string `json:"version"`
	BuildTime        string `json:"buildTime"`
	IsDebug          bool   `json:"isDebug"`
	IsProduction     bool   `json:"isProduction"`
	IsAdmin          bool   `json:"isAdmin"`
	IsUserInteractive bool  `json:"isUserInteractive"`
	StartupPath      string `json:"startupPath"`
	AppData          string `json:"appData"`
	OsName           string `json:"osName"`
	OsVersion        string `json:"osVersion"`
	Branch           string `json:"branch"`
	Authentication   string `json:"authentication"`
}