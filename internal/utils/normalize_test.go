package utils

import (
	"testing"
)

// TestNormalizeKeyword tests the keyword normalization functionality
// It covers various patterns including title casing, acronyms, special replacements,
// and pattern-based normalization for agencies, centuries, locations, etc.
func TestNormalizeKeyword(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Basic capitalization
		{"action", "Action"},
		{"science fiction", "Science Fiction"},
		{"drama", "Drama"},
		
		// Acronyms
		{"fbi", "FBI"},
		{"cia", "CIA"},
		{"usa", "USA"},
		{"3d", "3D"},
		{"ai", "AI"},
		{"cgi", "CGI"},
		
		// Critical replacements (hardcoded)
		{"sci-fi", "Sci-Fi"},
		{"scifi", "Sci-Fi"},
		{"sci fi", "Sci-Fi"},
		{"bio-pic", "Biopic"},
		{"romcom", "Romantic Comedy"},
		{"neo-noir", "Neo-Noir"},
		{"duringcreditsstinger", "During Credits Stinger"},
		{"aftercreditsstinger", "After Credits Stinger"},
		
		// Pattern-based: decades
		{"1940s", "1940s"},
		{"1990s", "1990s"},
		
		// Pattern-based: city, state
		{"san francisco, california", "San Francisco, California"},
		{"new york, new york", "New York, New York"},
		
		// Pattern-based: vs patterns
		{"man vs nature", "Man vs Nature"},
		{"good vs evil", "Good vs Evil"},
		
		// Pattern-based: based on
		{"based on novel", "Based on Novel"},
		{"based on comic book", "Based on Comic Book"},
		{"based on short story", "Based on Short Story"},
		
		// Pattern-based: relationships
		{"father daughter", "Father Daughter Relationship"},
		{"father daughter relationship", "Father Daughter Relationship"},
		{"mother son", "Mother Son Relationship"},
		
		// Pattern-based: ethnicity
		{"african american lead", "African American Lead"},
		{"asian american character", "Asian American Character"},
		
		// Pattern-based: acronyms in parentheses
		{"central intelligence agency (cia)", "Central Intelligence Agency (CIA)"},
		{"artificial intelligence (a.i.)", "Artificial Intelligence (A.I.)"},
		{"united states (u.s.)", "United States (U.S.)"},
		
		// Pattern-based: agency/organization roles
		{"dea agent", "DEA Agent"},
		{"fbi director", "FBI Director"},
		{"cia operative", "CIA Operative"},
		{"nsa analyst", "NSA Analyst"},
		
		// Pattern-based: centuries
		{"5th century bc", "5th Century BC"},
		{"10th century", "10th Century"},
		{"21st century", "21st Century"},
		
		// General title casing
		{"car accident", "Car Accident"},
		{"crash landing", "Crash Landing"},
		{"giant monster", "Giant Monster"},
		{"alien race", "Alien Race"},
		{"dysfunctional relationship", "Dysfunctional Relationship"},
		{"short-term memory loss", "Short-Term Memory Loss"},
		{"screwball comedy", "Screwball Comedy"},
		{"tough cop", "Tough Cop"},
		{"fake fight", "Fake Fight"},
		{"racial segregation", "Racial Segregation"},
		{"racial tension", "Racial Tension"},
		{"racial prejudice", "Racial Prejudice"},
		{"high tech", "High Tech"},
		{"true love", "True Love"},
		{"brooklyn dodgers", "Brooklyn Dodgers"},
		
		// Articles and prepositions
		{"woman in peril", "Woman in Peril"},
		{"man of the house", "Man of the House"},
		{"tale of two cities", "Tale of Two Cities"},
		{"lord of the rings", "Lord of the Rings"},
		
		// Mixed case preservation
		{"McDonald", "McDonald"},
		{"iPhone", "iPhone"},
		{"eBay", "eBay"},
		
		// Edge cases
		{"", ""},
		{"a", "A"},
		{"THE", "The"},
		{"and", "And"},
	}

	for _, test := range tests {
		result := NormalizeKeyword(test.input)
		if result != test.expected {
			t.Errorf("NormalizeKeyword(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

// TestNormalizeKeywords tests the batch normalization functionality
// It ensures duplicates are removed after normalization and that
// all keywords are properly processed
func TestNormalizeKeywords(t *testing.T) {
	input := []string{
		"action",
		"sci-fi", 
		"fbi",
		"based on novel",
		"time travel",
		"woman in peril",
		"action", // duplicate
		"ACTION", // duplicate but different case
	}
	
	expected := []string{
		"Action",
		"Sci-Fi",
		"FBI", 
		"Based on Novel",
		"Time Travel",
		"Woman in Peril",
		// duplicates should be removed
	}
	
	result := NormalizeKeywords(input)
	
	if len(result) != len(expected) {
		t.Errorf("Expected %d keywords, got %d", len(expected), len(result))
	}
	
	for i, exp := range expected {
		if i >= len(result) || result[i] != exp {
			t.Errorf("Expected keyword %d to be %q, got %q", i, exp, result[i])
		}
	}
}

// TestCleanDuplicateKeywords tests the duplicate cleaning functionality
// This ensures old unnormalized versions are removed when normalized versions are present
func TestCleanDuplicateKeywords(t *testing.T) {
	tests := []struct {
		name               string
		currentKeywords    []string
		newNormalizedKeywords []string
		expected          []string
	}{
		{
			name: "Remove old sci-fi variants",
			currentKeywords: []string{"Action", "sci-fi", "Drama", "Custom Tag"},
			newNormalizedKeywords: []string{"Sci-Fi", "Time Travel"},
			expected: []string{"Action", "Drama", "Custom Tag", "Sci-Fi", "Time Travel"},
		},
		{
			name: "Remove multiple duplicates",
			currentKeywords: []string{"fbi", "cia", "action", "romcom", "Custom Label"},
			newNormalizedKeywords: []string{"FBI", "CIA", "Action", "Romantic Comedy"},
			expected: []string{"Custom Label", "FBI", "CIA", "Action", "Romantic Comedy"},
		},
		{
			name: "Preserve manual keywords",
			currentKeywords: []string{"My Custom Tag", "sci-fi", "Watched", "4K"},
			newNormalizedKeywords: []string{"Sci-Fi", "Adventure"},
			expected: []string{"My Custom Tag", "Watched", "4K", "Sci-Fi", "Adventure"},
		},
		{
			name: "Handle agency patterns",
			currentKeywords: []string{"dea agent", "fbi director", "Drama"},
			newNormalizedKeywords: []string{"DEA Agent", "FBI Director"},
			expected: []string{"Drama", "DEA Agent", "FBI Director"},
		},
		{
			name: "No duplicates to clean",
			currentKeywords: []string{"Action", "Drama", "My Tag"},
			newNormalizedKeywords: []string{"Sci-Fi", "Adventure"},
			expected: []string{"Action", "Drama", "My Tag", "Sci-Fi", "Adventure"},
		},
		{
			name: "Complex normalization patterns",
			currentKeywords: []string{"central intelligence agency (cia)", "5th century bc", "Custom"},
			newNormalizedKeywords: []string{"Central Intelligence Agency (CIA)", "5th Century BC"},
			expected: []string{"Custom", "Central Intelligence Agency (CIA)", "5th Century BC"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := CleanDuplicateKeywords(test.currentKeywords, test.newNormalizedKeywords)
			
			if len(result) != len(test.expected) {
				t.Errorf("Expected %d keywords, got %d", len(test.expected), len(result))
				t.Errorf("Expected: %v", test.expected)
				t.Errorf("Got: %v", result)
				return
			}
			
			// Convert to maps for easier comparison since order might vary
			expectedMap := make(map[string]bool)
			for _, keyword := range test.expected {
				expectedMap[keyword] = true
			}
			
			resultMap := make(map[string]bool)
			for _, keyword := range result {
				resultMap[keyword] = true
			}
			
			for keyword := range expectedMap {
				if !resultMap[keyword] {
					t.Errorf("Expected keyword %q not found in result", keyword)
				}
			}
			
			for keyword := range resultMap {
				if !expectedMap[keyword] {
					t.Errorf("Unexpected keyword %q found in result", keyword)
				}
			}
		})
	}
}