package model

import (
	"testing"
	"time"
)

func TestCandidate_Validation(t *testing.T) {
	c := &Candidate{
		FullName: "  Budi Santoso  ",
		Email:    "  Budi.Santoso@Example.COM ",
	}
	_ = c.BeforeCreate(nil)

	if c.FullName != "Budi Santoso" {
		t.Fatalf("Expected trimmed FullName 'Budi Santoso', got '%s'", c.FullName)
	}
	if c.Email != "budi.santoso@example.com" {
		t.Fatalf("Expected normalized email 'budi.santoso@example.com', got '%s'", c.Email)
	}

	// Blank full name rejected
	cEmpty := &Candidate{FullName: "   "}
	if err := cEmpty.BeforeCreate(nil); err == nil {
		t.Fatal("Expected error for blank full name, got nil")
	}
}

func TestCandidateDocument_Validation(t *testing.T) {
	// Valid TEXT document
	doc := &CandidateDocument{
		SourceType: SourceTypeText,
		RawText:    "Candidate CV text content...",
	}
	if err := doc.BeforeCreate(nil); err != nil {
		t.Fatalf("Expected valid TEXT document, got error: %v", err)
	}

	// Empty raw text for TEXT rejected
	docEmpty := &CandidateDocument{
		SourceType: SourceTypeText,
		RawText:    "   ",
	}
	if err := docEmpty.BeforeCreate(nil); err == nil {
		t.Fatal("Expected error for empty raw text in TEXT document, got nil")
	}

	// Invalid source type rejected
	docInvalid := &CandidateDocument{
		SourceType: DocumentSourceType("UNKNOWN"),
		RawText:    "Some text",
	}
	if err := docInvalid.BeforeCreate(nil); err == nil {
		t.Fatal("Expected error for invalid source type, got nil")
	}
}

func TestCandidateEducation_DateIntegrity(t *testing.T) {
	start := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC)

	edu := &CandidateEducation{
		Institution: "Universitas Indonesia",
		StartDate:   &start,
		EndDate:     &end,
	}
	if err := edu.BeforeCreate(nil); err == nil {
		t.Fatal("Expected error when end date precedes start date in education, got nil")
	}
}

func TestCandidateExperience_DateIntegrity(t *testing.T) {
	start := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)

	exp := &CandidateExperience{
		Company:   "Tech Corp",
		Position:  "Software Engineer",
		StartDate: &start,
		EndDate:   &end,
		IsCurrent: false,
	}
	if err := exp.BeforeCreate(nil); err == nil {
		t.Fatal("Expected error when end date precedes start date in experience, got nil")
	}
}

func TestCandidateSkill_Normalization(t *testing.T) {
	s := &CandidateSkill{
		Skill: "  BPMN 2.0  ",
	}
	_ = s.BeforeCreate(nil)

	if s.Skill != "BPMN 2.0" {
		t.Fatalf("Expected trimmed Skill 'BPMN 2.0', got '%s'", s.Skill)
	}
	if s.NormalizedSkill != "bpmn 2.0" {
		t.Fatalf("Expected NormalizedSkill 'bpmn 2.0', got '%s'", s.NormalizedSkill)
	}
}
