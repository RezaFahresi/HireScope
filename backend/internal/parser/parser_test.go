package parser

import (
	"strings"
	"testing"
)

const sampleCV = `Budi Santoso
Senior Business Analyst
Jakarta, Indonesia
Email: budi.santoso@example.com | Phone: +62 812 3456 7890

SUMMARY
Experienced Business Analyst with over 7 years of background in financial technology and process re-engineering.

WORK EXPERIENCE
2021 - Present
Bank Central Asia
Senior Systems Analyst
Leading requirements elicitation for digital banking payments.

2018 - 2021
PT FinTech Solusindo
Junior Business Analyst
Assisted senior consultants in ERP process modeling and user acceptance testing.

EDUCATION
Universitas Indonesia
Bachelor of Computer Science
2014 - 2018

SKILLS
BPMN, SQL, PostgreSQL, Agile, Scrum, JIRA, Business Analysis, Requirements Gathering
`

func TestParser_FullExtraction(t *testing.T) {
	parsed := Parse(sampleCV)

	if parsed.Email != "budi.santoso@example.com" {
		t.Errorf("expected email 'budi.santoso@example.com', got '%s'", parsed.Email)
	}

	if !strings.Contains(parsed.Phone, "812") {
		t.Errorf("expected phone containing '812', got '%s'", parsed.Phone)
	}

	if parsed.FullName != "Budi Santoso" {
		t.Errorf("expected name 'Budi Santoso', got '%s'", parsed.FullName)
	}

	if !strings.Contains(parsed.Summary, "Experienced Business Analyst") {
		t.Errorf("expected summary, got '%s'", parsed.Summary)
	}

	// Verify Skills
	if len(parsed.Skills) < 5 {
		t.Errorf("expected at least 5 skills, got %d: %v", len(parsed.Skills), parsed.Skills)
	}
	foundSQL := false
	for _, s := range parsed.Skills {
		if strings.EqualFold(s, "SQL") {
			foundSQL = true
			break
		}
	}
	if !foundSQL {
		t.Errorf("expected skill 'SQL' in extracted skills: %v", parsed.Skills)
	}

	// Verify Educations
	if len(parsed.Educations) == 0 {
		t.Fatalf("expected education records, got 0")
	}
	edu := parsed.Educations[0]
	if !strings.Contains(edu.Institution, "Universitas Indonesia") {
		t.Errorf("expected institution 'Universitas Indonesia', got '%s'", edu.Institution)
	}

	// Verify Experiences
	if len(parsed.Experiences) < 2 {
		t.Fatalf("expected at least 2 experiences, got %d", len(parsed.Experiences))
	}
	exp1 := parsed.Experiences[0]
	if !exp1.IsCurrent {
		t.Errorf("expected first job to be current")
	}

	// Verify FieldsDetected
	if len(parsed.FieldsDetected) < 5 {
		t.Errorf("expected multiple detected fields, got: %v", parsed.FieldsDetected)
	}
}

func TestParser_EmptyAndMalformed(t *testing.T) {
	empty := Parse("")
	if len(empty.Educations) != 0 || len(empty.Skills) != 0 || len(empty.Experiences) != 0 {
		t.Errorf("expected empty slices for empty input")
	}

	noSections := Parse("Just some random text without any headings or emails.")
	if noSections.Email != "" || len(noSections.Educations) != 0 {
		t.Errorf("expected no false extractions for plain unstructured text")
	}
}

func TestParser_MixedCaseHeadings(t *testing.T) {
	text := `Jane Doe
jane.doe@example.org

tEcHnIcAl SkIlLs
Go, Python, Docker

eDuCaTiOn:
Bandung Institute of Technology
Bachelor of Science
2015 - 2019
`
	parsed := Parse(text)
	if len(parsed.Skills) != 3 {
		t.Errorf("expected 3 skills from mixed-case heading, got %d: %v", len(parsed.Skills), parsed.Skills)
	}
	if len(parsed.Educations) != 1 {
		t.Errorf("expected 1 education from mixed-case heading, got %d", len(parsed.Educations))
	}
}
