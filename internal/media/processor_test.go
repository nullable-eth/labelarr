package media

import "testing"

func TestExtractTMDbIDFromPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		// Direct Concatenation
		{
			name:     "Direct concatenation lowercase",
			path:     "/movies/The Matrix (1999) tmdb603/file.mkv",
			expected: "603",
		},
		{
			name:     "Direct concatenation uppercase",
			path:     "/movies/Inception (2010) TMDB27205/file.mkv",
			expected: "27205",
		},
		{
			name:     "Direct concatenation mixed case",
			path:     "/movies/Avatar (2009) Tmdb19995/file.mkv",
			expected: "19995",
		},

		// With Separators
		{
			name:     "Colon separator",
			path:     "/movies/Interstellar (2014) tmdb:157336/file.mkv",
			expected: "157336",
		},
		{
			name:     "Dash separator",
			path:     "/movies/The Dark Knight (2008) tmdb-155/file.mkv",
			expected: "155",
		},
		{
			name:     "Underscore separator",
			path:     "/movies/Pulp Fiction (1994) tmdb_680/file.mkv",
			expected: "680",
		},
		{
			name:     "Equals separator",
			path:     "/movies/Fight Club (1999) tmdb=550/file.mkv",
			expected: "550",
		},
		{
			name:     "Space separator",
			path:     "/movies/The Shawshank Redemption (1994) tmdb 278/file.mkv",
			expected: "278",
		},

		// With Brackets/Braces
		{
			name:     "Curly braces",
			path:     "/movies/Goodfellas (1990) {tmdb634}/file.mkv",
			expected: "634",
		},
		{
			name:     "Square brackets with dash",
			path:     "/movies/Forrest Gump (1994) [tmdb-13]/file.mkv",
			expected: "13",
		},
		{
			name:     "Parentheses with colon",
			path:     "/movies/The Godfather (1972) (tmdb:238)/file.mkv",
			expected: "238",
		},
		{
			name:     "Curly braces with equals",
			path:     "/movies/Taxi Driver (1976) {tmdb=103}/file.mkv",
			expected: "103",
		},
		{
			name:     "Parentheses with space",
			path:     "/movies/Casablanca (1942) (tmdb 289)/file.mkv",
			expected: "289",
		},

		// Mixed Examples
		{
			name:     "Mixed with extra text",
			path:     "/movies/Citizen Kane (1941) something tmdb: 15678 extra/file.mkv",
			expected: "15678",
		},
		{
			name:     "Curly braces with equals complex",
			path:     "/movies/Vertigo (1958) {tmdb=194884}/file.mkv",
			expected: "194884",
		},
		{
			name:     "Brackets with spaces",
			path:     "/movies/Psycho (1960) [ tmdb-539 ]/file.mkv",
			expected: "539",
		},

		// Original README Examples (Backward Compatibility)
		{
			name:     "Original bracket format",
			path:     "/movies/The Matrix (1999) [tmdb-603]/The Matrix.mkv",
			expected: "603",
		},
		{
			name:     "Original parentheses format",
			path:     "/movies/Inception (2010) (tmdb:27205)/Inception.mkv",
			expected: "27205",
		},
		{
			name:     "Original direct format",
			path:     "/movies/Avatar (2009) tmdb19995/Avatar.mkv",
			expected: "19995",
		},
		{
			name:     "Original uppercase underscore",
			path:     "/movies/Interstellar (2014) TMDB_157336/Interstellar.mkv",
			expected: "157336",
		},

		// Edge Cases - Multiple TMDb IDs (should match first)
		{
			name:     "Multiple TMDb IDs - matches first",
			path:     "/movies/Movie tmdb123 and tmdb456/file.mkv",
			expected: "123",
		},
		{
			name:     "TMDb ID in directory and filename",
			path:     "/movies/Movie tmdb123/filename tmdb456.mkv",
			expected: "123",
		},

		// Complex Real-World Examples
		{
			name:     "Complex path with year and quality",
			path:     "/media/Movies/The Matrix (1999) [1080p] {tmdb-603} [x264]/The.Matrix.1999.1080p.BluRay.x264.mkv",
			expected: "603",
		},
		{
			name:     "Radarr style naming",
			path:     "/movies/Inception (2010) {tmdb-27205} [Bluray-1080p][x264][DTS 5.1]-GROUP/Inception.mkv",
			expected: "27205",
		},

		// Should NOT Match Cases
		{
			name:     "Should not match - preceded by alphanumeric",
			path:     "mytmdb12345",
			expected: "",
		},
		{
			name:     "Should not match - followed by alphanumeric",
			path:     "tmdb12345abc",
			expected: "",
		},
		{
			name:     "Should not match - no digits",
			path:     "tmdb",
			expected: "",
		},
		{
			name:     "Should not match - no digits after tmdb",
			path:     "/movies/My Favorite tmdb Movie/file.mkv",
			expected: "",
		},
		{
			name:     "Should not match - embedded in word",
			path:     "/movies/sometmdbmovie123/file.mkv",
			expected: "",
		},
		{
			name:     "Should not match - tmdb without proper boundary",
			path:     "/movies/notmdb123/file.mkv",
			expected: "",
		},

		// Case Insensitive Tests
		{
			name:     "Mixed case TMDB",
			path:     "/movies/Movie (2020) TmDb12345/file.mkv",
			expected: "12345",
		},
		{
			name:     "All caps TMDB",
			path:     "/movies/Movie (2020) TMDB12345/file.mkv",
			expected: "12345",
		},

		// Special Characters and Unicode
		{
			name:     "Path with special characters",
			path:     "/movies/Café & Bar (2020) tmdb12345/file.mkv",
			expected: "12345",
		},
		{
			name:     "Path with unicode",
			path:     "/movies/Crème Brûlée (2020) tmdb12345/file.mkv",
			expected: "12345",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractTMDbIDFromPath(tt.path)
			if result != tt.expected {
				t.Errorf("ExtractTMDbIDFromPath(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}
}

// Benchmark test to ensure the regex is performant
func BenchmarkExtractTMDbIDFromPath(b *testing.B) {
	testPath := "/movies/The Matrix (1999) [tmdb-603]/The Matrix.mkv"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ExtractTMDbIDFromPath(testPath)
	}
}

// Test with empty and edge case inputs
func TestExtractTMDbIDFromPathEdgeCases(t *testing.T) {
	edgeCases := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "Empty string",
			path:     "",
			expected: "",
		},
		{
			name:     "Only filename",
			path:     "tmdb123.mkv",
			expected: "123",
		},
		{
			name:     "Root path",
			path:     "/tmdb123",
			expected: "123",
		},
		{
			name:     "Windows path",
			path:     "C:\\Movies\\Movie tmdb123\\file.mkv",
			expected: "123",
		},
		{
			name:     "Very long TMDb ID",
			path:     "/movies/Movie tmdb123456789012345/file.mkv",
			expected: "123456789012345",
		},
		{
			name:     "TMDb ID at start of path",
			path:     "tmdb123/movies/file.mkv",
			expected: "123",
		},
		{
			name:     "TMDb ID at end of path",
			path:     "/movies/file tmdb123",
			expected: "123",
		},
	}

	for _, tt := range edgeCases {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractTMDbIDFromPath(tt.path)
			if result != tt.expected {
				t.Errorf("ExtractTMDbIDFromPath(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}
}
