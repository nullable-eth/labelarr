package utils

import (
	"regexp"
	"strings"
	"unicode"
)

// Common acronyms and abbreviations that should remain uppercase
var commonAcronyms = map[string]bool{
	"usa":   true,
	"uk":    true,
	"us":    true,
	"u.s.":  true,
	"fbi":   true,
	"cia":   true,
	"nsa":   true,
	"dea":   true,
	"atf":   true,
	"ice":   true,
	"epa":   true,
	"irs":   true,
	"sec":   true,
	"nasa":  true,
	"nypd":  true,
	"lapd":  true,
	"swat":  true,
	"dc":    true,
	"nyc":   true,
	"la":    true,
	"sf":    true,
	"ai":    true,
	"a.i.":  true,
	"cgi":   true,
	"vr":    true,
	"ar":    true,
	"3d":    true,
	"4k":    true,
	"hd":    true,
	"uhd":   true,
	"lgbt":  true,
	"lgbtq": true,
	"wwi":   true,
	"wwii":  true,
	"ufo":   true,
	"tv":    true,
	"mtv":   true,
	"vhs":   true,
	"dvd":   true,
	"cd":    true,
	"dj":    true,
	"mc":    true,
	"bc":    true,
	"ad":    true,
	"bbc":   true,
	"cbs":   true,
	"nbc":   true,
	"abc":   true,
	"cnn":   true,
	"suv":   true,
	"rv":    true,
	"phd":   true,
	"md":    true,
	"ceo":   true,
	"cto":   true,
	"cfo":   true,
	"hr":    true,
	"it":    true,
	"pr":    true,
	"pc":    true,
	"mac":   true,
	"ios":   true,
	"os":    true,
}

// Words that should remain lowercase (articles, prepositions, conjunctions)
var lowercaseWords = map[string]bool{
	"a":      true,
	"an":     true,
	"and":    true,
	"as":     true,
	"at":     true,
	"but":    true,
	"by":     true,
	"for":    true,
	"from":   true,
	"in":     true,
	"into":   true,
	"nor":    true,
	"of":     true,
	"on":     true,
	"or":     true,
	"over":   true,
	"the":    true,
	"to":     true,
	"up":     true,
	"with":   true,
	"within": true,
}

// Critical replacements for well-known abbreviations and misspellings
var criticalReplacements = map[string]string{
	"sci-fi":               "Sci-Fi",
	"scifi":                "Sci-Fi",
	"sci fi":               "Sci-Fi",
	"romcom":               "Romantic Comedy",
	"rom-com":              "Romantic Comedy",
	"bio-pic":              "Biopic",
	"bio pic":              "Biopic",
	"neo-noir":             "Neo-Noir",
	"neo noir":             "Neo-Noir",
	"duringcreditsstinger": "During Credits Stinger",
	"aftercreditsstinger":  "After Credits Stinger",
	"midcreditsstinger":    "Mid Credits Stinger",
}

// Smart pattern matchers for dynamic normalization
var (
	// Match decade patterns like "1940s", "1990s"
	decadePattern = regexp.MustCompile(`^\d{4}s$`)

	// Match hyphenated compound words that should preserve hyphens
	hyphenatedPattern = regexp.MustCompile(`^[\w]+-[\w]+`)

	// Match "X vs Y" patterns
	versusPattern = regexp.MustCompile(`\b(\w+)\s+vs\s+(\w+)\b`)

	// Match "based on X" patterns
	basedOnPattern = regexp.MustCompile(`^based on (.+)$`)

	// Match relationship patterns like "father daughter", "mother son"
	relationshipPattern = regexp.MustCompile(`^(father|mother|parent|brother|sister|son|daughter)\s+(father|mother|parent|brother|sister|son|daughter)(?:\s+relationship)?$`)

	// Match city/state patterns like "san francisco, california"
	cityStatePattern = regexp.MustCompile(`^([^,]+),\s*([^,]+)$`)

	// Match ethnicity/nationality + descriptive word patterns
	ethnicityPattern = regexp.MustCompile(`^(african|asian|european|american|british|french|german|italian|spanish|chinese|japanese|korean|indian|mexican|latin|hispanic)\s+(american|lead|character|protagonist|antagonist|actor|actress)$`)

	// Match patterns with acronyms in parentheses like "central intelligence agency (cia)"
	acronymInParensPattern = regexp.MustCompile(`^(.+)\s+\(([a-z.]+)\)$`)

	// Match potential organization/agency patterns like "dea agent", "fbi director"
	agencyPattern = regexp.MustCompile(`^([a-z]{2,5})\s+(agent|director|officer|investigator|detective|operative|analyst|chief|deputy|special agent)$`)

	// Match century patterns like "5th century bc", "10th century"
	centuryPattern = regexp.MustCompile(`^(\d+)(st|nd|rd|th)\s+century(\s+[a-z]+)?$`)
)

// NormalizeKeyword normalizes a single keyword with proper capitalization
func NormalizeKeyword(keyword string) string {
	// Trim whitespace
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return keyword
	}

	// Convert to lowercase for pattern matching
	lowerKeyword := strings.ToLower(keyword)

	// 1. Check critical replacements first (known abbreviations)
	if replacement, exists := criticalReplacements[lowerKeyword]; exists {
		return replacement
	}

	// 2. Pattern-based normalization
	if normalized := applyPatternNormalization(lowerKeyword); normalized != "" {
		return normalized
	}

	// 3. Check if it's a known acronym (return as-is if all caps)
	if commonAcronyms[lowerKeyword] {
		return strings.ToUpper(keyword)
	}

	// 4. Apply intelligent title casing
	return applyTitleCase(keyword)
}

// applyPatternNormalization applies pattern-based rules
func applyPatternNormalization(keyword string) string {
	// Decades (1940s, 1990s, etc.)
	if decadePattern.MatchString(keyword) {
		return keyword // Keep as-is
	}

	// City, State patterns (san francisco, california)
	if matches := cityStatePattern.FindStringSubmatch(keyword); matches != nil {
		city := applyTitleCase(matches[1])
		state := applyTitleCase(matches[2])
		return city + ", " + state
	}

	// "X vs Y" patterns
	if matches := versusPattern.FindStringSubmatch(keyword); matches != nil {
		return applyTitleCase(matches[1]) + " vs " + applyTitleCase(matches[2])
	}

	// "based on X" patterns
	if matches := basedOnPattern.FindStringSubmatch(keyword); matches != nil {
		return "Based on " + applyTitleCase(matches[1])
	}

	// Relationship patterns (father daughter relationship)
	if relationshipPattern.MatchString(keyword) {
		parts := strings.Fields(keyword)
		normalized := make([]string, len(parts))
		for i, part := range parts {
			normalized[i] = titleCase(part)
		}
		// Add "Relationship" if not present
		result := strings.Join(normalized, " ")
		if !strings.HasSuffix(strings.ToLower(result), "relationship") {
			result += " Relationship"
		}
		return result
	}

	// Ethnicity + descriptor patterns (african american lead)
	if ethnicityPattern.MatchString(keyword) {
		parts := strings.Fields(keyword)
		normalized := make([]string, len(parts))
		for i, part := range parts {
			normalized[i] = titleCase(part)
		}
		return strings.Join(normalized, " ")
	}

	// Acronym in parentheses patterns (central intelligence agency (cia))
	if matches := acronymInParensPattern.FindStringSubmatch(keyword); matches != nil {
		mainPart := applyTitleCase(matches[1])
		acronymPart := strings.ToUpper(matches[2])
		return mainPart + " (" + acronymPart + ")"
	}

	// Agency/organization patterns (dea agent, fbi director)
	if matches := agencyPattern.FindStringSubmatch(keyword); matches != nil {
		agency := matches[1]
		role := matches[2]
		// Check if it's a known acronym or looks like one (2-4 letters)
		if commonAcronyms[agency] || len(agency) <= 4 {
			return strings.ToUpper(agency) + " " + titleCase(role)
		}
		// Otherwise just title case both parts
		return titleCase(agency) + " " + titleCase(role)
	}

	// Century patterns (5th century bc, 10th century)
	if matches := centuryPattern.FindStringSubmatch(keyword); matches != nil {
		century := matches[1] + matches[2] + " Century"
		if matches[3] != "" {
			// Handle BC/AD or other suffixes
			suffix := strings.TrimSpace(matches[3])
			if commonAcronyms[suffix] || len(suffix) <= 2 {
				century += " " + strings.ToUpper(suffix)
			} else {
				century += " " + titleCase(suffix)
			}
		}
		return century
	}

	return "" // No pattern matched
}

// applyTitleCase applies intelligent title casing to a phrase
func applyTitleCase(phrase string) string {
	words := strings.Fields(phrase)
	if len(words) == 0 {
		return phrase
	}

	// Title case each word
	for i, word := range words {
		lowerWord := strings.ToLower(word)

		// Check if it's an acronym
		if commonAcronyms[lowerWord] {
			words[i] = strings.ToUpper(word)
		} else if i == 0 || !lowercaseWords[lowerWord] {
			// Capitalize first word and any word that's not an article/preposition
			words[i] = titleCase(word)
		} else {
			// Keep articles/prepositions lowercase (unless first word)
			words[i] = strings.ToLower(word)
		}
	}

	return strings.Join(words, " ")
}

// titleCase converts a word to title case, preserving existing uppercase if mixed case
// Also handles hyphenated compounds by capitalizing each part
func titleCase(s string) string {
	if len(s) == 0 {
		return s
	}

	// Check if word has mixed case (like "McDonald" or "iPhone")
	hasMixedCase := false
	hasLower := false
	hasUpper := false

	for _, r := range s {
		if unicode.IsLower(r) {
			hasLower = true
		}
		if unicode.IsUpper(r) {
			hasUpper = true
		}
	}

	hasMixedCase = hasLower && hasUpper

	// If mixed case, preserve it
	if hasMixedCase {
		return s
	}

	// Handle hyphenated words by capitalizing each part
	if strings.Contains(s, "-") {
		parts := strings.Split(s, "-")
		for i, part := range parts {
			if len(part) > 0 {
				runes := []rune(strings.ToLower(part))
				runes[0] = unicode.ToUpper(runes[0])
				parts[i] = string(runes)
			}
		}
		return strings.Join(parts, "-")
	}

	// Otherwise, title case it
	runes := []rune(strings.ToLower(s))
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

// NormalizeKeywords normalizes a list of keywords
func NormalizeKeywords(keywords []string) []string {
	normalized := make([]string, 0, len(keywords))
	seen := make(map[string]bool)

	for _, keyword := range keywords {
		norm := NormalizeKeyword(keyword)

		// Avoid duplicates after normalization
		normLower := strings.ToLower(norm)
		if !seen[normLower] {
			normalized = append(normalized, norm)
			seen[normLower] = true
		}
	}

	return normalized
}

// CleanDuplicateKeywords removes old unnormalized versions when normalized versions are present
// This helps clean up libraries that have both "sci-fi" and "Sci-Fi" after normalization
func CleanDuplicateKeywords(currentKeywords, newNormalizedKeywords []string) []string {
	// Create a map of normalized keywords (lowercase) to their proper form
	normalizedMap := make(map[string]string)
	for _, keyword := range newNormalizedKeywords {
		normalizedMap[strings.ToLower(keyword)] = keyword
	}

	// Create reverse mapping - find what unnormalized versions should be replaced
	toRemove := make(map[string]bool)

	// Check each current keyword to see if it should be replaced by a normalized version
	for _, current := range currentKeywords {
		// Try to normalize this current keyword
		normalized := NormalizeKeyword(current)
		normalizedLower := strings.ToLower(normalized)

		// If the normalized version exists in our new keywords and is different from current
		if properForm, exists := normalizedMap[normalizedLower]; exists && current != properForm {
			// Mark the old version for removal
			toRemove[current] = true
		}
	}

	// Build the cleaned list
	var cleaned []string
	seen := make(map[string]bool)

	// First, add all current keywords that aren't being replaced
	for _, keyword := range currentKeywords {
		lowerKeyword := strings.ToLower(keyword)
		if !toRemove[keyword] && !seen[lowerKeyword] {
			cleaned = append(cleaned, keyword)
			seen[lowerKeyword] = true
		}
	}

	// Then add all new normalized keywords
	for _, keyword := range newNormalizedKeywords {
		lowerKeyword := strings.ToLower(keyword)
		if !seen[lowerKeyword] {
			cleaned = append(cleaned, keyword)
			seen[lowerKeyword] = true
		}
	}

	return cleaned
}
