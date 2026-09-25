package normalizer

import (
	"regexp"
	"strings"
	"unicode"
)

var (
	// regex for 3 or more consecutive newlines
	multiNewlineRegex = regexp.MustCompile(`\n{3,}`)
	// regex for multiple consecutive horizontal spaces/tabs
	multiSpaceRegex = regexp.MustCompile(`[^\S\r\n]+`)
)

// NormalizeCVText cleans and normalizes raw CV text while strictly preserving
// meaningful line boundaries and section structure.
func NormalizeCVText(raw string) string {
	if raw == "" {
		return ""
	}

	// 1. Normalize line endings to standard LF
	text := strings.ReplaceAll(raw, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	// 2. Filter out non-printable control characters, preserving \n, \t, and valid printable runes
	var cleanRunes strings.Builder
	cleanRunes.Grow(len(text))
	for _, r := range text {
		if r == '\n' || r == '\t' || unicode.IsPrint(r) {
			cleanRunes.WriteRune(r)
		}
	}
	text = cleanRunes.String()

	// 3. Process line-by-line: collapse repeated horizontal spaces within lines, trim trailing space
	lines := strings.Split(text, "\n")
	var cleanedLines []string
	for _, line := range lines {
		// Collapse repeated horizontal whitespace
		collapsed := multiSpaceRegex.ReplaceAllString(line, " ")
		trimmed := strings.TrimRight(collapsed, " \t")
		cleanedLines = append(cleanedLines, trimmed)
	}
	text = strings.Join(cleanedLines, "\n")

	// 4. Normalize excessive blank lines (max 2 consecutive newlines)
	text = multiNewlineRegex.ReplaceAllString(text, "\n\n")

	// 5. Trim leading and trailing whitespace
	return strings.TrimSpace(text)
}
