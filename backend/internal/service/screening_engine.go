package service

import (
	"context"
	"fmt"

	"hirescope/backend/internal/model"
	"hirescope/backend/internal/service/evaluators"
)

// ScreeningEngineResult represents the outcome of evaluating all job requirements against a candidate.
type ScreeningEngineResult struct {
	Status                 model.ScreeningResultStatus
	RequiredMatchCount     int
	RequiredPartialCount   int
	RequiredMismatchCount  int
	RequiredUnknownCount   int
	PreferredMatchCount    int
	PreferredPartialCount  int
	PreferredMismatchCount int
	PreferredUnknownCount  int
	Matches                []model.ScreeningMatch
}

// ScreeningEngine defines the contract for running deterministic requirement evaluations.
type ScreeningEngine interface {
	EvaluateRequirement(ctx context.Context, req *model.JobRequirement, cand *model.Candidate) (*evaluators.EvaluationResult, error)
	ScreenCandidate(ctx context.Context, job *model.Job, cand *model.Candidate) (*ScreeningEngineResult, error)
}

type screeningEngine struct {
	evaluators map[model.RequirementCategory]evaluators.RequirementEvaluator
	generic    evaluators.RequirementEvaluator
}

// NewScreeningEngine creates and registers all deterministic requirement evaluators.
func NewScreeningEngine() ScreeningEngine {
	skillEval := evaluators.NewSkillEvaluator()
	expEval := evaluators.NewExperienceEvaluator()
	eduEval := evaluators.NewEducationEvaluator()
	certEval := evaluators.NewCertificationEvaluator()
	langEval := evaluators.NewLanguageEvaluator()
	genericEval := evaluators.NewGenericEvaluator()

	evalMap := map[model.RequirementCategory]evaluators.RequirementEvaluator{
		model.CategorySkill:         skillEval,
		model.CategoryExperience:    expEval,
		model.CategoryEducation:     eduEval,
		model.CategoryCertification: certEval,
		model.CategoryLanguage:      langEval,
		model.CategoryOther:         genericEval,
	}

	return &screeningEngine{
		evaluators: evalMap,
		generic:    genericEval,
	}
}

func (e *screeningEngine) getEvaluator(category model.RequirementCategory) evaluators.RequirementEvaluator {
	if eval, ok := e.evaluators[category]; ok {
		return eval
	}
	return e.generic
}

func (e *screeningEngine) EvaluateRequirement(ctx context.Context, req *model.JobRequirement, cand *model.Candidate) (*evaluators.EvaluationResult, error) {
	evaluator := e.getEvaluator(req.Category)
	return evaluator.Evaluate(ctx, req, cand)
}

func (e *screeningEngine) ScreenCandidate(ctx context.Context, job *model.Job, cand *model.Candidate) (*ScreeningEngineResult, error) {
	result := &ScreeningEngineResult{
		Matches: make([]model.ScreeningMatch, 0, len(job.Requirements)),
	}

	for _, req := range job.Requirements {
		reqCopy := req
		evalRes, err := e.EvaluateRequirement(ctx, &reqCopy, cand)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate requirement %s: %w", req.ID, err)
		}

		match := model.ScreeningMatch{
			RequirementID: req.ID,
			Status:        evalRes.Status,
			Evidence:      evalRes.Evidence,
			Reason:        evalRes.Reason,
			Confidence:    evalRes.Confidence,
			Source:        evalRes.Source,
			MatchedValue:  evalRes.MatchedValue,
			Requirement:   &reqCopy,
		}
		result.Matches = append(result.Matches, match)

		// Aggregate counts by category and importance
		if req.Importance == model.ImportanceRequired {
			switch evalRes.Status {
			case model.MatchStatusMatch:
				result.RequiredMatchCount++
			case model.MatchStatusPartial:
				result.RequiredPartialCount++
			case model.MatchStatusMismatch:
				result.RequiredMismatchCount++
			default: // UNKNOWN or NOT_APPLICABLE
				result.RequiredUnknownCount++
			}
		} else { // PREFERRED
			switch evalRes.Status {
			case model.MatchStatusMatch:
				result.PreferredMatchCount++
			case model.MatchStatusPartial:
				result.PreferredPartialCount++
			case model.MatchStatusMismatch:
				result.PreferredMismatchCount++
			default: // UNKNOWN or NOT_APPLICABLE
				result.PreferredUnknownCount++
			}
		}
	}

	// Derive deterministic overall screening status:
	// QUALIFIED: All REQUIRED requirements are MATCH.
	// REVIEW: One or more REQUIRED requirements are PARTIAL or UNKNOWN and there is no explicit REQUIRED MISMATCH.
	// NOT_QUALIFIED: At least one REQUIRED requirement is MISMATCH.
	// PREFERRED requirements must NOT automatically disqualify a candidate.
	if result.RequiredMismatchCount > 0 {
		result.Status = model.StatusNotQualified
	} else if result.RequiredPartialCount > 0 || result.RequiredUnknownCount > 0 {
		result.Status = model.StatusReview
	} else {
		// All REQUIRED requirements are MATCH (or 0 REQUIRED requirements)
		result.Status = model.StatusQualified
	}

	return result, nil
}
