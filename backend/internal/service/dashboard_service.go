package service

import (
	"context"
	"time"

	"hirescope/backend/internal/model"
	"hirescope/backend/internal/repository"
)

// DashboardSummaryResponse presents aggregated high-level KPIs and pipeline distribution.
type DashboardSummaryResponse struct {
	ActiveJobs            int64                 `json:"active_jobs"`
	TotalCandidates       int64                 `json:"total_candidates"`
	ReviewCandidates      int64                 `json:"review_candidates"`
	ShortlistedCandidates int64                 `json:"shortlisted_candidates"`
	RejectedCandidates    int64                 `json:"rejected_candidates"`
	JobStatus             DashboardJobStatusDTO `json:"job_status"`
	Pipeline              DashboardPipelineDTO  `json:"pipeline"`
	NeedsAttention        DashboardAttentionDTO `json:"needs_attention"`
}

type DashboardJobStatusDTO struct {
	Draft    int64 `json:"draft"`
	Open     int64 `json:"open"`
	Closed   int64 `json:"closed"`
	Archived int64 `json:"archived"`
}

type DashboardPipelineDTO struct {
	Review      int64 `json:"review"`
	Shortlisted int64 `json:"shortlisted"`
	Rejected    int64 `json:"rejected"`
}

type DashboardAttentionDTO struct {
	AwaitingReviewCount  int64 `json:"awaiting_review_count"`
	OpenJobsNoCandidates int64 `json:"open_jobs_no_candidates"`
}

type DashboardRecentCandidateDTO struct {
	ID          string    `json:"id"`
	CandidateID string    `json:"candidate_id"`
	FullName    string    `json:"full_name"`
	Email       string    `json:"email,omitempty"`
	JobID       string    `json:"job_id,omitempty"`
	JobTitle    string    `json:"job_title"`
	Status      string    `json:"status"`
	AssignedAt  time.Time `json:"assigned_at"`
}

type DashboardOpenJobDTO struct {
	ID              string               `json:"id"`
	Code            string               `json:"code"`
	Title           string               `json:"title"`
	Department      string               `json:"department"`
	Location        string               `json:"location"`
	EmploymentType  model.EmploymentType `json:"employment_type"`
	Status          model.JobStatus      `json:"status"`
	CandidatesCount int64                `json:"candidates_count"`
	UpdatedAt       time.Time            `json:"updated_at"`
}

type DashboardActivityDTO struct {
	ID            string    `json:"id"`
	Action        string    `json:"action"`
	ActorName     string    `json:"actor_name"`
	JobID         *string   `json:"job_id,omitempty"`
	JobTitle      *string   `json:"job_title,omitempty"`
	CandidateID   *string   `json:"candidate_id,omitempty"`
	CandidateName *string   `json:"candidate_name,omitempty"`
	Metadata      string    `json:"metadata,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// DashboardService provides read-only business operations for recruiter dashboard intelligence.
type DashboardService interface {
	GetSummary(ctx context.Context) (*DashboardSummaryResponse, error)
	GetRecentCandidates(ctx context.Context, limit int) ([]DashboardRecentCandidateDTO, error)
	GetOpenJobs(ctx context.Context, limit int) ([]DashboardOpenJobDTO, error)
	GetRecentActivity(ctx context.Context, limit int) ([]DashboardActivityDTO, error)
}

type dashboardService struct {
	dashboardRepo repository.DashboardRepository
}

// NewDashboardService creates a new instance of DashboardService.
func NewDashboardService(dashboardRepo repository.DashboardRepository) DashboardService {
	return &dashboardService{dashboardRepo: dashboardRepo}
}

func (s *dashboardService) GetSummary(ctx context.Context) (*DashboardSummaryResponse, error) {
	counts, err := s.dashboardRepo.GetSummaryCounts(ctx)
	if err != nil {
		return nil, err
	}

	return &DashboardSummaryResponse{
		ActiveJobs:            counts.ActiveJobs,
		TotalCandidates:       counts.TotalCandidates,
		ReviewCandidates:      counts.ReviewCandidates,
		ShortlistedCandidates: counts.ShortlistedCandidates,
		RejectedCandidates:    counts.RejectedCandidates,
		JobStatus: DashboardJobStatusDTO{
			Draft:    counts.DraftJobs,
			Open:     counts.OpenJobs,
			Closed:   counts.ClosedJobs,
			Archived: counts.ArchivedJobs,
		},
		Pipeline: DashboardPipelineDTO{
			Review:      counts.ReviewCandidates,
			Shortlisted: counts.ShortlistedCandidates,
			Rejected:    counts.RejectedCandidates,
		},
		NeedsAttention: DashboardAttentionDTO{
			AwaitingReviewCount:  counts.ReviewCandidates,
			OpenJobsNoCandidates: counts.OpenJobsNoCandidates,
		},
	}, nil
}

func (s *dashboardService) GetRecentCandidates(ctx context.Context, limit int) ([]DashboardRecentCandidateDTO, error) {
	rows, err := s.dashboardRepo.GetRecentCandidates(ctx, limit)
	if err != nil {
		return nil, err
	}

	dtos := make([]DashboardRecentCandidateDTO, len(rows))
	for i, r := range rows {
		dtos[i] = DashboardRecentCandidateDTO{
			ID:          r.ID,
			CandidateID: r.CandidateID,
			FullName:    r.FullName,
			Email:       r.Email,
			JobID:       r.JobID,
			JobTitle:    r.JobTitle,
			Status:      r.Status,
			AssignedAt:  r.AssignedAt,
		}
	}
	return dtos, nil
}

func (s *dashboardService) GetOpenJobs(ctx context.Context, limit int) ([]DashboardOpenJobDTO, error) {
	rows, err := s.dashboardRepo.GetOpenJobsWithCounts(ctx, limit)
	if err != nil {
		return nil, err
	}

	dtos := make([]DashboardOpenJobDTO, len(rows))
	for i, r := range rows {
		dtos[i] = DashboardOpenJobDTO{
			ID:              r.ID,
			Code:            r.Code,
			Title:           r.Title,
			Department:      r.Department,
			Location:        r.Location,
			EmploymentType:  r.EmploymentType,
			Status:          r.Status,
			CandidatesCount: r.CandidatesCount,
			UpdatedAt:       r.UpdatedAt,
		}
	}
	return dtos, nil
}

func (s *dashboardService) GetRecentActivity(ctx context.Context, limit int) ([]DashboardActivityDTO, error) {
	rows, err := s.dashboardRepo.GetRecentActivity(ctx, limit)
	if err != nil {
		return nil, err
	}

	dtos := make([]DashboardActivityDTO, len(rows))
	for i, r := range rows {
		dtos[i] = DashboardActivityDTO{
			ID:            r.ID,
			Action:        r.Action,
			ActorName:     r.ActorName,
			JobID:         r.JobID,
			JobTitle:      r.JobTitle,
			CandidateID:   r.CandidateID,
			CandidateName: r.CandidateName,
			Metadata:      r.Metadata,
			CreatedAt:     r.CreatedAt,
		}
	}
	return dtos, nil
}
