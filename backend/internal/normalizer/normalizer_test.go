package normalizer

import (
	"strings"
	"testing"
)

func TestNormalizeCVText(t *testing.T) {
	input := "  John Doe   \r\n\r\n\r\n\r\nSoftware   Engineer  \x00\x08\r\n\r\n" +
		"WORK   EXPERIENCE\r\n\r\n\r\nABC   Company\t\t\tDeveloper\r\n2022 - 2025\r\n\n\n\n\nSKILLS\r\nGo,   PostgreSQL,   Docker\r\n  "

	expected := "John Doe\n\nSoftware Engineer\n\nWORK EXPERIENCE\n\nABC Company Developer\n2022 - 2025\n\nSKILLS\nGo, PostgreSQL, Docker"

	normalized := NormalizeCVText(input)
	if normalized != expected {
		t.Errorf("normalized text mismatch:\nExpected:\n%q\nGot:\n%q", expected, normalized)
	}

	// Verify semantic readability was not destroyed into a single line
	if !strings.Contains(normalized, "\n") {
		t.Errorf("expected preserved line boundaries, but got single line: %s", normalized)
	}
}

func TestNormalizeCVText_Empty(t *testing.T) {
	if NormalizeCVText("") != "" {
		t.Errorf("expected empty string for empty input")
	}
	if NormalizeCVText("   \r\n\t  ") != "" {
		t.Errorf("expected empty string for whitespace-only input")
	}
}
