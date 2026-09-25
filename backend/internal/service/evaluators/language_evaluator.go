package evaluators

import (
	"context"
	"fmt"
	"strings"

	"hirescope/backend/internal/model"
)

// LanguageEvaluator evaluates job requirements with category LANGUAGE.
type LanguageEvaluator struct{}

// NewLanguageEvaluator creates an instance of LanguageEvaluator.
func NewLanguageEvaluator() *LanguageEvaluator {
	return &LanguageEvaluator{}
}

func (e *LanguageEvaluator) Category() model.RequirementCategory {
	return model.CategoryLanguage
}

func (e *LanguageEvaluator) Evaluate(ctx context.Context, req *model.JobRequirement, cand *model.Candidate) (*EvaluationResult, error) {
	reqText := strings.TrimSpace(req.Requirement)
	if reqText == "" {
		return &EvaluationResult{
			Status:     model.MatchStatusNotApplicable,
			Evidence:   "Empty requirement text.",
			Reason:     "Requirement string is empty.",
			Confidence: model.ConfidenceLow,
			Source:     model.SourceNone,
		}, nil
	}

	normReq := NormalizeText(reqText)

	// 1. Search candidate skills for explicit language claim
	for _, skill := range cand.Skills {
		candSkillNorm := NormalizeText(skill.Skill)
		if strings.Contains(candSkillNorm, normReq) || strings.Contains(normReq, candSkillNorm) {
			return &EvaluationResult{
				Status:       model.MatchStatusMatch,
				Evidence:     fmt.Sprintf("Candidate skill: %s", skill.Skill),
				Reason:       fmt.Sprintf("Candidate explicitly lists language proficiency in %s.", skill.Skill),
				Confidence:   model.ConfidenceHigh,
				Source:       model.SourceLanguage,
				MatchedValue: skill.Skill,
			}, nil
		}
	}

	// 2. Search candidate headline, summary, or experience descriptions
	profileFields := []struct {
		name string
		val  string
	}{
		{"Headline", cand.Headline},
		{"Summary", cand.Summary},
	}

	for _, f := range profileFields {
		if f.val != "" && strings.Contains(NormalizeText(f.val), normReq) {
			evidence := f.val
			if len(evidence) > 80 {
				evidence = evidence[:80] + "..."
			}
			return &EvaluationResult{
				Status:       model.MatchStatusMatch,
				Evidence:     fmt.Sprintf("Candidate %s: %s", f.name, evidence),
				Reason:       fmt.Sprintf("Candidate explicitly indicates %s proficiency in profile %s.", reqText, strings.ToLower(f.name)),
				Confidence:   model.ConfidenceHigh,
				Source:       model.SourceLanguage,
				MatchedValue: evidence,
			}, nil
		}
	}

	for _, exp := range cand.Experiences {
		if strings.Contains(NormalizeText(exp.Description), normReq) {
			evidence := fmt.Sprintf("%s at %s", exp.Position, exp.Company)
			return &EvaluationResult{
				Status:       model.MatchStatusMatch,
				Evidence:     evidence,
				Reason:       fmt.Sprintf("Candidate work experience explicitly references %s proficiency.", reqText),
				Confidence:   model.ConfidenceMedium,
				Source:       model.SourceLanguage,
				MatchedValue: evidence,
			}, nil
		}
	}

	// Insufficient information -> UNKNOWN. Do not infer from education or location.
	return &EvaluationResult{
		Status:     model.MatchStatusUnknown,
		Evidence:   "",
		Reason:     fmt.Sprintf("No explicit language evidence for '%s' was found in candidate profile or skills.", reqText),
		Confidence: model.ConfidenceLow,
		Source:     model.SourceNone,
	}, nil
}
