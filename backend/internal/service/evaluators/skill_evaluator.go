package evaluators

import (
	"context"
	"fmt"
	"strings"

	"hirescope/backend/internal/model"
)

// SkillEvaluator evaluates job requirements with category SKILL against candidate skills.
type SkillEvaluator struct {
	aliasMap map[string]string
}

// NewSkillEvaluator creates an instance of SkillEvaluator with deterministic aliases.
func NewSkillEvaluator() *SkillEvaluator {
	return &SkillEvaluator{
		aliasMap: map[string]string{
			"postgres":              "postgresql",
			"golang":                "go",
			"js":                    "javascript",
			"ts":                    "typescript",
			"reactjs":               "react",
			"react.js":              "react",
			"vuejs":                 "vue",
			"vue.js":                "vue",
			"nodejs":                "node.js",
			"node":                  "node.js",
			"k8s":                   "kubernetes",
			"py":                    "python",
			"rb":                    "ruby",
			"cs":                    "c#",
			"csharp":                "c#",
			"cpp":                   "c++",
			"cplusplus":             "c++",
			"aws":                   "amazon web services",
			"amazon web services":   "aws",
			"gcp":                   "google cloud platform",
			"google cloud platform": "gcp",
		},
	}
}

func (e *SkillEvaluator) Category() model.RequirementCategory {
	return model.CategorySkill
}

func (e *SkillEvaluator) resolveAlias(normalized string) string {
	if canonical, found := e.aliasMap[normalized]; found {
		return canonical
	}
	return normalized
}

func (e *SkillEvaluator) isSkillMatch(reqSkillNorm, candSkillNorm string) bool {
	if reqSkillNorm == candSkillNorm {
		return true
	}
	// Check canonical alias equivalence
	aliasReq := e.resolveAlias(reqSkillNorm)
	aliasCand := e.resolveAlias(candSkillNorm)
	if aliasReq == candSkillNorm || aliasCand == reqSkillNorm || aliasReq == aliasCand {
		return true
	}
	return false
}

func (e *SkillEvaluator) Evaluate(ctx context.Context, req *model.JobRequirement, cand *model.Candidate) (*EvaluationResult, error) {
	reqTokens := SplitSkills(req.Requirement)
	if len(reqTokens) == 0 {
		return &EvaluationResult{
			Status:     model.MatchStatusNotApplicable,
			Evidence:   "Empty requirement text.",
			Reason:     "Requirement string does not contain any evaluable skills.",
			Confidence: model.ConfidenceLow,
			Source:     model.SourceNone,
		}, nil
	}

	// Build lookup set of candidate skills
	type candSkillInfo struct {
		raw        string
		normalized string
	}
	candSkills := make([]candSkillInfo, 0, len(cand.Skills))
	for _, s := range cand.Skills {
		norm := s.NormalizedSkill
		if norm == "" {
			norm = NormalizeText(s.Skill)
		}
		candSkills = append(candSkills, candSkillInfo{
			raw:        s.Skill,
			normalized: norm,
		})
	}

	// Single skill evaluation
	if len(reqTokens) == 1 {
		target := reqTokens[0]
		targetNorm := NormalizeText(target)

		for _, cs := range candSkills {
			if e.isSkillMatch(targetNorm, cs.normalized) {
				return &EvaluationResult{
					Status:       model.MatchStatusMatch,
					Evidence:     fmt.Sprintf("Candidate skill: %s", cs.raw),
					Reason:       fmt.Sprintf("The candidate explicitly lists %s.", cs.raw),
					Confidence:   model.ConfidenceHigh,
					Source:       model.SourceSkill,
					MatchedValue: cs.raw,
				}, nil
			}
		}

		// Not found in candidate skills
		return &EvaluationResult{
			Status:     model.MatchStatusUnknown,
			Evidence:   "",
			Reason:     fmt.Sprintf("No explicit evidence of skill '%s' found in candidate skills.", target),
			Confidence: model.ConfidenceLow,
			Source:     model.SourceNone,
		}, nil
	}

	// Multi-skill evaluation
	var matchedReqs []string
	var missingReqs []string
	var matchedValues []string

	for _, token := range reqTokens {
		tokenNorm := NormalizeText(token)
		matched := false
		for _, cs := range candSkills {
			if e.isSkillMatch(tokenNorm, cs.normalized) {
				matched = true
				matchedReqs = append(matchedReqs, token)
				matchedValues = append(matchedValues, cs.raw)
				break
			}
		}
		if !matched {
			missingReqs = append(missingReqs, token)
		}
	}

	total := len(reqTokens)
	matchedCount := len(matchedReqs)

	if matchedCount == total {
		return &EvaluationResult{
			Status:       model.MatchStatusMatch,
			Evidence:     fmt.Sprintf("Matched all requested skills: %s", strings.Join(matchedValues, ", ")),
			Reason:       fmt.Sprintf("All %d requested skills are explicitly present in candidate skills.", total),
			Confidence:   model.ConfidenceHigh,
			Source:       model.SourceSkill,
			MatchedValue: strings.Join(matchedValues, ", "),
		}, nil
	}

	if matchedCount > 0 {
		return &EvaluationResult{
			Status:       model.MatchStatusPartial,
			Evidence:     fmt.Sprintf("Matched: %s. Missing: %s.", strings.Join(matchedValues, ", "), strings.Join(missingReqs, ", ")),
			Reason:       fmt.Sprintf("%d of %d explicitly requested skills are present.", matchedCount, total),
			Confidence:   model.ConfidenceHigh,
			Source:       model.SourceSkill,
			MatchedValue: strings.Join(matchedValues, ", "),
		}, nil
	}

	// 0 matched
	return &EvaluationResult{
		Status:     model.MatchStatusUnknown,
		Evidence:   fmt.Sprintf("None of the requested skills (%s) were found in candidate skills.", strings.Join(reqTokens, ", ")),
		Reason:     "No explicit evidence of the requested skills was found.",
		Confidence: model.ConfidenceLow,
		Source:     model.SourceNone,
	}, nil
}
