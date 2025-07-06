package sonarr

// Series represents a TV series in Sonarr
type Series struct {
	ID               int               `json:"id"`
	Title            string            `json:"title"`
	AlternateTitles  []AlternateTitle  `json:"alternateTitles,omitempty"`
	SortTitle        string            `json:"sortTitle"`
	Year             int               `json:"year"`
	TVDbID           int               `json:"tvdbId"`
	TVRageID         int               `json:"tvRageId,omitempty"`
	TVMazeID         int               `json:"tvMazeId,omitempty"`
	TMDBID           int               `json:"tmdbId,omitempty"`
	IMDBID           string            `json:"imdbId,omitempty"`
	Status           string            `json:"status"`
	Path             string            `json:"path"`
	Images           []Image           `json:"images,omitempty"`
	Seasons          []Season          `json:"seasons,omitempty"`
	QualityProfileID int               `json:"qualityProfileId"`
	SeasonFolder     bool              `json:"seasonFolder"`
	Monitored        bool              `json:"monitored"`
	Runtime          int               `json:"runtime"`
	SeriesType       string            `json:"seriesType"`
	CleanTitle       string            `json:"cleanTitle"`
	TitleSlug        string            `json:"titleSlug"`
	FirstAired       string            `json:"firstAired,omitempty"`
	Added            string            `json:"added"`
}

// AlternateTitle represents alternate titles for a series
type AlternateTitle struct {
	Title      string `json:"title"`
	SeasonNumber int  `json:"seasonNumber,omitempty"`
}

// Image represents series artwork
type Image struct {
	CoverType string `json:"coverType"`
	URL       string `json:"url"`
	RemoteURL string `json:"remoteUrl,omitempty"`
}

// Season represents a season of a TV series
type Season struct {
	SeasonNumber int  `json:"seasonNumber"`
	Monitored    bool `json:"monitored"`
}

// Episode represents an episode of a TV series
type Episode struct {
	ID                    int      `json:"id"`
	SeriesID              int      `json:"seriesId"`
	EpisodeFileID         int      `json:"episodeFileId"`
	SeasonNumber          int      `json:"seasonNumber"`
	EpisodeNumber         int      `json:"episodeNumber"`
	Title                 string   `json:"title"`
	AirDate               string   `json:"airDate,omitempty"`
	AirDateUTC            string   `json:"airDateUtc,omitempty"`
	HasFile               bool     `json:"hasFile"`
	Monitored             bool     `json:"monitored"`
	AbsoluteEpisodeNumber int      `json:"absoluteEpisodeNumber,omitempty"`
	EpisodeFile           *EpisodeFile `json:"episodeFile,omitempty"`
}

// EpisodeFile represents the actual file for an episode
type EpisodeFile struct {
	ID           int    `json:"id"`
	SeriesID     int    `json:"seriesId"`
	SeasonNumber int    `json:"seasonNumber"`
	RelativePath string `json:"relativePath"`
	Path         string `json:"path"`
	Size         int64  `json:"size"`
	DateAdded    string `json:"dateAdded"`
}

// SystemStatus represents Sonarr system status
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