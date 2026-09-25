package service

import (
	"context"
	"errors"
	"time"

	"hirescope/backend/internal/model"
	"hirescope/backend/internal/repository"

	"github.com/google/uuid"
)

var (
	ErrInvalidUUID = errors.New("invalid UUID format")
)

// ScreeningResultDTO presents the high-level screening summary for API consumers.
type ScreeningResultDTO struct {
	ID                     string                      `json:"id"`
	JobID                  string                      `json:"job_id"`
	CandidateID            string                      `json:"candidate_id"`
	Status                 model.ScreeningResultStatus `json:"status"`
	RequiredMatchCount     int                         `json:"required_match_count"`
	RequiredPartialCount   int                         `json:"required_partial_count"`
	RequiredMismatchCount  int                         `json:"required_mismatch_count"`
	RequiredUnknownCount   int                         `json:"required_unknown_count"`
	PreferredMatchCount    int                         `json:"preferred_match_count"`
	PreferredPartialCount  int                         `json:"preferred_partial_count"`
	PreferredMismatchCount int                         `json:"preferred_mismatch_count"`
	PreferredUnknownCount  int                         `json:"preferred_unknown_count"`
	EvaluatedAt            time.Time                   `json:"evaluated_at"`
}

// ScreeningMatchDTO presents requirement-by-requirement explainable match data.
type ScreeningMatchDTO struct {
	RequirementID string                      `json:"requirement_id"`
	Requirement   string                      `json:"requirement"`
	Category      model.RequirementCategory   `json:"category"`
	Importance    model.RequirementImportance `json:"importance"`
	Status        model.MatchStatus           `json:"status"`
	Evidence      string                      `json:"evidence"`
	Reason        string                      `json:"reason"`
	Confidence    model.MatchConfidence       `json:"confidence"`
	Source        model.MatchSource           `json:"source"`
	MatchedValue  string                      `json:"matched_value,omitempty"`
}

// ScreeningResponse encapsulates both the overall summary and individual requirement matches.
type ScreeningResponse struct {
	ScreeningResult ScreeningResultDTO  `json:"screening_result"`
	Matches         []ScreeningMatchDTO `json:"matches"`
}

// ScreeningService defines business operations for candidate evaluation and screening retrieval.
type ScreeningService interface {
	ScreenCandidate(ctx context.Context, jobID, candidateID string) (*ScreeningResponse, error)
	GetLatestScreeningResult(ctx context.Context, jobID, candidateID string) (*ScreeningResponse, error)
}

type screeningService struct {
	screeningRepo   repository.ScreeningRepository
	jobRepo         repository.JobRepository
	candidateRepo   repository.CandidateRepository
	screeningEngine ScreeningEngine
}

// NewScreeningService creates an instance of ScreeningService.
func NewScreeningService(
	screeningRepo repository.ScreeningRepository,
	jobRepo repository.JobRepository,
	candidateRepo repository.CandidateRepository,
	screeningEngine ScreeningEngine,
) ScreeningService {
	return &screeningService{
		screeningRepo:   screeningRepo,
		jobRepo:         jobRepo,
		candidateRepo:   candidateRepo,
		screeningEngine: screeningEngine,
	}
}

func (s *screeningService) ScreenCandidate(ctx context.Context, jobID, candidateID string) (*ScreeningResponse, error) {
	// 1. Validate UUID formats
	if _, err := uuid.Parse(jobID); err != nil {
		return nil, ErrInvalidUUID
	}
	if _, err := uuid.Parse(candidateID); err != nil {
		return nil, ErrInvalidUUID
	}

	// 2. Verify Job exists and load requirements
	job, err := s.jobRepo.GetByID(ctx, jobID, true)
	if err != nil {
		return nil, err
	}

	// 3. Verify Candidate exists and load background details
	cand, err := s.candidateRepo.GetByID(ctx, candidateID, true)
	if err != nil {
		return nil, err
	}

	// 4. Verify Candidate is linked to Job
	associated, err := s.candidateRepo.IsJobCandidateAssociated(ctx, jobID, candidateID)
	if err != nil {
		return nil, err
	}
	if !associated {
		return nil, repository.ErrJobCandidateNotFound
	}

	// 5. Execute deterministic evaluation across all requirements
	engineResult, err := s.screeningEngine.ScreenCandidate(ctx, job, cand)
	if err != nil {
		return nil, err
	}

	// 6. Build model.ScreeningResult
	screeningResult := &model.ScreeningResult{
		CandidateID:            candidateID,
		JobID:                  jobID,
		Status:                 engineResult.Status,
		RequiredMatchCount:     engineResult.RequiredMatchCount,
		RequiredPartialCount:   engineResult.RequiredPartialCount,
		RequiredMismatchCount:  engineResult.RequiredMismatchCount,
		RequiredUnknownCount:   engineResult.RequiredUnknownCount,
		PreferredMatchCount:    engineResult.PreferredMatchCount,
		PreferredPartialCount:  engineResult.PreferredPartialCount,
		PreferredMismatchCount: engineResult.PreferredMismatchCount,
		PreferredUnknownCount:  engineResult.PreferredUnknownCount,
		EvaluatedAt:            time.Now().UTC(),
	}

	// 7. Persist atomically in repository
	savedResult, err := s.screeningRepo.SaveScreeningResult(ctx, screeningResult, engineResult.Matches)
	if err != nil {
		return nil, err
	}

	return toScreeningResponse(savedResult), nil
}

func (s *screeningService) GetLatestScreeningResult(ctx context.Context, jobID, candidateID string) (*ScreeningResponse, error) {
	// 1. Validate UUID formats
	if _, err := uuid.Parse(jobID); err != nil {
		return nil, ErrInvalidUUID
	}
	if _, err := uuid.Parse(candidateID); err != nil {
		return nil, ErrInvalidUUID
	}

	// 2. Verify Job exists
	if _, err := s.jobRepo.GetByID(ctx, jobID, false); err != nil {
		return nil, err
	}

	// 3. Verify Candidate exists
	if _, err := s.candidateRepo.GetByID(ctx, candidateID, false); err != nil {
		return nil, err
	}

	// 4. Verify Candidate is associated with Job
	associated, err := s.candidateRepo.IsJobCandidateAssociated(ctx, jobID, candidateID)
	if err != nil {
		return nil, err
	}
	if !associated {
		return nil, repository.ErrJobCandidateNotFound
	}

	// 5. Fetch from repository
	result, err := s.screeningRepo.GetByCandidateAndJob(ctx, candidateID, jobID)
	if err != nil {
		return nil, err
	}

	return toScreeningResponse(result), nil
}

func toScreeningResponse(r *model.ScreeningResult) *ScreeningResponse {
	resp := &ScreeningResponse{
		ScreeningResult: ScreeningResultDTO{
			ID:                     r.ID,
			JobID:                  r.JobID,
			CandidateID:            r.CandidateID,
			Status:                 r.Status,
			RequiredMatchCount:     r.RequiredMatchCount,
			RequiredPartialCount:   r.RequiredPartialCount,
			RequiredMismatchCount:  r.RequiredMismatchCount,
			RequiredUnknownCount:   r.RequiredUnknownCount,
			PreferredMatchCount:    r.PreferredMatchCount,
			PreferredPartialCount:  r.PreferredPartialCount,
			PreferredMismatchCount: r.PreferredMismatchCount,
			PreferredUnknownCount:  r.PreferredUnknownCount,
			EvaluatedAt:            r.EvaluatedAt,
		},
		Matches: make([]ScreeningMatchDTO, 0, len(r.Matches)),
	}

	for _, m := range r.Matches {
		dto := ScreeningMatchDTO{
			RequirementID: m.RequirementID,
			Status:        m.Status,
			Evidence:      m.Evidence,
			Reason:        m.Reason,
			Confidence:    m.Confidence,
			Source:        m.Source,
			MatchedValue:  m.MatchedValue,
		}
		if m.Requirement != nil {
			dto.Requirement = m.Requirement.Requirement
			dto.Category = m.Requirement.Category
			dto.Importance = m.Requirement.Importance
		}
		resp.Matches = append(resp.Matches, dto)
	}

	return resp
}
