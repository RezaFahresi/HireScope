package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"hirescope/backend/internal/extractor"
	"hirescope/backend/internal/model"
	"hirescope/backend/internal/normalizer"
	"hirescope/backend/internal/parser"
	"hirescope/backend/internal/repository"
	"hirescope/backend/internal/storage"
)

var (
	ErrFullNameRequired          = errors.New("full name is required")
	ErrCompanyRequired           = errors.New("company name is required")
	ErrPositionRequired          = errors.New("position is required")
	ErrInstitutionRequired       = errors.New("institution is required")
	ErrSkillRequired             = errors.New("skill name is required")
	ErrRawTextRequired           = errors.New("raw text is required for TEXT CV source")
	ErrTextTooLarge              = errors.New("CV text exceeds maximum allowed size (500 KB)")
	ErrDateOrder                 = errors.New("end date cannot precede start date")
	ErrFileTooLarge              = errors.New("file exceeds maximum allowed size (20 MB)")
	ErrDocumentCandidateMismatch = errors.New("document does not belong to the specified candidate")
	ErrStoragePathMissing        = errors.New("document storage path is missing")
)

const (
	MaxCVTextLength   = 500000           // 500 KB characters for direct text
	MaxUploadFileSize = 20 * 1024 * 1024 // 20 MB binary upload limit
)

// CreateCandidateInput encapsulates parameters for registering a candidate.
type CreateCandidateInput struct {
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Location string `json:"location"`
	Headline string `json:"headline"`
	Summary  string `json:"summary"`
}

// UpdateCandidateInput encapsulates parameters for updating candidate details.
type UpdateCandidateInput struct {
	FullName *string `json:"full_name"`
	Email    *string `json:"email"`
	Phone    *string `json:"phone"`
	Location *string `json:"location"`
	Headline *string `json:"headline"`
	Summary  *string `json:"summary"`
}

// CreateEducationInput encapsulates parameters for adding an education record.
type CreateEducationInput struct {
	Institution  string  `json:"institution"`
	Degree       string  `json:"degree"`
	FieldOfStudy string  `json:"field_of_study"`
	StartDate    *string `json:"start_date"`
	EndDate      *string `json:"end_date"`
	Description  string  `json:"description"`
}

// UpdateEducationInput encapsulates parameters for updating an education record.
type UpdateEducationInput struct {
	Institution  *string `json:"institution"`
	Degree       *string `json:"degree"`
	FieldOfStudy *string `json:"field_of_study"`
	StartDate    *string `json:"start_date"`
	EndDate      *string `json:"end_date"`
	Description  *string `json:"description"`
}

// CreateExperienceInput encapsulates parameters for adding work experience.
type CreateExperienceInput struct {
	Company        string                `json:"company"`
	Position       string                `json:"position"`
	Location       string                `json:"location"`
	EmploymentType *model.EmploymentType `json:"employment_type"`
	StartDate      *string               `json:"start_date"`
	EndDate        *string               `json:"end_date"`
	IsCurrent      bool                  `json:"is_current"`
	Description    string                `json:"description"`
}

// UpdateExperienceInput encapsulates parameters for updating work experience.
type UpdateExperienceInput struct {
	Company        *string               `json:"company"`
	Position       *string               `json:"position"`
	Location       *string               `json:"location"`
	EmploymentType *model.EmploymentType `json:"employment_type"`
	StartDate      *string               `json:"start_date"`
	EndDate        *string               `json:"end_date"`
	IsCurrent      *bool                 `json:"is_current"`
	Description    *string               `json:"description"`
}

// CreateSkillInput encapsulates parameters for adding a skill.
type CreateSkillInput struct {
	Skill string `json:"skill"`
}

// UpdateSkillInput encapsulates parameters for updating a skill.
type UpdateSkillInput struct {
	Skill string `json:"skill"`
}

// CreateDocumentInput encapsulates parameters for adding a CV document/text.
type CreateDocumentInput struct {
	SourceType model.DocumentSourceType `json:"source_type"`
	RawText    string                   `json:"raw_text"`
}

// ExtractionMeta contains technical metadata about text extraction.
type ExtractionMeta struct {
	Status     string `json:"status"`
	Characters int    `json:"characters"`
}

// ParsingMeta contains metadata about detected entities during parsing.
type ParsingMeta struct {
	Status         string   `json:"status"`
	FieldsDetected []string `json:"fields_detected"`
}

// ProcessDocumentResult encapsulates the result of CV extraction, normalization, and parsing.
type ProcessDocumentResult struct {
	DocumentID string           `json:"document_id"`
	SourceType string           `json:"source_type"`
	Extraction ExtractionMeta   `json:"extraction"`
	Parsing    ParsingMeta      `json:"parsing"`
	Candidate  *model.Candidate `json:"candidate"`
}

// CandidateService defines the interface for candidate management business operations.
type CandidateService interface {
	CreateCandidate(ctx context.Context, creatorID string, input CreateCandidateInput) (*model.Candidate, error)
	GetCandidateByID(ctx context.Context, id string) (*model.Candidate, error)
	ListCandidates(ctx context.Context, filter repository.CandidateFilter) (*repository.CandidateListResult, error)
	UpdateCandidate(ctx context.Context, id string, input UpdateCandidateInput) (*model.Candidate, error)

	// Job-Candidate associations
	AddCandidateToJob(ctx context.Context, jobID, candidateID string) error
	RemoveCandidateFromJob(ctx context.Context, jobID, candidateID string) error
	ListCandidatesByJob(ctx context.Context, jobID string, page, limit int) (*repository.CandidateListResult, error)
	ListJobsByCandidate(ctx context.Context, candidateID string) ([]model.Job, error)

	// Educations
	CreateEducation(ctx context.Context, candidateID string, input CreateEducationInput) (*model.CandidateEducation, error)
	ListEducations(ctx context.Context, candidateID string) ([]model.CandidateEducation, error)
	UpdateEducation(ctx context.Context, candidateID, eduID string, input UpdateEducationInput) (*model.CandidateEducation, error)
	DeleteEducation(ctx context.Context, candidateID, eduID string) error

	// Experiences
	CreateExperience(ctx context.Context, candidateID string, input CreateExperienceInput) (*model.CandidateExperience, error)
	ListExperiences(ctx context.Context, candidateID string) ([]model.CandidateExperience, error)
	UpdateExperience(ctx context.Context, candidateID, expID string, input UpdateExperienceInput) (*model.CandidateExperience, error)
	DeleteExperience(ctx context.Context, candidateID, expID string) error

	// Skills
	CreateSkill(ctx context.Context, candidateID string, input CreateSkillInput) (*model.CandidateSkill, error)
	ListSkills(ctx context.Context, candidateID string) ([]model.CandidateSkill, error)
	UpdateSkill(ctx context.Context, candidateID, skillID string, input UpdateSkillInput) (*model.CandidateSkill, error)
	DeleteSkill(ctx context.Context, candidateID, skillID string) error

	// Documents
	CreateDocument(ctx context.Context, candidateID string, input CreateDocumentInput) (*model.CandidateDocument, error)
	UploadDocument(ctx context.Context, candidateID, originalFilename, mimeType string, size int64, reader io.Reader) (*model.CandidateDocument, error)
	ProcessDocument(ctx context.Context, candidateID, docID string) (*ProcessDocumentResult, error)
	GetDocumentByID(ctx context.Context, candidateID, docID string) (*model.CandidateDocument, error)
	ListDocuments(ctx context.Context, candidateID string) ([]model.CandidateDocument, error)
	DeleteDocument(ctx context.Context, candidateID, docID string) error
}

type candidateService struct {
	candidateRepo  repository.CandidateRepository
	jobRepo        repository.JobRepository
	storageService storage.StorageService
	pdfExtractor   extractor.TextExtractor
	docxExtractor  extractor.TextExtractor
}

// NewCandidateService initializes a CandidateService instance with optional storage and extractor components.
func NewCandidateService(
	candidateRepo repository.CandidateRepository,
	jobRepo repository.JobRepository,
	storageService storage.StorageService,
	pdfExtractor extractor.TextExtractor,
	docxExtractor extractor.TextExtractor,
) CandidateService {
	if pdfExtractor == nil {
		pdfExtractor = extractor.NewPDFTextExtractor()
	}
	if docxExtractor == nil {
		docxExtractor = extractor.NewDOCXTextExtractor()
	}
	return &candidateService{
		candidateRepo:  candidateRepo,
		jobRepo:        jobRepo,
		storageService: storageService,
		pdfExtractor:   pdfExtractor,
		docxExtractor:  docxExtractor,
	}
}

func (s *candidateService) CreateCandidate(ctx context.Context, creatorID string, input CreateCandidateInput) (*model.Candidate, error) {
	name := strings.TrimSpace(input.FullName)
	if name == "" {
		return nil, ErrFullNameRequired
	}
	if len(name) > 255 {
		return nil, errors.New("full name exceeds maximum length of 255 characters")
	}

	email := strings.TrimSpace(input.Email)
	if email != "" {
		if !strings.Contains(email, "@") || !strings.Contains(email, ".") {
			return nil, ErrInvalidEmailFormat
		}
		email = model.NormalizeEmail(email)
	}

	candidate := &model.Candidate{
		FullName:  name,
		Email:     email,
		Phone:     strings.TrimSpace(input.Phone),
		Location:  strings.TrimSpace(input.Location),
		Headline:  strings.TrimSpace(input.Headline),
		Summary:   strings.TrimSpace(input.Summary),
		CreatedBy: creatorID, // Always derived from authenticated JWT context
	}

	if err := s.candidateRepo.Create(ctx, candidate); err != nil {
		return nil, err
	}

	return s.candidateRepo.GetByID(ctx, candidate.ID, true)
}

func (s *candidateService) GetCandidateByID(ctx context.Context, id string) (*model.Candidate, error) {
	if strings.TrimSpace(id) == "" {
		return nil, repository.ErrCandidateNotFound
	}
	return s.candidateRepo.GetByID(ctx, id, true)
}

func (s *candidateService) ListCandidates(ctx context.Context, filter repository.CandidateFilter) (*repository.CandidateListResult, error) {
	return s.candidateRepo.List(ctx, filter)
}

func (s *candidateService) UpdateCandidate(ctx context.Context, id string, input UpdateCandidateInput) (*model.Candidate, error) {
	candidate, err := s.candidateRepo.GetByID(ctx, id, false)
	if err != nil {
		return nil, err
	}

	if input.FullName != nil {
		name := strings.TrimSpace(*input.FullName)
		if name == "" {
			return nil, ErrFullNameRequired
		}
		if len(name) > 255 {
			return nil, errors.New("full name exceeds maximum length of 255 characters")
		}
		candidate.FullName = name
	}

	if input.Email != nil {
		email := strings.TrimSpace(*input.Email)
		if email != "" {
			if !strings.Contains(email, "@") || !strings.Contains(email, ".") {
				return nil, ErrInvalidEmailFormat
			}
			candidate.Email = model.NormalizeEmail(email)
		} else {
			candidate.Email = ""
		}
	}

	if input.Phone != nil {
		candidate.Phone = strings.TrimSpace(*input.Phone)
	}
	if input.Location != nil {
		candidate.Location = strings.TrimSpace(*input.Location)
	}
	if input.Headline != nil {
		candidate.Headline = strings.TrimSpace(*input.Headline)
	}
	if input.Summary != nil {
		candidate.Summary = strings.TrimSpace(*input.Summary)
	}

	if err := s.candidateRepo.Update(ctx, candidate); err != nil {
		return nil, err
	}

	return s.candidateRepo.GetByID(ctx, candidate.ID, true)
}

func (s *candidateService) AddCandidateToJob(ctx context.Context, jobID, candidateID string) error {
	// Verify job exists
	if _, err := s.jobRepo.GetByID(ctx, jobID, false); err != nil {
		return err
	}

	// Verify candidate exists
	if _, err := s.candidateRepo.GetByID(ctx, candidateID, false); err != nil {
		return err
	}

	return s.candidateRepo.AddJobCandidate(ctx, jobID, candidateID)
}

func (s *candidateService) RemoveCandidateFromJob(ctx context.Context, jobID, candidateID string) error {
	// Verify job exists
	if _, err := s.jobRepo.GetByID(ctx, jobID, false); err != nil {
		return err
	}

	// Verify candidate exists
	if _, err := s.candidateRepo.GetByID(ctx, candidateID, false); err != nil {
		return err
	}

	return s.candidateRepo.RemoveJobCandidate(ctx, jobID, candidateID)
}

func (s *candidateService) ListCandidatesByJob(ctx context.Context, jobID string, page, limit int) (*repository.CandidateListResult, error) {
	if _, err := s.jobRepo.GetByID(ctx, jobID, false); err != nil {
		return nil, err
	}
	return s.candidateRepo.ListCandidatesByJob(ctx, jobID, page, limit)
}

func (s *candidateService) ListJobsByCandidate(ctx context.Context, candidateID string) ([]model.Job, error) {
	if _, err := s.candidateRepo.GetByID(ctx, candidateID, false); err != nil {
		return nil, err
	}
	return s.candidateRepo.ListJobsByCandidate(ctx, candidateID)
}

func (s *candidateService) CreateEducation(ctx context.Context, candidateID string, input CreateEducationInput) (*model.CandidateEducation, error) {
	if _, err := s.candidateRepo.GetByID(ctx, candidateID, false); err != nil {
		return nil, err
	}

	inst := strings.TrimSpace(input.Institution)
	if inst == "" {
		return nil, ErrInstitutionRequired
	}

	startDate, err := parseDate(input.StartDate)
	if err != nil {
		return nil, err
	}
	endDate, err := parseDate(input.EndDate)
	if err != nil {
		return nil, err
	}

	if startDate != nil && endDate != nil && endDate.Before(*startDate) {
		return nil, ErrDateOrder
	}

	edu := &model.CandidateEducation{
		CandidateID:  candidateID,
		Institution:  inst,
		Degree:       strings.TrimSpace(input.Degree),
		FieldOfStudy: strings.TrimSpace(input.FieldOfStudy),
		StartDate:    startDate,
		EndDate:      endDate,
		Description:  strings.TrimSpace(input.Description),
	}

	if err := s.candidateRepo.CreateEducation(ctx, edu); err != nil {
		return nil, err
	}

	return edu, nil
}

func (s *candidateService) ListEducations(ctx context.Context, candidateID string) ([]model.CandidateEducation, error) {
	if _, err := s.candidateRepo.GetByID(ctx, candidateID, false); err != nil {
		return nil, err
	}
	return s.candidateRepo.ListEducations(ctx, candidateID)
}

func (s *candidateService) UpdateEducation(ctx context.Context, candidateID, eduID string, input UpdateEducationInput) (*model.CandidateEducation, error) {
	if _, err := s.candidateRepo.GetByID(ctx, candidateID, false); err != nil {
		return nil, err
	}

	edu, err := s.candidateRepo.GetEducationByID(ctx, eduID)
	if err != nil {
		return nil, err
	}
	if edu.CandidateID != candidateID {
		return nil, repository.ErrEducationNotFound
	}

	if input.Institution != nil {
		inst := strings.TrimSpace(*input.Institution)
		if inst == "" {
			return nil, ErrInstitutionRequired
		}
		edu.Institution = inst
	}
	if input.Degree != nil {
		edu.Degree = strings.TrimSpace(*input.Degree)
	}
	if input.FieldOfStudy != nil {
		edu.FieldOfStudy = strings.TrimSpace(*input.FieldOfStudy)
	}
	if input.Description != nil {
		edu.Description = strings.TrimSpace(*input.Description)
	}

	if input.StartDate != nil {
		d, err := parseDate(input.StartDate)
		if err != nil {
			return nil, err
		}
		edu.StartDate = d
	}
	if input.EndDate != nil {
		d, err := parseDate(input.EndDate)
		if err != nil {
			return nil, err
		}
		edu.EndDate = d
	}

	if edu.StartDate != nil && edu.EndDate != nil && edu.EndDate.Before(*edu.StartDate) {
		return nil, ErrDateOrder
	}

	if err := s.candidateRepo.UpdateEducation(ctx, edu); err != nil {
		return nil, err
	}

	return edu, nil
}

func (s *candidateService) DeleteEducation(ctx context.Context, candidateID, eduID string) error {
	if _, err := s.candidateRepo.GetByID(ctx, candidateID, false); err != nil {
		return err
	}
	return s.candidateRepo.DeleteEducation(ctx, candidateID, eduID)
}

func (s *candidateService) CreateExperience(ctx context.Context, candidateID string, input CreateExperienceInput) (*model.CandidateExperience, error) {
	if _, err := s.candidateRepo.GetByID(ctx, candidateID, false); err != nil {
		return nil, err
	}

	company := strings.TrimSpace(input.Company)
	if company == "" {
		return nil, ErrCompanyRequired
	}
	position := strings.TrimSpace(input.Position)
	if position == "" {
		return nil, ErrPositionRequired
	}

	if input.EmploymentType != nil && !input.EmploymentType.IsValid() {
		return nil, ErrInvalidEmploymentType
	}

	startDate, err := parseDate(input.StartDate)
	if err != nil {
		return nil, err
	}
	endDate, err := parseDate(input.EndDate)
	if err != nil {
		return nil, err
	}

	if startDate != nil && endDate != nil && !input.IsCurrent && endDate.Before(*startDate) {
		return nil, ErrDateOrder
	}

	exp := &model.CandidateExperience{
		CandidateID:    candidateID,
		Company:        company,
		Position:       position,
		Location:       strings.TrimSpace(input.Location),
		EmploymentType: input.EmploymentType,
		StartDate:      startDate,
		EndDate:        endDate,
		IsCurrent:      input.IsCurrent,
		Description:    strings.TrimSpace(input.Description),
	}

	if err := s.candidateRepo.CreateExperience(ctx, exp); err != nil {
		return nil, err
	}

	return exp, nil
}

func (s *candidateService) ListExperiences(ctx context.Context, candidateID string) ([]model.CandidateExperience, error) {
	if _, err := s.candidateRepo.GetByID(ctx, candidateID, false); err != nil {
		return nil, err
	}
	return s.candidateRepo.ListExperiences(ctx, candidateID)
}

func (s *candidateService) UpdateExperience(ctx context.Context, candidateID, expID string, input UpdateExperienceInput) (*model.CandidateExperience, error) {
	if _, err := s.candidateRepo.GetByID(ctx, candidateID, false); err != nil {
		return nil, err
	}

	exp, err := s.candidateRepo.GetExperienceByID(ctx, expID)
	if err != nil {
		return nil, err
	}
	if exp.CandidateID != candidateID {
		return nil, repository.ErrExperienceNotFound
	}

	if input.Company != nil {
		comp := strings.TrimSpace(*input.Company)
		if comp == "" {
			return nil, ErrCompanyRequired
		}
		exp.Company = comp
	}
	if input.Position != nil {
		pos := strings.TrimSpace(*input.Position)
		if pos == "" {
			return nil, ErrPositionRequired
		}
		exp.Position = pos
	}
	if input.Location != nil {
		exp.Location = strings.TrimSpace(*input.Location)
	}
	if input.EmploymentType != nil {
		if !input.EmploymentType.IsValid() {
			return nil, ErrInvalidEmploymentType
		}
		exp.EmploymentType = input.EmploymentType
	}
	if input.IsCurrent != nil {
		exp.IsCurrent = *input.IsCurrent
	}
	if input.Description != nil {
		exp.Description = strings.TrimSpace(*input.Description)
	}

	if input.StartDate != nil {
		d, err := parseDate(input.StartDate)
		if err != nil {
			return nil, err
		}
		exp.StartDate = d
	}
	if input.EndDate != nil {
		d, err := parseDate(input.EndDate)
		if err != nil {
			return nil, err
		}
		exp.EndDate = d
	}

	if exp.StartDate != nil && exp.EndDate != nil && !exp.IsCurrent && exp.EndDate.Before(*exp.StartDate) {
		return nil, ErrDateOrder
	}

	if err := s.candidateRepo.UpdateExperience(ctx, exp); err != nil {
		return nil, err
	}

	return exp, nil
}

func (s *candidateService) DeleteExperience(ctx context.Context, candidateID, expID string) error {
	if _, err := s.candidateRepo.GetByID(ctx, candidateID, false); err != nil {
		return err
	}
	return s.candidateRepo.DeleteExperience(ctx, candidateID, expID)
}

func (s *candidateService) CreateSkill(ctx context.Context, candidateID string, input CreateSkillInput) (*model.CandidateSkill, error) {
	if _, err := s.candidateRepo.GetByID(ctx, candidateID, false); err != nil {
		return nil, err
	}

	name := strings.TrimSpace(input.Skill)
	if name == "" {
		return nil, ErrSkillRequired
	}
	if len(name) > 100 {
		return nil, errors.New("skill name exceeds maximum length of 100 characters")
	}

	normalized := strings.ToLower(name)
	exists, err := s.candidateRepo.HasSkill(ctx, candidateID, normalized, "")
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, repository.ErrDuplicateSkill
	}

	skill := &model.CandidateSkill{
		CandidateID:     candidateID,
		Skill:           name,
		NormalizedSkill: normalized,
	}

	if err := s.candidateRepo.CreateSkill(ctx, skill); err != nil {
		return nil, err
	}

	return skill, nil
}

func (s *candidateService) ListSkills(ctx context.Context, candidateID string) ([]model.CandidateSkill, error) {
	if _, err := s.candidateRepo.GetByID(ctx, candidateID, false); err != nil {
		return nil, err
	}
	return s.candidateRepo.ListSkills(ctx, candidateID)
}

func (s *candidateService) UpdateSkill(ctx context.Context, candidateID, skillID string, input UpdateSkillInput) (*model.CandidateSkill, error) {
	if _, err := s.candidateRepo.GetByID(ctx, candidateID, false); err != nil {
		return nil, err
	}

	skill, err := s.candidateRepo.GetSkillByID(ctx, skillID)
	if err != nil {
		return nil, err
	}
	if skill.CandidateID != candidateID {
		return nil, repository.ErrSkillNotFound
	}

	name := strings.TrimSpace(input.Skill)
	if name == "" {
		return nil, ErrSkillRequired
	}
	if len(name) > 100 {
		return nil, errors.New("skill name exceeds maximum length of 100 characters")
	}

	normalized := strings.ToLower(name)
	exists, err := s.candidateRepo.HasSkill(ctx, candidateID, normalized, skillID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, repository.ErrDuplicateSkill
	}

	skill.Skill = name
	skill.NormalizedSkill = normalized

	if err := s.candidateRepo.UpdateSkill(ctx, skill); err != nil {
		return nil, err
	}

	return skill, nil
}

func (s *candidateService) DeleteSkill(ctx context.Context, candidateID, skillID string) error {
	if _, err := s.candidateRepo.GetByID(ctx, candidateID, false); err != nil {
		return err
	}
	return s.candidateRepo.DeleteSkill(ctx, candidateID, skillID)
}

func (s *candidateService) CreateDocument(ctx context.Context, candidateID string, input CreateDocumentInput) (*model.CandidateDocument, error) {
	if _, err := s.candidateRepo.GetByID(ctx, candidateID, false); err != nil {
		return nil, err
	}

	if input.SourceType == model.SourceTypePDF || input.SourceType == model.SourceTypeDOCX {
		return nil, errors.New("direct JSON creation is only supported for TEXT CV source; please use the upload endpoint for PDF and DOCX documents")
	}

	if input.SourceType != model.SourceTypeText {
		return nil, errors.New("invalid document source type; expected TEXT")
	}

	text := strings.TrimSpace(input.RawText)
	if text == "" {
		return nil, ErrRawTextRequired
	}
	if len(text) > MaxCVTextLength {
		return nil, ErrTextTooLarge
	}

	doc := &model.CandidateDocument{
		CandidateID: candidateID,
		SourceType:  model.SourceTypeText,
		RawText:     text,
	}

	if err := s.candidateRepo.CreateDocument(ctx, doc); err != nil {
		return nil, err
	}

	return doc, nil
}

func (s *candidateService) UploadDocument(ctx context.Context, candidateID, originalFilename, mimeType string, size int64, reader io.Reader) (*model.CandidateDocument, error) {
	if _, err := s.candidateRepo.GetByID(ctx, candidateID, false); err != nil {
		return nil, err
	}

	if size > MaxUploadFileSize {
		return nil, ErrFileTooLarge
	}

	ext := strings.ToLower(filepath.Ext(originalFilename))
	if ext != ".pdf" && ext != ".docx" {
		return nil, errors.New("unsupported file extension, only .pdf and .docx are supported")
	}

	if s.storageService == nil {
		return nil, errors.New("storage service is not configured")
	}

	// Save stream safely to storage
	storagePath, err := s.storageService.Save(ctx, ext, reader)
	if err != nil {
		return nil, err
	}

	// Open reader to validate format and structural signature
	rAt, fileSize, closer, err := s.storageService.GetReaderAt(ctx, storagePath)
	if err != nil {
		_ = s.storageService.Delete(ctx, storagePath)
		return nil, err
	}

	detectedFormat, err := extractor.DetectAndValidateFormat(originalFilename, mimeType, rAt, fileSize)
	_ = closer.Close()
	if err != nil {
		_ = s.storageService.Delete(ctx, storagePath)
		return nil, err
	}

	sanitizedName := extractor.SanitizeFilename(originalFilename)
	actualSize := fileSize

	doc := &model.CandidateDocument{
		CandidateID:      candidateID,
		SourceType:       model.DocumentSourceType(detectedFormat),
		OriginalFilename: &sanitizedName,
		MimeType:         &mimeType,
		FileSize:         &actualSize,
		StoragePath:      &storagePath,
		RawText:          "",
	}

	if err := s.candidateRepo.CreateDocument(ctx, doc); err != nil {
		_ = s.storageService.Delete(ctx, storagePath)
		return nil, err
	}

	return doc, nil
}

func (s *candidateService) ProcessDocument(ctx context.Context, candidateID, docID string) (*ProcessDocumentResult, error) {
	candidate, err := s.candidateRepo.GetByID(ctx, candidateID, true)
	if err != nil {
		return nil, err
	}

	doc, err := s.candidateRepo.GetDocumentByID(ctx, docID)
	if err != nil {
		return nil, err
	}
	if doc.CandidateID != candidateID {
		return nil, ErrDocumentCandidateMismatch
	}

	var rawText string

	switch doc.SourceType {
	case model.SourceTypeText:
		rawText = doc.RawText
		if strings.TrimSpace(rawText) == "" {
			return nil, extractor.ErrExtractionFailed
		}
	case model.SourceTypePDF:
		if doc.StoragePath == nil || *doc.StoragePath == "" {
			return nil, ErrStoragePathMissing
		}
		if s.storageService == nil || s.pdfExtractor == nil {
			return nil, errors.New("PDF processing components are not configured")
		}
		rAt, size, closer, err := s.storageService.GetReaderAt(ctx, *doc.StoragePath)
		if err != nil {
			return nil, err
		}
		extracted, err := s.pdfExtractor.ExtractText(ctx, rAt, size)
		_ = closer.Close()
		if err != nil {
			return nil, err
		}
		rawText = extracted
	case model.SourceTypeDOCX:
		if doc.StoragePath == nil || *doc.StoragePath == "" {
			return nil, ErrStoragePathMissing
		}
		if s.storageService == nil || s.docxExtractor == nil {
			return nil, errors.New("DOCX processing components are not configured")
		}
		rAt, size, closer, err := s.storageService.GetReaderAt(ctx, *doc.StoragePath)
		if err != nil {
			return nil, err
		}
		extracted, err := s.docxExtractor.ExtractText(ctx, rAt, size)
		_ = closer.Close()
		if err != nil {
			return nil, err
		}
		rawText = extracted
	default:
		return nil, extractor.ErrUnsupportedFormat
	}

	// 1. Text Normalization
	normalizedText := normalizer.NormalizeCVText(rawText)
	if strings.TrimSpace(normalizedText) == "" {
		return nil, extractor.ErrExtractionFailed
	}
	if len(normalizedText) > extractor.MaxExtractedTextLength {
		return nil, extractor.ErrCVTextTooLarge
	}

	// Update document with normalized raw text
	doc.RawText = normalizedText
	if err := s.candidateRepo.UpdateDocument(ctx, doc); err != nil {
		return nil, err
	}

	// 2. Deterministic Parsing
	parsed := parser.Parse(normalizedText)

	// 3. Persist Candidate Profile Data (Manual Data Protection & Missing Field Fill)
	candidateUpdated := false
	if strings.TrimSpace(candidate.FullName) == "" && parsed.FullName != "" {
		candidate.FullName = parsed.FullName
		candidateUpdated = true
	}
	if strings.TrimSpace(candidate.Email) == "" && parsed.Email != "" {
		candidate.Email = model.NormalizeEmail(parsed.Email)
		candidateUpdated = true
	}
	if strings.TrimSpace(candidate.Phone) == "" && parsed.Phone != "" {
		candidate.Phone = parsed.Phone
		candidateUpdated = true
	}
	if strings.TrimSpace(candidate.Headline) == "" && parsed.Headline != "" {
		candidate.Headline = parsed.Headline
		candidateUpdated = true
	}
	if strings.TrimSpace(candidate.Summary) == "" && parsed.Summary != "" {
		candidate.Summary = parsed.Summary
		candidateUpdated = true
	}
	if candidateUpdated {
		_ = s.candidateRepo.Update(ctx, candidate)
	}

	// 4. Persist Skills (Idempotent: skip existing case-insensitive)
	for _, skillName := range parsed.Skills {
		cleanSkill := strings.TrimSpace(skillName)
		if cleanSkill == "" {
			continue
		}
		norm := strings.ToLower(cleanSkill)
		has, err := s.candidateRepo.HasSkill(ctx, candidateID, norm, "")
		if err == nil && !has {
			_ = s.candidateRepo.CreateSkill(ctx, &model.CandidateSkill{
				CandidateID:     candidateID,
				Skill:           cleanSkill,
				NormalizedSkill: norm,
			})
		}
	}

	// 5. Persist Educations (Idempotent: skip existing by institution)
	existingEdus, _ := s.candidateRepo.ListEducations(ctx, candidateID)
	for _, pe := range parsed.Educations {
		cleanInst := strings.TrimSpace(pe.Institution)
		if cleanInst == "" {
			continue
		}
		exists := false
		for _, ee := range existingEdus {
			if strings.EqualFold(strings.TrimSpace(ee.Institution), cleanInst) {
				exists = true
				break
			}
		}
		if !exists {
			sDate, _ := parseDate(pe.StartDate)
			eDate, _ := parseDate(pe.EndDate)
			_ = s.candidateRepo.CreateEducation(ctx, &model.CandidateEducation{
				CandidateID:  candidateID,
				Institution:  cleanInst,
				Degree:       strings.TrimSpace(pe.Degree),
				FieldOfStudy: strings.TrimSpace(pe.FieldOfStudy),
				StartDate:    sDate,
				EndDate:      eDate,
				Description:  strings.TrimSpace(pe.Description),
			})
		}
	}

	// 6. Persist Experiences (Idempotent: skip existing by company and position)
	existingExps, _ := s.candidateRepo.ListExperiences(ctx, candidateID)
	for _, px := range parsed.Experiences {
		cleanComp := strings.TrimSpace(px.Company)
		cleanPos := strings.TrimSpace(px.Position)
		if cleanComp == "" && cleanPos == "" {
			continue
		}
		exists := false
		for _, ex := range existingExps {
			if strings.EqualFold(strings.TrimSpace(ex.Company), cleanComp) &&
				strings.EqualFold(strings.TrimSpace(ex.Position), cleanPos) {
				exists = true
				break
			}
		}
		if !exists {
			sDate, _ := parseDate(px.StartDate)
			eDate, _ := parseDate(px.EndDate)
			_ = s.candidateRepo.CreateExperience(ctx, &model.CandidateExperience{
				CandidateID:    candidateID,
				Company:        cleanComp,
				Position:       cleanPos,
				Location:       strings.TrimSpace(px.Location),
				EmploymentType: px.EmploymentType,
				StartDate:      sDate,
				EndDate:        eDate,
				IsCurrent:      px.IsCurrent,
				Description:    strings.TrimSpace(px.Description),
			})
		}
	}

	// 7. Reload Candidate with fresh relations
	finalCandidate, err := s.candidateRepo.GetByID(ctx, candidateID, true)
	if err != nil {
		finalCandidate = candidate
	}

	return &ProcessDocumentResult{
		DocumentID: doc.ID,
		SourceType: string(doc.SourceType),
		Extraction: ExtractionMeta{
			Status:     "SUCCESS",
			Characters: len(normalizedText),
		},
		Parsing: ParsingMeta{
			Status:         "COMPLETED",
			FieldsDetected: parsed.FieldsDetected,
		},
		Candidate: finalCandidate,
	}, nil
}

func (s *candidateService) GetDocumentByID(ctx context.Context, candidateID, docID string) (*model.CandidateDocument, error) {
	if _, err := s.candidateRepo.GetByID(ctx, candidateID, false); err != nil {
		return nil, err
	}

	doc, err := s.candidateRepo.GetDocumentByID(ctx, docID)
	if err != nil {
		return nil, err
	}
	if doc.CandidateID != candidateID {
		return nil, repository.ErrDocumentNotFound
	}
	return doc, nil
}

func (s *candidateService) ListDocuments(ctx context.Context, candidateID string) ([]model.CandidateDocument, error) {
	if _, err := s.candidateRepo.GetByID(ctx, candidateID, false); err != nil {
		return nil, err
	}
	return s.candidateRepo.ListDocuments(ctx, candidateID)
}

func (s *candidateService) DeleteDocument(ctx context.Context, candidateID, docID string) error {
	if _, err := s.candidateRepo.GetByID(ctx, candidateID, false); err != nil {
		return err
	}

	doc, err := s.candidateRepo.GetDocumentByID(ctx, docID)
	if err != nil {
		return err
	}
	if doc.CandidateID != candidateID {
		return repository.ErrDocumentNotFound
	}

	// Safely clean up physical storage file if present
	if s.storageService != nil && doc.StoragePath != nil && *doc.StoragePath != "" {
		_ = s.storageService.Delete(ctx, *doc.StoragePath)
	}

	return s.candidateRepo.DeleteDocument(ctx, candidateID, docID)
}

func parseDate(val *string) (*time.Time, error) {
	if val == nil {
		return nil, nil
	}
	raw := strings.TrimSpace(*val)
	if raw == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return nil, fmt.Errorf("invalid date format, expected YYYY-MM-DD: %w", err)
	}
	return &t, nil
}
