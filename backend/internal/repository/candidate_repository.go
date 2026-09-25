package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"hirescope/backend/internal/model"

	"gorm.io/gorm"
)

var (
	ErrCandidateNotFound         = errors.New("candidate not found")
	ErrJobCandidateAlreadyExists = errors.New("candidate is already associated with this job")
	ErrJobCandidateNotFound      = errors.New("candidate is not associated with this job")
	ErrEducationNotFound         = errors.New("education record not found")
	ErrExperienceNotFound        = errors.New("experience record not found")
	ErrSkillNotFound             = errors.New("skill record not found")
	ErrDuplicateSkill            = errors.New("skill already exists for this candidate")
	ErrDocumentNotFound          = errors.New("candidate document not found")
)

// CandidateFilter encapsulates query parameters for searching candidates.
type CandidateFilter struct {
	Search   string
	Location string
	Page     int
	Limit    int
	Sort     string
	Order    string
}

// CandidateListResult contains paginated candidate items and pagination metadata.
type CandidateListResult struct {
	Items      []model.Candidate `json:"items"`
	Total      int64             `json:"total"`
	Page       int               `json:"page"`
	Limit      int               `json:"limit"`
	TotalPages int               `json:"total_pages"`
}

// CandidateRepository defines data access methods for Candidates and their child entities.
type CandidateRepository interface {
	Create(ctx context.Context, candidate *model.Candidate) error
	GetByID(ctx context.Context, id string, includeRelations bool) (*model.Candidate, error)
	List(ctx context.Context, filter CandidateFilter) (*CandidateListResult, error)
	Update(ctx context.Context, candidate *model.Candidate) error

	// Job-Candidate association
	AddJobCandidate(ctx context.Context, jobID, candidateID string) error
	RemoveJobCandidate(ctx context.Context, jobID, candidateID string) error
	IsJobCandidateAssociated(ctx context.Context, jobID, candidateID string) (bool, error)
	ListCandidatesByJob(ctx context.Context, jobID string, page, limit int) (*CandidateListResult, error)
	ListJobsByCandidate(ctx context.Context, candidateID string) ([]model.Job, error)

	// Educations
	CreateEducation(ctx context.Context, edu *model.CandidateEducation) error
	GetEducationByID(ctx context.Context, id string) (*model.CandidateEducation, error)
	ListEducations(ctx context.Context, candidateID string) ([]model.CandidateEducation, error)
	UpdateEducation(ctx context.Context, edu *model.CandidateEducation) error
	DeleteEducation(ctx context.Context, candidateID, eduID string) error

	// Experiences
	CreateExperience(ctx context.Context, exp *model.CandidateExperience) error
	GetExperienceByID(ctx context.Context, id string) (*model.CandidateExperience, error)
	ListExperiences(ctx context.Context, candidateID string) ([]model.CandidateExperience, error)
	UpdateExperience(ctx context.Context, exp *model.CandidateExperience) error
	DeleteExperience(ctx context.Context, candidateID, expID string) error

	// Skills
	CreateSkill(ctx context.Context, skill *model.CandidateSkill) error
	GetSkillByID(ctx context.Context, id string) (*model.CandidateSkill, error)
	ListSkills(ctx context.Context, candidateID string) ([]model.CandidateSkill, error)
	UpdateSkill(ctx context.Context, skill *model.CandidateSkill) error
	DeleteSkill(ctx context.Context, candidateID, skillID string) error
	HasSkill(ctx context.Context, candidateID, normalizedSkill string, excludeSkillID string) (bool, error)

	// Documents
	CreateDocument(ctx context.Context, doc *model.CandidateDocument) error
	GetDocumentByID(ctx context.Context, id string) (*model.CandidateDocument, error)
	ListDocuments(ctx context.Context, candidateID string) ([]model.CandidateDocument, error)
	UpdateDocument(ctx context.Context, doc *model.CandidateDocument) error
	DeleteDocument(ctx context.Context, candidateID, docID string) error
}

type gormCandidateRepository struct {
	db *gorm.DB
}

// NewCandidateRepository creates a new GORM-backed CandidateRepository.
func NewCandidateRepository(db *gorm.DB) CandidateRepository {
	return &gormCandidateRepository{db: db}
}

func (r *gormCandidateRepository) Create(ctx context.Context, candidate *model.Candidate) error {
	return r.db.WithContext(ctx).Create(candidate).Error
}

func (r *gormCandidateRepository) GetByID(ctx context.Context, id string, includeRelations bool) (*model.Candidate, error) {
	var candidate model.Candidate
	query := r.db.WithContext(ctx).Where("id = ?", id)
	if includeRelations {
		query = query.
			Preload("Creator").
			Preload("Educations").
			Preload("Experiences").
			Preload("Skills").
			Preload("Documents")
	}

	err := query.First(&candidate).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCandidateNotFound
		}
		return nil, err
	}
	return &candidate, nil
}

func (r *gormCandidateRepository) List(ctx context.Context, filter CandidateFilter) (*CandidateListResult, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 10
	} else if filter.Limit > 100 {
		filter.Limit = 100
	}

	query := r.db.WithContext(ctx).Model(&model.Candidate{})

	// Search filter
	if strings.TrimSpace(filter.Search) != "" {
		s := "%" + strings.ToLower(strings.TrimSpace(filter.Search)) + "%"
		query = query.Where("LOWER(full_name) LIKE ? OR LOWER(email) LIKE ? OR LOWER(headline) LIKE ?", s, s, s)
	}

	// Location filter
	if strings.TrimSpace(filter.Location) != "" {
		query = query.Where("LOWER(location) = ?", strings.ToLower(strings.TrimSpace(filter.Location)))
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Sorting
	allowedSorts := map[string]string{
		"created_at": "created_at",
		"full_name":  "full_name",
		"email":      "email",
		"location":   "location",
	}
	sortField, ok := allowedSorts[strings.ToLower(filter.Sort)]
	if !ok {
		sortField = "created_at"
	}

	orderDir := "DESC"
	if strings.EqualFold(filter.Order, "asc") {
		orderDir = "ASC"
	}
	orderClause := fmt.Sprintf("%s %s", sortField, orderDir)

	offset := (filter.Page - 1) * filter.Limit
	var items []model.Candidate
	err := query.Order(orderClause).
		Offset(offset).
		Limit(filter.Limit).
		Preload("Skills").
		Find(&items).Error
	if err != nil {
		return nil, err
	}

	totalPages := int((total + int64(filter.Limit) - 1) / int64(filter.Limit))
	if totalPages == 0 {
		totalPages = 1
	}

	return &CandidateListResult{
		Items:      items,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

func (r *gormCandidateRepository) Update(ctx context.Context, candidate *model.Candidate) error {
	return r.db.WithContext(ctx).Save(candidate).Error
}

func (r *gormCandidateRepository) AddJobCandidate(ctx context.Context, jobID, candidateID string) error {
	assoc := &model.JobCandidate{
		JobID:       jobID,
		CandidateID: candidateID,
	}
	err := r.db.WithContext(ctx).Create(assoc).Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(err.Error(), "idx_job_candidate") {
			return ErrJobCandidateAlreadyExists
		}
		return err
	}
	return nil
}

func (r *gormCandidateRepository) RemoveJobCandidate(ctx context.Context, jobID, candidateID string) error {
	res := r.db.WithContext(ctx).
		Where("job_id = ? AND candidate_id = ?", jobID, candidateID).
		Delete(&model.JobCandidate{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrJobCandidateNotFound
	}
	return nil
}

func (r *gormCandidateRepository) IsJobCandidateAssociated(ctx context.Context, jobID, candidateID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.JobCandidate{}).
		Where("job_id = ? AND candidate_id = ?", jobID, candidateID).
		Count(&count).Error
	return count > 0, err
}

func (r *gormCandidateRepository) ListCandidatesByJob(ctx context.Context, jobID string, page, limit int) (*CandidateListResult, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	} else if limit > 100 {
		limit = 100
	}

	var total int64
	countQuery := r.db.WithContext(ctx).Model(&model.JobCandidate{}).Where("job_id = ?", jobID)
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, err
	}

	offset := (page - 1) * limit
	var associations []model.JobCandidate
	err := r.db.WithContext(ctx).
		Where("job_id = ?", jobID).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Preload("Candidate").
		Preload("Candidate.Skills").
		Find(&associations).Error
	if err != nil {
		return nil, err
	}

	var candidates []model.Candidate
	for _, assoc := range associations {
		if assoc.Candidate != nil {
			candidates = append(candidates, *assoc.Candidate)
		}
	}

	totalPages := int((total + int64(limit) - 1) / int64(limit))
	if totalPages == 0 {
		totalPages = 1
	}

	return &CandidateListResult{
		Items:      candidates,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

func (r *gormCandidateRepository) ListJobsByCandidate(ctx context.Context, candidateID string) ([]model.Job, error) {
	var associations []model.JobCandidate
	err := r.db.WithContext(ctx).
		Where("candidate_id = ?", candidateID).
		Order("created_at DESC").
		Preload("Job").
		Find(&associations).Error
	if err != nil {
		return nil, err
	}

	var jobs []model.Job
	for _, assoc := range associations {
		if assoc.Job != nil {
			jobs = append(jobs, *assoc.Job)
		}
	}
	return jobs, nil
}

func (r *gormCandidateRepository) CreateEducation(ctx context.Context, edu *model.CandidateEducation) error {
	return r.db.WithContext(ctx).Create(edu).Error
}

func (r *gormCandidateRepository) GetEducationByID(ctx context.Context, id string) (*model.CandidateEducation, error) {
	var edu model.CandidateEducation
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&edu).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrEducationNotFound
		}
		return nil, err
	}
	return &edu, nil
}

func (r *gormCandidateRepository) ListEducations(ctx context.Context, candidateID string) ([]model.CandidateEducation, error) {
	var list []model.CandidateEducation
	err := r.db.WithContext(ctx).
		Where("candidate_id = ?", candidateID).
		Order("start_date DESC").
		Find(&list).Error
	return list, err
}

func (r *gormCandidateRepository) UpdateEducation(ctx context.Context, edu *model.CandidateEducation) error {
	return r.db.WithContext(ctx).Save(edu).Error
}

func (r *gormCandidateRepository) DeleteEducation(ctx context.Context, candidateID, eduID string) error {
	res := r.db.WithContext(ctx).
		Where("id = ? AND candidate_id = ?", eduID, candidateID).
		Delete(&model.CandidateEducation{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrEducationNotFound
	}
	return nil
}

func (r *gormCandidateRepository) CreateExperience(ctx context.Context, exp *model.CandidateExperience) error {
	return r.db.WithContext(ctx).Create(exp).Error
}

func (r *gormCandidateRepository) GetExperienceByID(ctx context.Context, id string) (*model.CandidateExperience, error) {
	var exp model.CandidateExperience
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&exp).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrExperienceNotFound
		}
		return nil, err
	}
	return &exp, nil
}

func (r *gormCandidateRepository) ListExperiences(ctx context.Context, candidateID string) ([]model.CandidateExperience, error) {
	var list []model.CandidateExperience
	err := r.db.WithContext(ctx).
		Where("candidate_id = ?", candidateID).
		Order("start_date DESC").
		Find(&list).Error
	return list, err
}

func (r *gormCandidateRepository) UpdateExperience(ctx context.Context, exp *model.CandidateExperience) error {
	return r.db.WithContext(ctx).Save(exp).Error
}

func (r *gormCandidateRepository) DeleteExperience(ctx context.Context, candidateID, expID string) error {
	res := r.db.WithContext(ctx).
		Where("id = ? AND candidate_id = ?", expID, candidateID).
		Delete(&model.CandidateExperience{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrExperienceNotFound
	}
	return nil
}

func (r *gormCandidateRepository) CreateSkill(ctx context.Context, skill *model.CandidateSkill) error {
	err := r.db.WithContext(ctx).Create(skill).Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(err.Error(), "idx_cand_skill") {
			return ErrDuplicateSkill
		}
		return err
	}
	return nil
}

func (r *gormCandidateRepository) GetSkillByID(ctx context.Context, id string) (*model.CandidateSkill, error) {
	var skill model.CandidateSkill
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&skill).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSkillNotFound
		}
		return nil, err
	}
	return &skill, nil
}

func (r *gormCandidateRepository) ListSkills(ctx context.Context, candidateID string) ([]model.CandidateSkill, error) {
	var list []model.CandidateSkill
	err := r.db.WithContext(ctx).
		Where("candidate_id = ?", candidateID).
		Order("skill ASC").
		Find(&list).Error
	return list, err
}

func (r *gormCandidateRepository) UpdateSkill(ctx context.Context, skill *model.CandidateSkill) error {
	err := r.db.WithContext(ctx).Save(skill).Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(err.Error(), "idx_cand_skill") {
			return ErrDuplicateSkill
		}
		return err
	}
	return nil
}

func (r *gormCandidateRepository) DeleteSkill(ctx context.Context, candidateID, skillID string) error {
	res := r.db.WithContext(ctx).
		Where("id = ? AND candidate_id = ?", skillID, candidateID).
		Delete(&model.CandidateSkill{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrSkillNotFound
	}
	return nil
}

func (r *gormCandidateRepository) HasSkill(ctx context.Context, candidateID, normalizedSkill string, excludeSkillID string) (bool, error) {
	query := r.db.WithContext(ctx).Model(&model.CandidateSkill{}).
		Where("candidate_id = ? AND normalized_skill = ?", candidateID, normalizedSkill)
	if excludeSkillID != "" {
		query = query.Where("id != ?", excludeSkillID)
	}
	var count int64
	err := query.Count(&count).Error
	return count > 0, err
}

func (r *gormCandidateRepository) CreateDocument(ctx context.Context, doc *model.CandidateDocument) error {
	return r.db.WithContext(ctx).Create(doc).Error
}

func (r *gormCandidateRepository) GetDocumentByID(ctx context.Context, id string) (*model.CandidateDocument, error) {
	var doc model.CandidateDocument
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&doc).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDocumentNotFound
		}
		return nil, err
	}
	return &doc, nil
}

func (r *gormCandidateRepository) ListDocuments(ctx context.Context, candidateID string) ([]model.CandidateDocument, error) {
	var list []model.CandidateDocument
	err := r.db.WithContext(ctx).
		Where("candidate_id = ?", candidateID).
		Order("created_at DESC").
		Find(&list).Error
	return list, err
}

func (r *gormCandidateRepository) UpdateDocument(ctx context.Context, doc *model.CandidateDocument) error {
	return r.db.WithContext(ctx).Save(doc).Error
}

func (r *gormCandidateRepository) DeleteDocument(ctx context.Context, candidateID, docID string) error {
	res := r.db.WithContext(ctx).
		Where("id = ? AND candidate_id = ?", docID, candidateID).
		Delete(&model.CandidateDocument{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrDocumentNotFound
	}
	return nil
}
