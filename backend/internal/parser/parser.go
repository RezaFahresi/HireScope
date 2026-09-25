package parser

import (
	"regexp"
	"strings"
	"unicode"

	"hirescope/backend/internal/model"
)

// SectionType categorizes detected document sections.
type SectionType string

const (
	SectionProfile        SectionType = "PROFILE"
	SectionExperience     SectionType = "EXPERIENCE"
	SectionEducation      SectionType = "EDUCATION"
	SectionSkills         SectionType = "SKILLS"
	SectionCertifications SectionType = "CERTIFICATIONS"
	SectionOther          SectionType = "OTHER"
)

// ParsedEducation represents an extracted educational credential.
type ParsedEducation struct {
	Institution  string  `json:"institution"`
	Degree       string  `json:"degree,omitempty"`
	FieldOfStudy string  `json:"field_of_study,omitempty"`
	StartDate    *string `json:"start_date,omitempty"`
	EndDate      *string `json:"end_date,omitempty"`
	Description  string  `json:"description,omitempty"`
}

// ParsedExperience represents an extracted professional work history entry.
type ParsedExperience struct {
	Company        string                `json:"company"`
	Position       string                `json:"position"`
	Location       string                `json:"location,omitempty"`
	EmploymentType *model.EmploymentType `json:"employment_type,omitempty"`
	StartDate      *string               `json:"start_date,omitempty"`
	EndDate        *string               `json:"end_date,omitempty"`
	IsCurrent      bool                  `json:"is_current"`
	Description    string                `json:"description,omitempty"`
}

// ParsedCV contains all structured data deterministically extracted from a CV.
type ParsedCV struct {
	FullName       string             `json:"full_name,omitempty"`
	Email          string             `json:"email,omitempty"`
	Phone          string             `json:"phone,omitempty"`
	Location       string             `json:"location,omitempty"`
	Headline       string             `json:"headline,omitempty"`
	Summary        string             `json:"summary,omitempty"`
	Educations     []ParsedEducation  `json:"educations"`
	Experiences    []ParsedExperience `json:"experiences"`
	Skills         []string           `json:"skills"`
	FieldsDetected []string           `json:"fields_detected"`
}

var (
	emailRegex = regexp.MustCompile(`(?i)[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}`)
	phoneRegex = regexp.MustCompile(`(?:(?:\+|00)\d{1,3}[-.\s]?)?\(?\d{2,4}\)?[-.\s]?\d{3,4}[-.\s]?\d{3,6}`)

	// Year or Month-Year range regex: e.g., "2019 - 2023", "Jan 2020 - Present", "08/2018 - 05/2022"
	dateRangeRegex = regexp.MustCompile(`(?i)(?:(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*[\s,./]*)?(\b(?:19|20)\d{2}\b)\s*[-–—to]+\s*(?:(?:(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*[\s,./]*)?(\b(?:19|20)\d{2}\b)|(Present|Current|Now))`)

	degreeKeywords = []string{
		"Bachelor", "Master", "Doctor", "PhD", "Ph.D", "B.Sc", "M.Sc", "B.S.", "M.S.",
		"B.A.", "M.A.", "B.Tech", "M.Tech", "S.Kom", "S.T.", "S.E.", "S.Si", "Diploma",
		"Associate", "Sarjana", "Magister", "Doktor",
	}

	institutionKeywords = []string{
		"University", "Universitas", "College", "Institute", "Institut", "Politeknik",
		"Polytechnic", "Academy", "Akademi", "School", "Sekolah",
	}
)

// Parse deterministic extraction from normalized CV text.
func Parse(text string) *ParsedCV {
	result := &ParsedCV{
		Educations:     []ParsedEducation{},
		Experiences:    []ParsedExperience{},
		Skills:         []string{},
		FieldsDetected: []string{},
	}

	if strings.TrimSpace(text) == "" {
		return result
	}

	// 1. Extract Email
	if emailMatch := emailRegex.FindString(text); emailMatch != "" {
		result.Email = strings.ToLower(emailMatch)
		result.FieldsDetected = append(result.FieldsDetected, "email")
	}

	// 2. Extract Phone
	if phoneMatch := phoneRegex.FindString(text); phoneMatch != "" {
		digits := extractDigits(phoneMatch)
		if len(digits) >= 8 && len(digits) <= 15 {
			result.Phone = strings.TrimSpace(phoneMatch)
			result.FieldsDetected = append(result.FieldsDetected, "phone")
		}
	}

	// 3. Segment into sections
	sections, headerLines := segmentSections(text)

	// 4. Candidate Name Heuristic (from headerLines prior to first section heading)
	if name := detectCandidateName(headerLines); name != "" {
		result.FullName = name
		result.FieldsDetected = append(result.FieldsDetected, "full_name")
	}

	// 5. Headline heuristic (lines right under name, before first section)
	if headline := detectHeadline(headerLines, result.FullName); headline != "" {
		result.Headline = headline
		result.FieldsDetected = append(result.FieldsDetected, "headline")
	}

	// 6. Parse Profile / Summary section
	if profLines, ok := sections[SectionProfile]; ok {
		summary := strings.TrimSpace(strings.Join(profLines, " "))
		if len(summary) > 1000 {
			summary = summary[:1000]
		}
		if summary != "" {
			result.Summary = summary
			result.FieldsDetected = append(result.FieldsDetected, "summary")
		}
	}

	// 7. Parse Skills section
	if skillLines, ok := sections[SectionSkills]; ok {
		skills := parseSkills(skillLines)
		if len(skills) > 0 {
			result.Skills = skills
			result.FieldsDetected = append(result.FieldsDetected, "skills")
		}
	}

	// 8. Parse Education section
	if eduLines, ok := sections[SectionEducation]; ok {
		edus := parseEducations(eduLines)
		if len(edus) > 0 {
			result.Educations = edus
			result.FieldsDetected = append(result.FieldsDetected, "education")
		}
	}

	// 9. Parse Experience section
	if expLines, ok := sections[SectionExperience]; ok {
		exps := parseExperiences(expLines)
		if len(exps) > 0 {
			result.Experiences = exps
			result.FieldsDetected = append(result.FieldsDetected, "experience")
		}
	}

	return result
}

// segmentSections splits text by detected section headings.
func segmentSections(text string) (map[SectionType][]string, []string) {
	lines := strings.Split(text, "\n")
	sections := make(map[SectionType][]string)
	var headerLines []string
	currentSection := SectionOther
	firstSectionFound := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if secType, isHeading := identifyHeading(trimmed); isHeading {
			currentSection = secType
			firstSectionFound = true
			continue
		}

		if !firstSectionFound {
			headerLines = append(headerLines, trimmed)
		} else if currentSection != SectionOther {
			sections[currentSection] = append(sections[currentSection], trimmed)
		}
	}

	return sections, headerLines
}

// identifyHeading checks if a line corresponds to a standard CV section heading.
func identifyHeading(line string) (SectionType, bool) {
	// Strip trailing punctuation like colons
	cleaned := strings.Trim(line, ": \t#*-")
	upper := strings.ToUpper(cleaned)

	// Headings should be short (1-4 words)
	if len(strings.Fields(upper)) > 4 || len(upper) > 40 {
		return SectionOther, false
	}

	switch {
	case upper == "PROFILE" || upper == "SUMMARY" || upper == "ABOUT ME" ||
		upper == "PROFESSIONAL SUMMARY" || upper == "EXECUTIVE SUMMARY" ||
		upper == "CAREER OBJECTIVE" || upper == "BIOGRAPHY":
		return SectionProfile, true

	case upper == "EXPERIENCE" || upper == "WORK EXPERIENCE" ||
		upper == "PROFESSIONAL EXPERIENCE" || upper == "EMPLOYMENT HISTORY" ||
		upper == "WORK HISTORY" || upper == "EXPERIENCES":
		return SectionExperience, true

	case upper == "EDUCATION" || upper == "EDUCATIONAL BACKGROUND" ||
		upper == "ACADEMIC BACKGROUND" || upper == "EDUCATION & QUALIFICATIONS" ||
		upper == "ACADEMIC HISTORY":
		return SectionEducation, true

	case upper == "SKILLS" || upper == "TECHNICAL SKILLS" ||
		upper == "CORE SKILLS" || upper == "KEY SKILLS" ||
		upper == "SKILLS & EXPERTISE" || upper == "COMPETENCIES" ||
		upper == "AREAS OF EXPERTISE":
		return SectionSkills, true

	case upper == "CERTIFICATIONS" || upper == "CERTIFICATION" ||
		upper == "LICENSES & CERTIFICATIONS":
		return SectionCertifications, true
	}

	return SectionOther, false
}

// detectCandidateName inspects header lines conservatively for a candidate name.
func detectCandidateName(lines []string) string {
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Ignore if contains email or phone indicators
		if strings.Contains(trimmed, "@") || strings.Contains(trimmed, "+") || strings.Contains(trimmed, "www.") {
			continue
		}
		words := strings.Fields(trimmed)
		// Names typically consist of 2 to 4 words
		if len(words) >= 2 && len(words) <= 4 {
			allAlpha := true
			for _, w := range words {
				// Strip dots like in "S.Kom" or initials "J."
				cleanWord := strings.Trim(w, ".")
				for _, r := range cleanWord {
					if !unicode.IsLetter(r) && r != '\'' && r != '-' {
						allAlpha = false
						break
					}
				}
			}
			if allAlpha && len(trimmed) <= 50 {
				return trimmed
			}
		}
	}
	return ""
}

// detectHeadline looks for a role or title line right below name in header.
func detectHeadline(lines []string, name string) string {
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == name || strings.Contains(trimmed, "@") || strings.Contains(trimmed, "+") {
			continue
		}
		if len(trimmed) > 3 && len(trimmed) < 80 {
			// Check if looks like a title (e.g. "Senior Business Analyst")
			return trimmed
		}
	}
	return ""
}

// parseSkills parses comma, bullet, or newline separated skill tokens.
func parseSkills(lines []string) []string {
	var skills []string
	seen := make(map[string]bool)

	for _, line := range lines {
		// Split by commas, bullets, pipes, or semicolons
		tokens := strings.FieldsFunc(line, func(r rune) bool {
			return r == ',' || r == '•' || r == '|' || r == ';' || r == '*'
		})

		for _, t := range tokens {
			s := strings.Trim(t, " \t-•*#")
			if s == "" || len(s) < 2 || len(s) > 60 {
				continue
			}

			// Exclude common headings/noise phrases
			lower := strings.ToLower(s)
			if strings.HasPrefix(lower, "proficient in") || strings.HasPrefix(lower, "experience with") ||
				strings.HasPrefix(lower, "skills include") {
				continue
			}

			if !seen[lower] {
				seen[lower] = true
				skills = append(skills, s)
			}
		}
	}

	return skills
}

// parseEducations extracts institution, degree, and dates from education section.
func parseEducations(lines []string) []ParsedEducation {
	var edus []ParsedEducation

	var currentEdu *ParsedEducation

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Check if line contains an institution keyword
		hasInst := false
		for _, kw := range institutionKeywords {
			if strings.Contains(strings.ToLower(trimmed), strings.ToLower(kw)) {
				hasInst = true
				break
			}
		}

		// Check for date range
		dateMatch := dateRangeRegex.FindStringSubmatch(trimmed)

		if hasInst {
			// Finish previous education if present
			if currentEdu != nil && currentEdu.Institution != "" {
				edus = append(edus, *currentEdu)
			}
			currentEdu = &ParsedEducation{
				Institution: trimmed,
			}
			if len(dateMatch) > 0 {
				start, end := extractDatesFromMatch(dateMatch)
				currentEdu.StartDate = start
				currentEdu.EndDate = end
			}
			continue
		}

		if currentEdu != nil {
			if len(dateMatch) > 0 && currentEdu.StartDate == nil {
				start, end := extractDatesFromMatch(dateMatch)
				currentEdu.StartDate = start
				currentEdu.EndDate = end
			}

			// Check for degree
			for _, dkw := range degreeKeywords {
				if strings.Contains(strings.ToLower(trimmed), strings.ToLower(dkw)) {
					currentEdu.Degree = trimmed
					break
				}
			}
		}
	}

	if currentEdu != nil && currentEdu.Institution != "" {
		edus = append(edus, *currentEdu)
	}

	return edus
}

// parseExperiences extracts professional experience records.
func parseExperiences(lines []string) []ParsedExperience {
	var exps []ParsedExperience
	var currentExp *ParsedExperience

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		dateMatch := dateRangeRegex.FindStringSubmatch(trimmed)
		if len(dateMatch) > 0 {
			// Date line signals a new or continuing job entry
			if currentExp != nil && currentExp.Company != "" {
				exps = append(exps, *currentExp)
			}
			start, end := extractDatesFromMatch(dateMatch)
			isCurr := strings.Contains(strings.ToLower(trimmed), "present") || strings.Contains(strings.ToLower(trimmed), "current")

			// The line might also contain the company or title
			remainder := strings.TrimSpace(dateRangeRegex.ReplaceAllString(trimmed, ""))

			currentExp = &ParsedExperience{
				StartDate: start,
				EndDate:   end,
				IsCurrent: isCurr,
			}
			if remainder != "" {
				currentExp.Company = remainder
			}
			continue
		}

		if currentExp != nil {
			if currentExp.Company == "" {
				currentExp.Company = trimmed
			} else if currentExp.Position == "" {
				currentExp.Position = trimmed
			} else {
				if currentExp.Description == "" {
					currentExp.Description = trimmed
				} else {
					currentExp.Description += " " + trimmed
				}
			}
		}
	}

	if currentExp != nil && (currentExp.Company != "" || currentExp.Position != "") {
		if currentExp.Company == "" {
			currentExp.Company = "Company"
		}
		if currentExp.Position == "" {
			currentExp.Position = "Role"
		}
		exps = append(exps, *currentExp)
	}

	return exps
}

// extractDatesFromMatch extracts formatted YYYY-MM-DD from regex submatches.
func extractDatesFromMatch(match []string) (*string, *string) {
	if len(match) < 3 {
		return nil, nil
	}

	startYear := match[2]
	var startDate *string
	if startYear != "" {
		d := startYear + "-01-01"
		startDate = &d
	}

	var endDate *string
	endYear := match[4]
	if endYear != "" {
		d := endYear + "-01-01"
		endDate = &d
	}

	return startDate, endDate
}

func extractDigits(s string) string {
	var sb strings.Builder
	for _, r := range s {
		if unicode.IsDigit(r) {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}
