package evaluators

import (
	"context"
	"regexp"
	"strings"

	"hirescope/backend/internal/model"
)

// EvaluationResult represents the deterministic evaluation result for a single requirement.
type EvaluationResult struct {
	Status       model.MatchStatus     `json:"status"`
	Evidence     string                `json:"evidence"`
	Reason       string                `json:"reason"`
	Confidence   model.MatchConfidence `json:"confidence"`
	Source       model.MatchSource     `json:"source"`
	MatchedValue string                `json:"matched_value,omitempty"`
}

// RequirementEvaluator defines the contract for a requirement evaluation strategy.
type RequirementEvaluator interface {
	Category() model.RequirementCategory
	Evaluate(ctx context.Context, req *model.JobRequirement, cand *model.Candidate) (*EvaluationResult, error)
}

var multiSpaceRegex = regexp.MustCompile(`\s+`)

// NormalizeText trims, lowercases, and collapses multiple whitespace characters.
func NormalizeText(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	s = multiSpaceRegex.ReplaceAllString(s, " ")
	return s
}

// SplitSkills splits multi-skill requirements by comma, semicolon, slash, or " and ".
func SplitSkills(s string) []string {
	// First normalize separators
	s = strings.ReplaceAll(s, ";", ",")
	s = strings.ReplaceAll(s, "/", ",")

	// Split by " and " (with word boundaries or surrounding spaces)
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		subParts := strings.Split(p, " and ")
		for _, sp := range subParts {
			token := strings.TrimSpace(sp)
			if token != "" {
				result = append(result, token)
			}
		}
	}
	return result
}
