package evaluators

import (
	"context"
	"fmt"
	"strings"

	"hirescope/backend/internal/model"
)

// CertificationEvaluator evaluates job requirements with category CERTIFICATION.
type CertificationEvaluator struct{}

// NewCertificationEvaluator creates a new CertificationEvaluator.
func NewCertificationEvaluator() *CertificationEvaluator {
	return &CertificationEvaluator{}
}

func (e *CertificationEvaluator) Category() model.RequirementCategory {
	return model.CategoryCertification
}

func (e *CertificationEvaluator) Evaluate(ctx context.Context, req *model.JobRequirement, cand *model.Candidate) (*EvaluationResult, error) {
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

	// 1. Search candidate skills for certification mention
	for _, skill := range cand.Skills {
		candSkillNorm := NormalizeText(skill.Skill)
		if strings.Contains(candSkillNorm, normReq) || strings.Contains(normReq, candSkillNorm) {
			return &EvaluationResult{
				Status:       model.MatchStatusMatch,
				Evidence:     fmt.Sprintf("Candidate skill: %s", skill.Skill),
				Reason:       fmt.Sprintf("Candidate explicitly lists certification '%s'.", skill.Skill),
				Confidence:   model.ConfidenceHigh,
				Source:       model.SourceCertification,
				MatchedValue: skill.Skill,
			}, nil
		}
	}

	// 2. Search candidate headline or summary
	profileText := fmt.Sprintf("%s %s", cand.Headline, cand.Summary)
	if strings.Contains(NormalizeText(profileText), normReq) {
		snippet := cand.Headline
		if snippet == "" || !strings.Contains(NormalizeText(snippet), normReq) {
			snippet = cand.Summary
			if len(snippet) > 80 {
				snippet = snippet[:80] + "..."
			}
		}
		return &EvaluationResult{
			Status:       model.MatchStatusMatch,
			Evidence:     fmt.Sprintf("Candidate profile: %s", snippet),
			Reason:       fmt.Sprintf("Candidate profile explicitly mentions certification '%s'.", reqText),
			Confidence:   model.ConfidenceHigh,
			Source:       model.SourceCertification,
			MatchedValue: snippet,
		}, nil
	}

	// 3. Search experience descriptions
	for _, exp := range cand.Experiences {
		comb := NormalizeText(fmt.Sprintf("%s %s %s", exp.Position, exp.Company, exp.Description))
		if strings.Contains(comb, normReq) {
			evidence := fmt.Sprintf("%s at %s", exp.Position, exp.Company)
			return &EvaluationResult{
				Status:       model.MatchStatusMatch,
				Evidence:     evidence,
				Reason:       fmt.Sprintf("Candidate experience mentions certification '%s'.", reqText),
				Confidence:   model.ConfidenceMedium,
				Source:       model.SourceCertification,
				MatchedValue: evidence,
			}, nil
		}
	}

	// Not found -> UNKNOWN. Never assume absence means MISMATCH.
	return &EvaluationResult{
		Status:     model.MatchStatusUnknown,
		Evidence:   "",
		Reason:     fmt.Sprintf("No explicit certification evidence for '%s' was found in candidate profile or skills.", reqText),
		Confidence: model.ConfidenceLow,
		Source:     model.SourceNone,
	}, nil
}
