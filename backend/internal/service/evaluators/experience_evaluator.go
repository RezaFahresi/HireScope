package evaluators

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"hirescope/backend/internal/model"
)

// ExperienceEvaluator evaluates job requirements with category EXPERIENCE.
type ExperienceEvaluator struct {
	durationRegexes []*regexp.Regexp
}

// NewExperienceEvaluator creates an instance of ExperienceEvaluator.
func NewExperienceEvaluator() *ExperienceEvaluator {
	return &ExperienceEvaluator{
		durationRegexes: []*regexp.Regexp{
			// "5+ years of experience", "3 years experience", "3 yrs"
			regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)\s*(?:\+)?\s*(?:years?|yrs?)(?:\s+of)?(?:\s+experience)?`),
			// "minimum 2 years", "at least 4 years", "min 3 years"
			regexp.MustCompile(`(?i)(?:minimum|at least|min\.?)\s*(\d+(?:\.\d+)?)\s*(?:years?|yrs?)`),
			// "2 - 5 years" (takes lower bound)
			regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)\s*-\s*\d+(?:\.\d+)?\s*(?:years?|yrs?)`),
		},
	}
}

func (e *ExperienceEvaluator) Category() model.RequirementCategory {
	return model.CategoryExperience
}

type dateInterval struct {
	start time.Time
	end   time.Time
}

// calculateTotalExperience merges overlapping intervals and computes total years and months.
func calculateTotalExperience(experiences []model.CandidateExperience, now time.Time) (float64, int, int, bool) {
	var intervals []dateInterval

	for _, exp := range experiences {
		if exp.StartDate == nil {
			continue
		}
		start := *exp.StartDate
		var end time.Time
		if exp.IsCurrent || exp.EndDate == nil {
			end = now
		} else {
			end = *exp.EndDate
		}

		if end.Before(start) {
			continue
		}
		intervals = append(intervals, dateInterval{start: start, end: end})
	}

	if len(intervals) == 0 {
		return 0, 0, 0, false
	}

	// Sort intervals chronologically by start date
	sort.Slice(intervals, func(i, j int) bool {
		return intervals[i].start.Before(intervals[j].start)
	})

	// Merge overlapping or adjacent intervals
	var merged []dateInterval
	current := intervals[0]

	for i := 1; i < len(intervals); i++ {
		iv := intervals[i]
		if iv.start.Before(current.end) || iv.start.Equal(current.end) {
			if iv.end.After(current.end) {
				current.end = iv.end
			}
		} else {
			merged = append(merged, current)
			current = iv
		}
	}
	merged = append(merged, current)

	var totalDays float64
	for _, iv := range merged {
		totalDays += iv.end.Sub(iv.start).Hours() / 24.0
	}

	totalYears := totalDays / 365.25
	yearsInt := int(totalYears)
	remainderDays := totalDays - (float64(yearsInt) * 365.25)
	monthsInt := int(remainderDays / 30.4375)
	if monthsInt < 0 {
		monthsInt = 0
	}
	if monthsInt >= 12 {
		yearsInt++
		monthsInt = 0
	}

	return totalYears, yearsInt, monthsInt, true
}

func (e *ExperienceEvaluator) parseRequiredYears(text string) (float64, bool) {
	for _, re := range e.durationRegexes {
		matches := re.FindStringSubmatch(text)
		if len(matches) > 1 {
			val, err := strconv.ParseFloat(matches[1], 64)
			if err == nil && val > 0 {
				return val, true
			}
		}
	}
	return 0, false
}

// extractRoleFilter checks if requirement specifies a role/domain like "as Business Analyst" or "in fintech".
func extractRoleFilter(text string) string {
	clean := NormalizeText(text)
	// Remove common duration prefixes
	for _, prefix := range []string{"years of experience in ", "years of experience as ", "years experience in ", "years experience as ", "years in ", "years as ", "experience in ", "experience as ", "experience with "} {
		if idx := strings.Index(clean, prefix); idx != -1 {
			return strings.TrimSpace(clean[idx+len(prefix):])
		}
	}
	return ""
}

func (e *ExperienceEvaluator) Evaluate(ctx context.Context, req *model.JobRequirement, cand *model.Candidate) (*EvaluationResult, error) {
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

	reqYears, hasRequiredYears := e.parseRequiredYears(reqText)
	roleFilter := extractRoleFilter(reqText)

	// Filter candidate experiences if role or domain is explicitly targeted
	targetExperiences := cand.Experiences
	if roleFilter != "" {
		var filtered []model.CandidateExperience
		roleTokens := strings.Fields(roleFilter)
		for _, exp := range cand.Experiences {
			comb := NormalizeText(fmt.Sprintf("%s %s %s", exp.Position, exp.Company, exp.Description))
			allTokensFound := true
			for _, token := range roleTokens {
				// Ignore noise words
				if token == "a" || token == "an" || token == "the" || token == "or" || token == "and" {
					continue
				}
				if !strings.Contains(comb, token) {
					allTokensFound = false
					break
				}
			}
			if allTokensFound {
				filtered = append(filtered, exp)
			}
		}
		targetExperiences = filtered
	}

	now := time.Now().UTC()

	// Case 1: Requirement specifies a duration (e.g. "3 years experience")
	if hasRequiredYears {
		totalYears, yearsInt, monthsInt, hasUsableDates := calculateTotalExperience(targetExperiences, now)
		if !hasUsableDates {
			if len(cand.Experiences) == 0 {
				return &EvaluationResult{
					Status:     model.MatchStatusUnknown,
					Evidence:   "",
					Reason:     "Candidate has no recorded work experience.",
					Confidence: model.ConfidenceLow,
					Source:     model.SourceNone,
				}, nil
			}
			return &EvaluationResult{
				Status:     model.MatchStatusUnknown,
				Evidence:   "",
				Reason:     "Candidate experience records do not have usable date ranges to determine duration.",
				Confidence: model.ConfidenceLow,
				Source:     model.SourceNone,
			}, nil
		}

		durationStr := fmt.Sprintf("%d years %d months", yearsInt, monthsInt)
		if yearsInt == 0 {
			durationStr = fmt.Sprintf("%d months", monthsInt)
		}

		// Exact or greater duration
		// Allow a conservative grace margin of 0.1 year (~1 month) for rounding
		if totalYears >= (reqYears - 0.08) {
			return &EvaluationResult{
				Status:       model.MatchStatusMatch,
				Evidence:     fmt.Sprintf("Candidate experience records indicate approximately %s.", durationStr),
				Reason:       fmt.Sprintf("Candidate meets or exceeds the required duration of %.1f years.", reqYears),
				Confidence:   model.ConfidenceHigh,
				Source:       model.SourceExperience,
				MatchedValue: durationStr,
			}, nil
		}

		// Relevant experience exists, but duration is less
		return &EvaluationResult{
			Status:       model.MatchStatusPartial,
			Evidence:     fmt.Sprintf("Candidate experience records indicate approximately %s.", durationStr),
			Reason:       fmt.Sprintf("Candidate has relevant experience but does not fully satisfy the required duration of %.1f years.", reqYears),
			Confidence:   model.ConfidenceMedium,
			Source:       model.SourceExperience,
			MatchedValue: durationStr,
		}, nil
	}

	// Case 2: Qualitative experience requirement (e.g. "Experience with PostgreSQL", "Experience in fintech")
	normReq := NormalizeText(reqText)
	// Remove generic leading phrase "experience with " or "experience in "
	searchTarget := normReq
	for _, p := range []string{"experience with ", "experience in ", "experience as "} {
		if strings.HasPrefix(searchTarget, p) {
			searchTarget = strings.TrimPrefix(searchTarget, p)
			break
		}
	}
	searchTarget = strings.TrimSpace(searchTarget)

	for _, exp := range cand.Experiences {
		combined := NormalizeText(fmt.Sprintf("%s at %s — %s", exp.Position, exp.Company, exp.Description))
		if strings.Contains(combined, searchTarget) {
			evidenceSnippet := fmt.Sprintf("%s at %s", exp.Position, exp.Company)
			if exp.Description != "" {
				descSnippet := exp.Description
				if len(descSnippet) > 80 {
					descSnippet = descSnippet[:80] + "..."
				}
				evidenceSnippet = fmt.Sprintf("%s — %s", evidenceSnippet, descSnippet)
			}
			return &EvaluationResult{
				Status:       model.MatchStatusMatch,
				Evidence:     evidenceSnippet,
				Reason:       fmt.Sprintf("Candidate experience explicitly mentions %s.", searchTarget),
				Confidence:   model.ConfidenceHigh,
				Source:       model.SourceExperience,
				MatchedValue: evidenceSnippet,
			}, nil
		}
	}

	// If not found in experience records
	return &EvaluationResult{
		Status:     model.MatchStatusUnknown,
		Evidence:   "",
		Reason:     fmt.Sprintf("No explicit evidence of '%s' was found in the candidate experience records.", searchTarget),
		Confidence: model.ConfidenceLow,
		Source:     model.SourceNone,
	}, nil
}
