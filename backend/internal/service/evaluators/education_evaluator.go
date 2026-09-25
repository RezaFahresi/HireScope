package evaluators

import (
	"context"
	"fmt"
	"strings"

	"hirescope/backend/internal/model"
)

// EducationDegreeRank represents academic hierarchy rank.
type EducationDegreeRank int

const (
	RankNone      EducationDegreeRank = -1
	RankDiploma   EducationDegreeRank = 0 // D3, D4, Diploma, Associate
	RankBachelor  EducationDegreeRank = 1 // S1, Bachelor, BSc, Sarjana
	RankMaster    EducationDegreeRank = 2 // S2, Master, MSc, Magister
	RankDoctorate EducationDegreeRank = 3 // S3, PhD, Doctorate
)

// EducationEvaluator evaluates job requirements with category EDUCATION.
type EducationEvaluator struct{}

// NewEducationEvaluator creates a new EducationEvaluator.
func NewEducationEvaluator() *EducationEvaluator {
	return &EducationEvaluator{}
}

func (e *EducationEvaluator) Category() model.RequirementCategory {
	return model.CategoryEducation
}

func parseDegreeRank(text string) EducationDegreeRank {
	clean := NormalizeText(text)

	// Check doctorate
	for _, term := range []string{"doctorate", "phd", "ph.d", "doktor", "s3"} {
		if strings.Contains(clean, term) {
			return RankDoctorate
		}
	}

	// Check master
	for _, term := range []string{"master", "master's", "masters", "msc", "m.sc", "magister", "s2", "mba", "m.kom", "m.t."} {
		if strings.Contains(clean, term) {
			return RankMaster
		}
	}

	// Check bachelor
	for _, term := range []string{"bachelor", "bachelor's", "bachelors", "bsc", "b.sc", "bs", "sarjana", "s1", "s.kom", "s.t.", "s.e.", "b.eng", "undergraduate"} {
		if strings.Contains(clean, term) {
			return RankBachelor
		}
	}

	// Check diploma / associate
	for _, term := range []string{"diploma", "associate", "d3", "d4", "ahli madya"} {
		if strings.Contains(clean, term) {
			return RankDiploma
		}
	}

	return RankNone
}

func rankName(r EducationDegreeRank) string {
	switch r {
	case RankDoctorate:
		return "Doctorate (PhD / S3)"
	case RankMaster:
		return "Master's (S2 / MSc)"
	case RankBachelor:
		return "Bachelor's (S1 / BSc)"
	case RankDiploma:
		return "Diploma / Associate"
	default:
		return "Degree"
	}
}

var commonFieldAliases = map[string][]string{
	"computer science":       {"computer science", "informatika", "ilmu komputer", "cs", "teknik informatika", "sistem informasi", "information technology", "software engineering", "rekayasa perangkat lunak"},
	"information technology": {"information technology", "it", "teknologi informasi", "sistem informasi", "computer science"},
	"software engineering":   {"software engineering", "rekayasa perangkat lunak", "computer science", "informatika"},
	"data science":           {"data science", "sains data", "statistics", "statistika", "computer science"},
	"electrical engineering": {"electrical engineering", "teknik elektro", "electronics"},
	"economics":              {"economics", "ekonomi", "manajemen", "management", "accounting", "akuntansi", "finance", "keuangan"},
	"business":               {"business", "bisnis", "manajemen", "management", "business administration"},
}

func containsFieldToken(text, token string) bool {
	cleanText := NormalizeText(text)
	cleanToken := NormalizeText(token)
	if len(cleanToken) <= 3 {
		tokens := strings.FieldsFunc(cleanText, func(r rune) bool {
			return !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'))
		})
		for _, t := range tokens {
			if t == cleanToken {
				return true
			}
		}
		return false
	}
	return strings.Contains(cleanText, cleanToken)
}

func extractTargetField(text string) string {
	clean := NormalizeText(text)

	// First prioritize explicit "in <field>" construct
	if idx := strings.Index(clean, " in "); idx != -1 {
		field := strings.TrimSpace(clean[idx+4:])
		field = strings.TrimSuffix(field, " degree")
		// Check if this explicit field matches any canonical key or alias
		for key, aliases := range commonFieldAliases {
			if containsFieldToken(field, key) {
				return key
			}
			for _, alias := range aliases {
				if containsFieldToken(field, alias) {
					return key
				}
			}
		}
		return field
	}

	// Fallback to checking full text
	for key, aliases := range commonFieldAliases {
		if containsFieldToken(clean, key) {
			return key
		}
		for _, alias := range aliases {
			if containsFieldToken(clean, alias) {
				return key
			}
		}
	}

	return ""
}

func isFieldMatch(targetField, candidateFieldOrDegree string) bool {
	if targetField == "" {
		return true // No field specified
	}
	cleanCand := NormalizeText(candidateFieldOrDegree)
	if containsFieldToken(cleanCand, targetField) {
		return true
	}
	if aliases, found := commonFieldAliases[targetField]; found {
		for _, a := range aliases {
			if containsFieldToken(cleanCand, a) {
				return true
			}
		}
	}
	return false
}

func (e *EducationEvaluator) Evaluate(ctx context.Context, req *model.JobRequirement, cand *model.Candidate) (*EvaluationResult, error) {
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

	reqRank := parseDegreeRank(reqText)
	reqField := extractTargetField(reqText)

	if len(cand.Educations) == 0 {
		return &EvaluationResult{
			Status:     model.MatchStatusUnknown,
			Evidence:   "",
			Reason:     "No explicit education records were found in the candidate profile.",
			Confidence: model.ConfidenceLow,
			Source:     model.SourceNone,
		}, nil
	}

	// Inspect candidate educations
	var maxRank EducationDegreeRank = RankNone
	var bestMatch *model.CandidateEducation
	var hasRankWithField bool
	var hasDegreeRankOnly bool
	var hasAnyDegreeText bool

	for i := range cand.Educations {
		edu := &cand.Educations[i]
		combText := fmt.Sprintf("%s %s", edu.Degree, edu.FieldOfStudy)
		if strings.TrimSpace(combText) != "" {
			hasAnyDegreeText = true
		}

		rank := parseDegreeRank(combText)
		if rank > maxRank {
			maxRank = rank
			bestMatch = edu
		}

		if rank >= reqRank && (reqRank != RankNone) {
			hasDegreeRankOnly = true
			if reqField == "" || isFieldMatch(reqField, combText) {
				hasRankWithField = true
				bestMatch = edu
				break
			}
		}
	}

	// If candidate education records have institutions but no degree/field details at all
	if !hasAnyDegreeText {
		return &EvaluationResult{
			Status:     model.MatchStatusUnknown,
			Evidence:   fmt.Sprintf("Institution: %s", cand.Educations[0].Institution),
			Reason:     "Education data has institution but no degree or field of study information.",
			Confidence: model.ConfidenceLow,
			Source:     model.SourceEducation,
		}, nil
	}

	// Case 1: Requirement specifies a minimum degree rank (e.g. Master, Bachelor)
	if reqRank != RankNone {
		// Degree rank mismatch (e.g. Requires Master's, candidate only has Bachelor's)
		if maxRank < reqRank {
			var candEvidence string
			if bestMatch != nil {
				candEvidence = fmt.Sprintf("%s — %s %s", bestMatch.Institution, bestMatch.Degree, bestMatch.FieldOfStudy)
			} else {
				candEvidence = "No higher education degree found."
			}
			return &EvaluationResult{
				Status:       model.MatchStatusMismatch,
				Evidence:     strings.TrimSpace(candEvidence),
				Reason:       fmt.Sprintf("Candidate has %s, which does not satisfy the required %s.", rankName(maxRank), rankName(reqRank)),
				Confidence:   model.ConfidenceHigh,
				Source:       model.SourceEducation,
				MatchedValue: strings.TrimSpace(candEvidence),
			}, nil
		}

		// Degree rank is satisfied. Now evaluate field of study if specified
		if reqField != "" {
			if hasRankWithField {
				evidence := fmt.Sprintf("%s — %s", bestMatch.Institution, bestMatch.Degree)
				if bestMatch.FieldOfStudy != "" && !strings.Contains(bestMatch.Degree, bestMatch.FieldOfStudy) {
					evidence = fmt.Sprintf("%s (%s)", evidence, bestMatch.FieldOfStudy)
				}
				return &EvaluationResult{
					Status:       model.MatchStatusMatch,
					Evidence:     evidence,
					Reason:       fmt.Sprintf("Candidate education explicitly contains a %s in a relevant field.", rankName(reqRank)),
					Confidence:   model.ConfidenceHigh,
					Source:       model.SourceEducation,
					MatchedValue: evidence,
				}, nil
			}

			// Degree rank is satisfied, but field is missing or explicitly different
			if hasDegreeRankOnly {
				var candField string
				if bestMatch != nil && bestMatch.FieldOfStudy != "" {
					candField = bestMatch.FieldOfStudy
				} else if bestMatch != nil && bestMatch.Degree != "" {
					candField = bestMatch.Degree
				}

				if candField != "" {
					// Field is explicitly different
					evidence := fmt.Sprintf("%s — %s", bestMatch.Institution, candField)
					return &EvaluationResult{
						Status:       model.MatchStatusMismatch,
						Evidence:     evidence,
						Reason:       fmt.Sprintf("Candidate degree is in '%s', which does not match the required field of '%s'.", candField, reqField),
						Confidence:   model.ConfidenceHigh,
						Source:       model.SourceEducation,
						MatchedValue: evidence,
					}, nil
				}

				// Field of study is missing
				return &EvaluationResult{
					Status:     model.MatchStatusUnknown,
					Evidence:   fmt.Sprintf("%s — %s", bestMatch.Institution, bestMatch.Degree),
					Reason:     fmt.Sprintf("Candidate has %s, but field of study is unspecified to evaluate against '%s'.", rankName(reqRank), reqField),
					Confidence: model.ConfidenceMedium,
					Source:     model.SourceEducation,
				}, nil
			}
		}

		// Degree rank satisfied and no specific field required
		evidence := fmt.Sprintf("%s — %s", bestMatch.Institution, bestMatch.Degree)
		return &EvaluationResult{
			Status:       model.MatchStatusMatch,
			Evidence:     evidence,
			Reason:       fmt.Sprintf("Candidate explicitly satisfies the required %s credential.", rankName(reqRank)),
			Confidence:   model.ConfidenceHigh,
			Source:       model.SourceEducation,
			MatchedValue: evidence,
		}, nil
	}

	// Case 2: Requirement specifies a field without explicit degree rank (e.g. "Computer Science degree")
	if reqField != "" {
		for _, edu := range cand.Educations {
			comb := fmt.Sprintf("%s %s", edu.Degree, edu.FieldOfStudy)
			if isFieldMatch(reqField, comb) {
				evidence := fmt.Sprintf("%s — %s", edu.Institution, comb)
				return &EvaluationResult{
					Status:       model.MatchStatusMatch,
					Evidence:     evidence,
					Reason:       fmt.Sprintf("Candidate education explicitly mentions studies in %s.", reqField),
					Confidence:   model.ConfidenceHigh,
					Source:       model.SourceEducation,
					MatchedValue: evidence,
				}, nil
			}
		}

		return &EvaluationResult{
			Status:     model.MatchStatusUnknown,
			Evidence:   "",
			Reason:     fmt.Sprintf("No explicit evidence of degree/studies in '%s' found in candidate education.", reqField),
			Confidence: model.ConfidenceLow,
			Source:     model.SourceNone,
		}, nil
	}

	// Generic education text match against institution or degree
	for _, edu := range cand.Educations {
		comb := NormalizeText(fmt.Sprintf("%s %s %s", edu.Institution, edu.Degree, edu.FieldOfStudy))
		if strings.Contains(comb, NormalizeText(reqText)) {
			evidence := fmt.Sprintf("%s — %s %s", edu.Institution, edu.Degree, edu.FieldOfStudy)
			return &EvaluationResult{
				Status:       model.MatchStatusMatch,
				Evidence:     strings.TrimSpace(evidence),
				Reason:       "Explicit education match found in candidate profile.",
				Confidence:   model.ConfidenceHigh,
				Source:       model.SourceEducation,
				MatchedValue: strings.TrimSpace(evidence),
			}, nil
		}
	}

	return &EvaluationResult{
		Status:     model.MatchStatusUnknown,
		Evidence:   "",
		Reason:     "No explicit evidence was found in candidate education records.",
		Confidence: model.ConfidenceLow,
		Source:     model.SourceNone,
	}, nil
}
