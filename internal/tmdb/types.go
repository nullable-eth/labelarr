package tmdb

// Movie represents a TMDb movie
type Movie struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	Overview string `json:"overview"`
}

// Keyword represents a TMDb keyword
type Keyword struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// KeywordsResponse represents the response from TMDb movie keywords endpoint
type KeywordsResponse struct {
	ID       int       `json:"id"`
	Keywords []Keyword `json:"keywords"`
}

// TVKeywordsResponse represents the response from TMDb TV keywords endpoint
type TVKeywordsResponse struct {
	ID      int       `json:"id"`
	Results []Keyword `json:"results"`
}
