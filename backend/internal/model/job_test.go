package model

import (
	"testing"
)

func TestJobStatus_Transitions(t *testing.T) {
	// DRAFT transitions
	if !JobStatusDraft.CanTransitionTo(JobStatusOpen) {
		t.Error("Expected DRAFT -> OPEN to be valid")
	}
	if !JobStatusDraft.CanTransitionTo(JobStatusArchived) {
		t.Error("Expected DRAFT -> ARCHIVED to be valid")
	}
	if JobStatusDraft.CanTransitionTo(JobStatusClosed) {
		t.Error("Expected DRAFT -> CLOSED to be invalid")
	}

	// OPEN transitions
	if !JobStatusOpen.CanTransitionTo(JobStatusClosed) {
		t.Error("Expected OPEN -> CLOSED to be valid")
	}
	if !JobStatusOpen.CanTransitionTo(JobStatusArchived) {
		t.Error("Expected OPEN -> ARCHIVED to be valid")
	}
	if JobStatusOpen.CanTransitionTo(JobStatusDraft) {
		t.Error("Expected OPEN -> DRAFT to be invalid")
	}

	// CLOSED transitions
	if !JobStatusClosed.CanTransitionTo(JobStatusOpen) {
		t.Error("Expected CLOSED -> OPEN to be valid")
	}
	if !JobStatusClosed.CanTransitionTo(JobStatusArchived) {
		t.Error("Expected CLOSED -> ARCHIVED to be valid")
	}
	if JobStatusClosed.CanTransitionTo(JobStatusDraft) {
		t.Error("Expected CLOSED -> DRAFT to be invalid")
	}

	// ARCHIVED transitions (terminal state)
	if JobStatusArchived.CanTransitionTo(JobStatusOpen) {
		t.Error("Expected ARCHIVED -> OPEN to be invalid")
	}
	if JobStatusArchived.CanTransitionTo(JobStatusDraft) {
		t.Error("Expected ARCHIVED -> DRAFT to be invalid")
	}
	if JobStatusArchived.CanTransitionTo(JobStatusClosed) {
		t.Error("Expected ARCHIVED -> CLOSED to be invalid")
	}

	// Same state transition (no-op)
	if !JobStatusOpen.CanTransitionTo(JobStatusOpen) {
		t.Error("Expected OPEN -> OPEN to be valid (idempotent)")
	}
}

func TestEmploymentType_Validation(t *testing.T) {
	valid := []EmploymentType{EmpFullTime, EmpPartTime, EmpContract, EmpInternship, EmpFreelance}
	for _, v := range valid {
		if !v.IsValid() {
			t.Errorf("Expected %s to be valid", v)
		}
	}

	if EmploymentType("INVALID_TYPE").IsValid() {
		t.Error("Expected INVALID_TYPE to be invalid")
	}
}

func TestRequirement_Enums(t *testing.T) {
	validCategories := []RequirementCategory{
		CategorySkill, CategoryExperience, CategoryEducation,
		CategoryCertification, CategoryLanguage, CategoryOther,
	}
	for _, c := range validCategories {
		if !c.IsValid() {
			t.Errorf("Expected category %s to be valid", c)
		}
	}
	if RequirementCategory("UNKNOWN").IsValid() {
		t.Error("Expected UNKNOWN category to be invalid")
	}

	if !ImportanceRequired.IsValid() || !ImportancePreferred.IsValid() {
		t.Error("Expected REQUIRED and PREFERRED to be valid importance")
	}
	if RequirementImportance("OPTIONAL").IsValid() {
		t.Error("Expected OPTIONAL importance to be invalid (must be REQUIRED or PREFERRED)")
	}
}
