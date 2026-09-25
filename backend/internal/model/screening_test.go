package model

import (
	"testing"
)

func TestScreeningResultStatus_Validation(t *testing.T) {
	validStatuses := []ScreeningResultStatus{
		StatusQualified,
		StatusReview,
		StatusNotQualified,
	}

	for _, s := range validStatuses {
		if !s.IsValid() {
			t.Errorf("expected status %s to be valid", s)
		}
	}

	invalidStatuses := []ScreeningResultStatus{
		"",
		"PASS",
		"REJECTED",
		"UNKNOWN",
	}

	for _, s := range invalidStatuses {
		if s.IsValid() {
			t.Errorf("expected status %s to be invalid", s)
		}
	}
}

func TestMatchStatus_Validation(t *testing.T) {
	validStatuses := []MatchStatus{
		MatchStatusMatch,
		MatchStatusPartial,
		MatchStatusMismatch,
		MatchStatusUnknown,
		MatchStatusNotApplicable,
	}

	for _, s := range validStatuses {
		if !s.IsValid() {
			t.Errorf("expected match status %s to be valid", s)
		}
	}

	if MatchStatus("INVALID").IsValid() {
		t.Errorf("expected INVALID to be invalid")
	}
}

func TestMatchConfidenceAndSource_Validation(t *testing.T) {
	if !ConfidenceHigh.IsValid() || !ConfidenceMedium.IsValid() || !ConfidenceLow.IsValid() {
		t.Errorf("expected valid confidences")
	}
	if MatchConfidence("VERY_HIGH").IsValid() {
		t.Errorf("expected invalid confidence")
	}

	validSources := []MatchSource{
		SourceSkill,
		SourceExperience,
		SourceEducation,
		SourceCertification,
		SourceLanguage,
		SourceProfile,
		SourceNone,
	}
	for _, src := range validSources {
		if !src.IsValid() {
			t.Errorf("expected source %s to be valid", src)
		}
	}
	if MatchSource("AI").IsValid() {
		t.Errorf("expected AI to be invalid match source")
	}
}

func TestScreeningResult_BeforeCreate(t *testing.T) {
	r := &ScreeningResult{
		Status: StatusQualified,
	}
	if err := r.BeforeCreate(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.ID == "" {
		t.Errorf("expected UUID to be assigned")
	}
	if r.EvaluatedAt.IsZero() {
		t.Errorf("expected EvaluatedAt to be set")
	}

	// Invalid status
	bad := &ScreeningResult{
		Status: "INVALID",
	}
	if err := bad.BeforeCreate(nil); err == nil {
		t.Errorf("expected error for invalid status")
	}
}

func TestScreeningMatch_BeforeCreate(t *testing.T) {
	m := &ScreeningMatch{
		Status:     MatchStatusMatch,
		Confidence: ConfidenceHigh,
		Source:     SourceSkill,
	}
	if err := m.BeforeCreate(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.ID == "" {
		t.Errorf("expected UUID to be assigned")
	}

	// Invalid status
	mBad := &ScreeningMatch{
		Status:     "BAD",
		Confidence: ConfidenceHigh,
		Source:     SourceSkill,
	}
	if err := mBad.BeforeCreate(nil); err == nil {
		t.Errorf("expected error for bad status")
	}
}
