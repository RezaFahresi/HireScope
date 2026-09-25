package repository

import (
	"context"
	"time"

	"hirescope/backend/internal/model"

	"gorm.io/gorm"
)

// DashboardSummaryCounts encapsulates raw counts aggregated across recruitment entities.
type DashboardSummaryCounts struct {
	ActiveJobs            int64
	TotalCandidates       int64
	ReviewCandidates      int64
	ShortlistedCandidates int64
	RejectedCandidates    int64
	DraftJobs             int64
	OpenJobs              int64
	ClosedJobs            int64
	ArchivedJobs          int64
	OpenJobsNoCandidates  int64
}

// RecentCandidateRow represents an assigned or registered candidate row for the dashboard.
type RecentCandidateRow struct {
	ID          string    `json:"id"`
	CandidateID string    `json:"candidate_id"`
	FullName    string    `json:"full_name"`
	Email       string    `json:"email"`
	JobID       string    `json:"job_id"`
	JobTitle    string    `json:"job_title"`
	Status      string    `json:"status"`
	AssignedAt  time.Time `json:"assigned_at"`
}

// OpenJobRow encapsulates an open vacancy along with its total candidate applicant count.
type OpenJobRow struct {
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

// DashboardActivityRow encapsulates a recruiter audit trail log item for the dashboard.
type DashboardActivityRow struct {
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

// DashboardRepository defines read-only database aggregation queries for recruiter operations.
type DashboardRepository interface {
	GetSummaryCounts(ctx context.Context) (*DashboardSummaryCounts, error)
	GetRecentCandidates(ctx context.Context, limit int) ([]RecentCandidateRow, error)
	GetOpenJobsWithCounts(ctx context.Context, limit int) ([]OpenJobRow, error)
	GetRecentActivity(ctx context.Context, limit int) ([]DashboardActivityRow, error)
}

type gormDashboardRepository struct {
	db *gorm.DB
}

// NewDashboardRepository creates a new GORM-backed DashboardRepository.
func NewDashboardRepository(db *gorm.DB) DashboardRepository {
	return &gormDashboardRepository{db: db}
}

func (r *gormDashboardRepository) GetSummaryCounts(ctx context.Context) (*DashboardSummaryCounts, error) {
	counts := &DashboardSummaryCounts{}

	// 1. Job counts grouped by status
	type statusCount struct {
		Status string
		Count  int64
	}
	var jobCounts []statusCount
	if err := r.db.WithContext(ctx).Model(&model.Job{}).Select("status, count(*) as count").Group("status").Scan(&jobCounts).Error; err != nil {
		return nil, err
	}
	for _, sc := range jobCounts {
		switch model.JobStatus(sc.Status) {
		case model.JobStatusDraft:
			counts.DraftJobs = sc.Count
		case model.JobStatusOpen:
			counts.OpenJobs = sc.Count
			counts.ActiveJobs = sc.Count
		case model.JobStatusClosed:
			counts.ClosedJobs = sc.Count
		case model.JobStatusArchived:
			counts.ArchivedJobs = sc.Count
		}
	}

	// 2. Total candidate records
	if err := r.db.WithContext(ctx).Model(&model.Candidate{}).Count(&counts.TotalCandidates).Error; err != nil {
		return nil, err
	}

	// 3. Pipeline workflow counts grouped by status
	var pipelineCounts []statusCount
	if err := r.db.WithContext(ctx).Model(&model.JobCandidate{}).Select("status, count(*) as count").Group("status").Scan(&pipelineCounts).Error; err != nil {
		return nil, err
	}
	for _, sc := range pipelineCounts {
		switch model.JobCandidateStatus(sc.Status) {
		case model.JobCandidateStatusReview:
			counts.ReviewCandidates = sc.Count
		case model.JobCandidateStatusShortlisted:
			counts.ShortlistedCandidates = sc.Count
		case model.JobCandidateStatusRejected:
			counts.RejectedCandidates = sc.Count
		}
	}

	// 4. Open jobs with 0 candidates
	if err := r.db.WithContext(ctx).Model(&model.Job{}).
		Where("status = ? AND id NOT IN (SELECT DISTINCT job_id FROM job_candidates)", model.JobStatusOpen).
		Count(&counts.OpenJobsNoCandidates).Error; err != nil {
		return nil, err
	}

	return counts, nil
}

func (r *gormDashboardRepository) GetRecentCandidates(ctx context.Context, limit int) ([]RecentCandidateRow, error) {
	if limit <= 0 || limit > 50 {
		limit = 5
	}
	var rows []RecentCandidateRow
	err := r.db.WithContext(ctx).Table("job_candidates jc").
		Select("jc.id, jc.candidate_id, c.full_name, c.email, jc.job_id, j.title as job_title, jc.status, jc.created_at as assigned_at").
		Joins("JOIN candidates c ON c.id = jc.candidate_id").
		Joins("JOIN jobs j ON j.id = jc.job_id").
		Order("jc.created_at DESC").
		Limit(limit).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	// If no job candidates have been assigned yet, display newly added candidate profiles
	if len(rows) == 0 {
		var candidates []model.Candidate
		if err := r.db.WithContext(ctx).Order("created_at DESC").Limit(limit).Find(&candidates).Error; err == nil {
			for _, c := range candidates {
				rows = append(rows, RecentCandidateRow{
					ID:          c.ID,
					CandidateID: c.ID,
					FullName:    c.FullName,
					Email:       c.Email,
					JobID:       "",
					JobTitle:    "—",
					Status:      "—",
					AssignedAt:  c.CreatedAt,
				})
			}
		}
	}

	return rows, nil
}

func (r *gormDashboardRepository) GetOpenJobsWithCounts(ctx context.Context, limit int) ([]OpenJobRow, error) {
	if limit <= 0 || limit > 50 {
		limit = 5
	}
	var rows []OpenJobRow
	err := r.db.WithContext(ctx).Table("jobs j").
		Select("j.id, j.code, j.title, j.department, j.location, j.employment_type, j.status, j.updated_at, COUNT(jc.candidate_id) as candidates_count").
		Joins("LEFT JOIN job_candidates jc ON jc.job_id = j.id").
		Where("j.status = ?", model.JobStatusOpen).
		Group("j.id, j.code, j.title, j.department, j.location, j.employment_type, j.status, j.updated_at").
		Order("j.updated_at DESC").
		Limit(limit).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *gormDashboardRepository) GetRecentActivity(ctx context.Context, limit int) ([]DashboardActivityRow, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	var rows []DashboardActivityRow
	err := r.db.WithContext(ctx).Table("audit_logs a").
		Select("a.id, a.action, COALESCE(u.name, 'Recruiter') as actor_name, a.job_id, j.title as job_title, a.candidate_id, c.full_name as candidate_name, a.metadata, a.created_at").
		Joins("LEFT JOIN users u ON u.id = a.actor_id").
		Joins("LEFT JOIN jobs j ON j.id = a.job_id").
		Joins("LEFT JOIN candidates c ON c.id = a.candidate_id").
		Order("a.created_at DESC").
		Limit(limit).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}
