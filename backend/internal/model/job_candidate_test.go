package model

import (
	"strings"
	"testing"
)

func TestJobCandidateStatus_Validation(t *testing.T) {
	validStatuses := []JobCandidateStatus{
		JobCandidateStatusReview,
		JobCandidateStatusShortlisted,
		JobCandidateStatusRejected,
	}

	for _, s := range validStatuses {
		if !s.IsValid() {
			t.Errorf("expected status %s to be valid", s)
		}
	}

	invalidStatuses := []JobCandidateStatus{
		"",
		"HIRED",
		"ACCEPTED",
		"UNKNOWN",
	}

	for _, s := range invalidStatuses {
		if s.IsValid() {
			t.Errorf("expected status %s to be invalid", s)
		}
	}
}

func TestJobCandidate_BeforeCreate(t *testing.T) {
	jc := &JobCandidate{}
	if err := jc.BeforeCreate(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if jc.ID == "" {
		t.Errorf("expected UUID to be assigned")
	}
	if jc.Status != JobCandidateStatusReview {
		t.Errorf("expected default status REVIEW, got %s", jc.Status)
	}

	// Valid custom status
	jc2 := &JobCandidate{Status: JobCandidateStatusShortlisted}
	if err := jc2.BeforeCreate(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if jc2.Status != JobCandidateStatusShortlisted {
		t.Errorf("expected SHORTLISTED, got %s", jc2.Status)
	}

	// Invalid custom status
	jcBad := &JobCandidate{Status: "INVALID"}
	if err := jcBad.BeforeCreate(nil); err == nil {
		t.Errorf("expected error for invalid status")
	}
}

func TestCandidateNote_Validation(t *testing.T) {
	// Valid note
	n := &CandidateNote{
		Content: "Valid candidate evaluation note.",
	}
	if err := n.BeforeCreate(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n.ID == "" {
		t.Errorf("expected UUID to be generated")
	}

	// Empty note
	nEmpty := &CandidateNote{Content: ""}
	if err := nEmpty.BeforeCreate(nil); err == nil {
		t.Errorf("expected error for empty note")
	}

	// Whitespace only
	nWhitespace := &CandidateNote{Content: "   \t\n  "}
	if err := nWhitespace.BeforeCreate(nil); err == nil {
		t.Errorf("expected error for whitespace-only note")
	}

	// Length > 5000
	longContent := strings.Repeat("A", 5001)
	nLong := &CandidateNote{Content: longContent}
	if err := nLong.BeforeCreate(nil); err == nil {
		t.Errorf("expected error for note exceeding 5000 characters")
	}
}
