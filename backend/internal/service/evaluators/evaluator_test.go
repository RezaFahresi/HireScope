package evaluators

import (
	"context"
	"testing"
	"time"

	"hirescope/backend/internal/model"
)

func TestSkillEvaluator_ExactAndAliases(t *testing.T) {
	eval := NewSkillEvaluator()
	ctx := context.Background()

	cand := &model.Candidate{
		Skills: []model.CandidateSkill{
			{Skill: "Go", NormalizedSkill: "go"},
			{Skill: "PostgreSQL", NormalizedSkill: "postgresql"},
			{Skill: "JavaScript", NormalizedSkill: "javascript"},
		},
	}

	// 1. Exact match
	res1, _ := eval.Evaluate(ctx, &model.JobRequirement{Requirement: "Go"}, cand)
	if res1.Status != model.MatchStatusMatch || res1.Source != model.SourceSkill {
		t.Errorf("expected MATCH for Go, got %v", res1.Status)
	}

	// 2. Case-insensitive / normalized match
	res2, _ := eval.Evaluate(ctx, &model.JobRequirement{Requirement: "  POSTGRESQL  "}, cand)
	if res2.Status != model.MatchStatusMatch {
		t.Errorf("expected MATCH for POSTGRESQL, got %v", res2.Status)
	}

	// 3. Alias match: Postgres -> postgresql
	res3, _ := eval.Evaluate(ctx, &model.JobRequirement{Requirement: "Postgres"}, cand)
	if res3.Status != model.MatchStatusMatch {
		t.Errorf("expected MATCH for Postgres alias, got %v", res3.Status)
	}

	// 4. Alias match: JS -> javascript
	res4, _ := eval.Evaluate(ctx, &model.JobRequirement{Requirement: "JS"}, cand)
	if res4.Status != model.MatchStatusMatch {
		t.Errorf("expected MATCH for JS alias, got %v", res4.Status)
	}

	// 5. Missing skill -> UNKNOWN
	res5, _ := eval.Evaluate(ctx, &model.JobRequirement{Requirement: "Rust"}, cand)
	if res5.Status != model.MatchStatusUnknown {
		t.Errorf("expected UNKNOWN for Rust, got %v", res5.Status)
	}
}

func TestSkillEvaluator_MultiSkill(t *testing.T) {
	eval := NewSkillEvaluator()
	ctx := context.Background()

	cand := &model.Candidate{
		Skills: []model.CandidateSkill{
			{Skill: "Go", NormalizedSkill: "go"},
			{Skill: "PostgreSQL", NormalizedSkill: "postgresql"},
		},
	}

	// 1. All matched -> MATCH
	resAll, _ := eval.Evaluate(ctx, &model.JobRequirement{Requirement: "Go, PostgreSQL"}, cand)
	if resAll.Status != model.MatchStatusMatch {
		t.Errorf("expected MATCH for full multi-skill, got %v", resAll.Status)
	}

	// 2. Partial match -> PARTIAL
	resPart, _ := eval.Evaluate(ctx, &model.JobRequirement{Requirement: "Go, PostgreSQL, Docker"}, cand)
	if resPart.Status != model.MatchStatusPartial {
		t.Errorf("expected PARTIAL for 2/3 skills, got %v", resPart.Status)
	}
	if resPart.Evidence != "Matched: Go, PostgreSQL. Missing: Docker." {
		t.Errorf("unexpected evidence: %s", resPart.Evidence)
	}

	// 3. None matched -> UNKNOWN
	resNone, _ := eval.Evaluate(ctx, &model.JobRequirement{Requirement: "Ruby, Rails, Redis"}, cand)
	if resNone.Status != model.MatchStatusUnknown {
		t.Errorf("expected UNKNOWN for 0 matched skills, got %v", resNone.Status)
	}
}

func TestExperienceEvaluator_DurationAndOverlaps(t *testing.T) {
	eval := NewExperienceEvaluator()
	ctx := context.Background()

	// 3 years ago and 1 year ago
	now := time.Now().UTC()
	d3YearsAgo := now.AddDate(-3, 0, 0)
	d1YearAgo := now.AddDate(-1, 0, 0)
	d2YearsAgo := now.AddDate(-2, 0, 0)

	// Overlapping jobs: Job 1 from 3 years ago to 1 year ago (2 years)
	// Job 2 from 2 years ago to present (2 years)
	// Combined merged non-overlapping interval: 3 years ago to present = 3 years total!
	cand := &model.Candidate{
		Experiences: []model.CandidateExperience{
			{
				Position:  "Backend Engineer",
				Company:   "Tech Co",
				StartDate: &d3YearsAgo,
				EndDate:   &d1YearAgo,
			},
			{
				Position:  "Senior Backend Engineer",
				Company:   "Startup Inc",
				StartDate: &d2YearsAgo,
				IsCurrent: true,
			},
		},
	}

	// 1. Requirement: "3 years experience" -> Should MATCH
	resMatch, _ := eval.Evaluate(ctx, &model.JobRequirement{Requirement: "3 years experience"}, cand)
	if resMatch.Status != model.MatchStatusMatch {
		t.Errorf("expected MATCH for 3 years experience, got %v (reason: %s)", resMatch.Status, resMatch.Reason)
	}

	// 2. Requirement: "5+ years of experience" -> Should be PARTIAL (candidate has 3 years)
	resPartial, _ := eval.Evaluate(ctx, &model.JobRequirement{Requirement: "5+ years of experience"}, cand)
	if resPartial.Status != model.MatchStatusPartial {
		t.Errorf("expected PARTIAL for 5 years requirement, got %v", resPartial.Status)
	}

	// 3. No usable dates -> UNKNOWN
	candNoDates := &model.Candidate{
		Experiences: []model.CandidateExperience{
			{Position: "Developer", Company: "Corp"},
		},
	}
	resUnknown, _ := eval.Evaluate(ctx, &model.JobRequirement{Requirement: "2 years experience"}, candNoDates)
	if resUnknown.Status != model.MatchStatusUnknown {
		t.Errorf("expected UNKNOWN for missing dates, got %v", resUnknown.Status)
	}

	// 4. Role-specific experience
	resRoleMatch, _ := eval.Evaluate(ctx, &model.JobRequirement{Requirement: "2 years experience as Backend Engineer"}, cand)
	if resRoleMatch.Status != model.MatchStatusMatch {
		t.Errorf("expected MATCH for Backend Engineer role experience, got %v", resRoleMatch.Status)
	}

	resRoleMismatch, _ := eval.Evaluate(ctx, &model.JobRequirement{Requirement: "2 years experience as Product Manager"}, cand)
	if resRoleMismatch.Status != model.MatchStatusUnknown {
		t.Errorf("expected UNKNOWN for non-existent role, got %v", resRoleMismatch.Status)
	}
}

func TestExperienceEvaluator_Qualitative(t *testing.T) {
	eval := NewExperienceEvaluator()
	ctx := context.Background()

	cand := &model.Candidate{
		Experiences: []model.CandidateExperience{
			{
				Position:    "Backend Developer",
				Company:     "ABC Corp",
				Description: "PostgreSQL used in application development and high-throughput query optimization",
			},
		},
	}

	// Requirement: "Experience with PostgreSQL"
	res, _ := eval.Evaluate(ctx, &model.JobRequirement{Requirement: "Experience with PostgreSQL"}, cand)
	if res.Status != model.MatchStatusMatch {
		t.Errorf("expected MATCH, got %v", res.Status)
	}
	if res.Source != model.SourceExperience {
		t.Errorf("expected SourceExperience, got %v", res.Source)
	}

	// Requirement: "Experience in fintech" (not present)
	resFintech, _ := eval.Evaluate(ctx, &model.JobRequirement{Requirement: "Experience in fintech"}, cand)
	if resFintech.Status != model.MatchStatusUnknown {
		t.Errorf("expected UNKNOWN, got %v", resFintech.Status)
	}
}

func TestEducationEvaluator(t *testing.T) {
	eval := NewEducationEvaluator()
	ctx := context.Background()

	// Candidate with Bachelor of Computer Science
	candCS := &model.Candidate{
		Educations: []model.CandidateEducation{
			{
				Institution:  "Institut Teknologi Bandung",
				Degree:       "Bachelor of Computer Science",
				FieldOfStudy: "Computer Science",
			},
		},
	}

	// 1. Bachelor match -> MATCH
	res1, _ := eval.Evaluate(ctx, &model.JobRequirement{Requirement: "Bachelor's degree"}, candCS)
	if res1.Status != model.MatchStatusMatch {
		t.Errorf("expected MATCH for Bachelor's degree, got %v", res1.Status)
	}

	// 2. Bachelor in CS match -> MATCH
	res2, _ := eval.Evaluate(ctx, &model.JobRequirement{Requirement: "Bachelor's degree in Computer Science"}, candCS)
	if res2.Status != model.MatchStatusMatch {
		t.Errorf("expected MATCH for Bachelor in CS, got %v", res2.Status)
	}

	// 3. Master's requirement -> MISMATCH (Candidate only has Bachelor)
	res3, _ := eval.Evaluate(ctx, &model.JobRequirement{Requirement: "Master's degree"}, candCS)
	if res3.Status != model.MatchStatusMismatch {
		t.Errorf("expected MISMATCH for Master's degree, got %v", res3.Status)
	}

	// 4. Field mismatch: Required Economics, candidate has CS -> MISMATCH
	res4, _ := eval.Evaluate(ctx, &model.JobRequirement{Requirement: "Bachelor's degree in Economics"}, candCS)
	if res4.Status != model.MatchStatusMismatch {
		t.Errorf("expected MISMATCH for Economics field mismatch, got %v", res4.Status)
	}

	// 5. Candidate with only institution and no degree/field -> UNKNOWN
	candNoDegree := &model.Candidate{
		Educations: []model.CandidateEducation{
			{Institution: "Universitas Indonesia"},
		},
	}
	res5, _ := eval.Evaluate(ctx, &model.JobRequirement{Requirement: "Bachelor's degree"}, candNoDegree)
	if res5.Status != model.MatchStatusUnknown {
		t.Errorf("expected UNKNOWN for missing degree info, got %v", res5.Status)
	}
}

func TestCertificationEvaluator(t *testing.T) {
	eval := NewCertificationEvaluator()
	ctx := context.Background()

	cand := &model.Candidate{
		Skills: []model.CandidateSkill{
			{Skill: "AWS Certified Solutions Architect"},
		},
	}

	// 1. Match
	resMatch, _ := eval.Evaluate(ctx, &model.JobRequirement{Requirement: "AWS Certified Solutions Architect"}, cand)
	if resMatch.Status != model.MatchStatusMatch || resMatch.Source != model.SourceCertification {
		t.Errorf("expected MATCH with SourceCertification, got %v", resMatch.Status)
	}

	// 2. Missing -> UNKNOWN (never MISMATCH)
	resMissing, _ := eval.Evaluate(ctx, &model.JobRequirement{Requirement: "PMP Certification"}, cand)
	if resMissing.Status != model.MatchStatusUnknown {
		t.Errorf("expected UNKNOWN for missing certification, got %v", resMissing.Status)
	}
}

func TestLanguageEvaluator(t *testing.T) {
	eval := NewLanguageEvaluator()
	ctx := context.Background()

	cand := &model.Candidate{
		Summary: "Senior software engineer. English — Professional Working Proficiency.",
	}

	// 1. Explicit mention -> MATCH
	resMatch, _ := eval.Evaluate(ctx, &model.JobRequirement{Requirement: "English"}, cand)
	if resMatch.Status != model.MatchStatusMatch || resMatch.Source != model.SourceLanguage {
		t.Errorf("expected MATCH with SourceLanguage, got %v", resMatch.Status)
	}

	// 2. Missing language data -> UNKNOWN
	resMissing, _ := eval.Evaluate(ctx, &model.JobRequirement{Requirement: "German"}, cand)
	if resMissing.Status != model.MatchStatusUnknown {
		t.Errorf("expected UNKNOWN for missing language, got %v", resMissing.Status)
	}
}

func TestGenericEvaluator(t *testing.T) {
	eval := NewGenericEvaluator()
	ctx := context.Background()

	cand := &model.Candidate{
		Headline: "Experienced Scrum Master and Agile Coach",
	}

	resMatch, _ := eval.Evaluate(ctx, &model.JobRequirement{Requirement: "Agile Coach"}, cand)
	if resMatch.Status != model.MatchStatusMatch || resMatch.Source != model.SourceProfile {
		t.Errorf("expected MATCH with SourceProfile, got %v", resMatch.Status)
	}

	resMissing, _ := eval.Evaluate(ctx, &model.JobRequirement{Requirement: "Six Sigma Black Belt"}, cand)
	if resMissing.Status != model.MatchStatusUnknown {
		t.Errorf("expected UNKNOWN, got %v", resMissing.Status)
	}
}
