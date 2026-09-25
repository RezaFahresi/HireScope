package evaluators

import (
	"context"
	"fmt"
	"strings"

	"hirescope/backend/internal/model"
)

// GenericEvaluator evaluates job requirements with category OTHER or fallback.
type GenericEvaluator struct{}

// NewGenericEvaluator creates an instance of GenericEvaluator.
func NewGenericEvaluator() *GenericEvaluator {
	return &GenericEvaluator{}
}

func (e *GenericEvaluator) Category() model.RequirementCategory {
	return model.CategoryOther
}

func (e *GenericEvaluator) Evaluate(ctx context.Context, req *model.JobRequirement, cand *model.Candidate) (*EvaluationResult, error) {
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

	// 1. Search candidate skills
	for _, skill := range cand.Skills {
		candSkillNorm := NormalizeText(skill.Skill)
		if strings.Contains(candSkillNorm, normReq) || strings.Contains(normReq, candSkillNorm) {
			return &EvaluationResult{
				Status:       model.MatchStatusMatch,
				Evidence:     fmt.Sprintf("Candidate skill: %s", skill.Skill),
				Reason:       "Explicit mention found in candidate skills.",
				Confidence:   model.ConfidenceHigh,
				Source:       model.SourceSkill,
				MatchedValue: skill.Skill,
			}, nil
		}
	}

	// 2. Search headline and summary
	if cand.Headline != "" && strings.Contains(NormalizeText(cand.Headline), normReq) {
		return &EvaluationResult{
			Status:       model.MatchStatusMatch,
			Evidence:     fmt.Sprintf("Candidate headline: %s", cand.Headline),
			Reason:       "Explicit mention found in candidate headline.",
			Confidence:   model.ConfidenceMedium,
			Source:       model.SourceProfile,
			MatchedValue: cand.Headline,
		}, nil
	}

	if cand.Summary != "" && strings.Contains(NormalizeText(cand.Summary), normReq) {
		snippet := cand.Summary
		if len(snippet) > 100 {
			snippet = snippet[:100] + "..."
		}
		return &EvaluationResult{
			Status:       model.MatchStatusMatch,
			Evidence:     fmt.Sprintf("Candidate summary: %s", snippet),
			Reason:       "Explicit mention found in candidate profile summary.",
			Confidence:   model.ConfidenceMedium,
			Source:       model.SourceProfile,
			MatchedValue: snippet,
		}, nil
	}

	// 3. Search experience descriptions
	for _, exp := range cand.Experiences {
		combined := NormalizeText(fmt.Sprintf("%s %s %s", exp.Position, exp.Company, exp.Description))
		if strings.Contains(combined, normReq) {
			evidence := fmt.Sprintf("%s at %s", exp.Position, exp.Company)
			return &EvaluationResult{
				Status:       model.MatchStatusMatch,
				Evidence:     evidence,
				Reason:       "Explicit mention found in candidate work experience.",
				Confidence:   model.ConfidenceMedium,
				Source:       model.SourceExperience,
				MatchedValue: evidence,
			}, nil
		}
	}

	// 4. Search education descriptions
	for _, edu := range cand.Educations {
		combined := NormalizeText(fmt.Sprintf("%s %s %s %s", edu.Institution, edu.Degree, edu.FieldOfStudy, edu.Description))
		if strings.Contains(combined, normReq) {
			evidence := fmt.Sprintf("%s — %s", edu.Institution, edu.Degree)
			return &EvaluationResult{
				Status:       model.MatchStatusMatch,
				Evidence:     evidence,
				Reason:       "Explicit mention found in candidate education background.",
				Confidence:   model.ConfidenceMedium,
				Source:       model.SourceEducation,
				MatchedValue: evidence,
			}, nil
		}
	}

	return &EvaluationResult{
		Status:     model.MatchStatusUnknown,
		Evidence:   "",
		Reason:     "No explicit evidence was found in the candidate profile.",
		Confidence: model.ConfidenceLow,
		Source:     model.SourceNone,
	}, nil
}
