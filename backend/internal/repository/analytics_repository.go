package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"hirescope/backend/internal/model"

	"gorm.io/gorm"
)

// AnalyticsFilter defines multidimensional filters for analytics queries.
type AnalyticsFilter struct {
	From        *time.Time
	To          *time.Time
	JobID       string
	Department  string
	JobStatus   string
	RecruiterID string
}

// SummaryCounts encapsulates headline operational KPIs.
type SummaryCounts struct {
	TotalCandidates         int64 `json:"total_candidates"`
	ActiveJobs              int64 `json:"active_jobs"`
	TotalInterviews         int64 `json:"total_interviews"`
	ScheduledInterviews     int64 `json:"scheduled_interviews"`
	CompletedInterviews     int64 `json:"completed_interviews"`
	CancelledInterviews     int64 `json:"cancelled_interviews"`
	ShortlistedApplications int64 `json:"shortlisted_applications"`
}

// FunnelCounts models the progression of candidates through recruitment stages.
type FunnelCounts struct {
	TotalCandidates              int64 `json:"total_candidates"`
	ReviewedCandidates           int64 `json:"reviewed_candidates"`
	ShortlistedApplications      int64 `json:"shortlisted_applications"`
	InterviewedCandidates        int64 `json:"interviewed_candidates"`
	CompletedInterviewCandidates int64 `json:"completed_interview_candidates"`
}

// StatusCountItem represents a grouped count item (e.g. for status, modality, stage).
type StatusCountItem struct {
	Key   string `json:"key"`
	Count int64  `json:"count"`
}

// RequirementMatchCount encapsulates match distribution grouped by importance and status.
type RequirementMatchCount struct {
	Importance  string `json:"importance"`
	MatchStatus string `json:"match_status"`
	Count       int64  `json:"count"`
}

// TrendDataPoint models candidates registered vs interviews scheduled over a time period.
type TrendDataPoint struct {
	Period          string `json:"period"`
	Date            string `json:"date"`
	CandidatesCount int64  `json:"candidates_count"`
	InterviewsCount int64  `json:"interviews_count"`
}

// JobPerformanceRow contains aggregated metrics per job requisition.
type JobPerformanceRow struct {
	JobID               string          `json:"job_id"`
	JobCode             string          `json:"job_code"`
	JobTitle            string          `json:"job_title"`
	Department          string          `json:"department"`
	Status              model.JobStatus `json:"status"`
	TotalCandidates     int64           `json:"total_candidates"`
	Reviewed            int64           `json:"reviewed"`
	Shortlisted         int64           `json:"shortlisted"`
	Interviews          int64           `json:"interviews"`
	CompletedInterviews int64           `json:"completed_interviews"`
}

// RecruiterActivityRow aggregates operational workload metrics for a recruiter user.
type RecruiterActivityRow struct {
	RecruiterID         string     `json:"recruiter_id"`
	RecruiterName       string     `json:"recruiter_name"`
	RecruiterEmail      string     `json:"recruiter_email"`
	Role                string     `json:"role"`
	CandidatesReviewed  int64      `json:"candidates_reviewed"`
	NotesCreated        int64      `json:"notes_created"`
	InterviewsScheduled int64      `json:"interviews_scheduled"`
	InterviewsCompleted int64      `json:"interviews_completed"`
	LastActivityAt      *time.Time `json:"last_activity_at,omitempty"`
}

// TimeMetrics measures average elapsed operational time intervals.
type TimeMetrics struct {
	AvgAssignmentToReviewHours    *float64 `json:"avg_assignment_to_review_hours"`
	AvgReviewToInterviewHours     *float64 `json:"avg_review_to_interview_hours"`
	AvgInterviewToCompletionHours *float64 `json:"avg_interview_to_completion_hours"`
	SampleSize                    int64    `json:"sample_size"`
}

// AnalyticsRepository provides high-performance SQL aggregation methods for analytics.
type AnalyticsRepository interface {
	GetSummaryCounts(ctx context.Context, filter AnalyticsFilter) (*SummaryCounts, error)
	GetFunnelCounts(ctx context.Context, filter AnalyticsFilter) (*FunnelCounts, error)
	GetPipelineDistribution(ctx context.Context, filter AnalyticsFilter) ([]StatusCountItem, error)
	GetScreeningDistribution(ctx context.Context, filter AnalyticsFilter) ([]StatusCountItem, error)
	GetScreeningRequirementMatches(ctx context.Context, filter AnalyticsFilter) ([]RequirementMatchCount, error)
	GetInterviewStatusDistribution(ctx context.Context, filter AnalyticsFilter) ([]StatusCountItem, error)
	GetInterviewTypeDistribution(ctx context.Context, filter AnalyticsFilter) ([]StatusCountItem, error)
	GetInterviewStageDistribution(ctx context.Context, filter AnalyticsFilter) ([]StatusCountItem, error)
	GetRecruitmentTrends(ctx context.Context, filter AnalyticsFilter, interval string) ([]TrendDataPoint, error)
	GetJobPerformance(ctx context.Context, filter AnalyticsFilter) ([]JobPerformanceRow, error)
	GetRecruiterActivity(ctx context.Context, filter AnalyticsFilter) ([]RecruiterActivityRow, error)
	GetTimeMetrics(ctx context.Context, filter AnalyticsFilter) (*TimeMetrics, error)
	GetDistinctDepartments(ctx context.Context) ([]string, error)
}

type gormAnalyticsRepository struct {
	db *gorm.DB
}

// NewAnalyticsRepository constructs a new AnalyticsRepository with GORM.
func NewAnalyticsRepository(db *gorm.DB) AnalyticsRepository {
	return &gormAnalyticsRepository{db: db}
}

func (r *gormAnalyticsRepository) GetSummaryCounts(ctx context.Context, filter AnalyticsFilter) (*SummaryCounts, error) {
	counts := &SummaryCounts{}

	// 1. Total Distinct Candidates matching filter
	candQuery := r.db.WithContext(ctx).Table("job_candidates jc").
		Joins("JOIN jobs j ON jc.job_id = j.id").
		Select("COUNT(DISTINCT jc.candidate_id)")
	candQuery = applyJobFilters(candQuery, filter, "jc", "j")
	if filter.From != nil {
		candQuery = candQuery.Where("jc.created_at >= ?", *filter.From)
	}
	if filter.To != nil {
		candQuery = candQuery.Where("jc.created_at <= ?", *filter.To)
	}
	if err := candQuery.Scan(&counts.TotalCandidates).Error; err != nil {
		return nil, fmt.Errorf("failed to count total candidates: %w", err)
	}

	// 2. Active Jobs (OPEN status, filtered by department/recruiter)
	jobQuery := r.db.WithContext(ctx).Model(&model.Job{}).Where("status = ?", model.JobStatusOpen)
	if filter.Department != "" {
		jobQuery = jobQuery.Where("department = ?", filter.Department)
	}
	if filter.RecruiterID != "" {
		jobQuery = jobQuery.Where("created_by = ?", filter.RecruiterID)
	}
	if err := jobQuery.Count(&counts.ActiveJobs).Error; err != nil {
		return nil, fmt.Errorf("failed to count active jobs: %w", err)
	}

	// 3. Shortlisted Applications
	shortlistQuery := r.db.WithContext(ctx).Table("job_candidates jc").
		Joins("JOIN jobs j ON jc.job_id = j.id").
		Where("jc.status = ?", model.JobCandidateStatusShortlisted)
	shortlistQuery = applyJobFilters(shortlistQuery, filter, "jc", "j")
	if filter.From != nil {
		shortlistQuery = shortlistQuery.Where("jc.updated_at >= ?", *filter.From)
	}
	if filter.To != nil {
		shortlistQuery = shortlistQuery.Where("jc.updated_at <= ?", *filter.To)
	}
	if err := shortlistQuery.Count(&counts.ShortlistedApplications).Error; err != nil {
		return nil, fmt.Errorf("failed to count shortlisted applications: %w", err)
	}

	// 4. Interviews Summary (scoped by scheduled_start date range)
	itwQuery := r.db.WithContext(ctx).Table("interviews i").
		Joins("JOIN jobs j ON i.job_id = j.id").
		Select(`
			COUNT(i.id) as total_interviews,
			COUNT(CASE WHEN i.status = 'SCHEDULED' THEN 1 END) as scheduled_interviews,
			COUNT(CASE WHEN i.status = 'COMPLETED' THEN 1 END) as completed_interviews,
			COUNT(CASE WHEN i.status = 'CANCELLED' THEN 1 END) as cancelled_interviews
		`)
	itwQuery = applyJobFilters(itwQuery, filter, "i", "j")
	if filter.From != nil {
		itwQuery = itwQuery.Where("i.scheduled_start >= ?", *filter.From)
	}
	if filter.To != nil {
		itwQuery = itwQuery.Where("i.scheduled_start <= ?", *filter.To)
	}

	type interviewAgg struct {
		TotalInterviews     int64
		ScheduledInterviews int64
		CompletedInterviews int64
		CancelledInterviews int64
	}
	var itwRes interviewAgg
	if err := itwQuery.Scan(&itwRes).Error; err != nil {
		return nil, fmt.Errorf("failed to aggregate interview counts: %w", err)
	}

	counts.TotalInterviews = itwRes.TotalInterviews
	counts.ScheduledInterviews = itwRes.ScheduledInterviews
	counts.CompletedInterviews = itwRes.CompletedInterviews
	counts.CancelledInterviews = itwRes.CancelledInterviews

	return counts, nil
}

func (r *gormAnalyticsRepository) GetFunnelCounts(ctx context.Context, filter AnalyticsFilter) (*FunnelCounts, error) {
	funnel := &FunnelCounts{}

	// 1. Total Candidates & Reviewed Candidates
	baseQuery := r.db.WithContext(ctx).Table("job_candidates jc").
		Joins("JOIN jobs j ON jc.job_id = j.id").
		Select(`
			COUNT(DISTINCT jc.candidate_id) as total_candidates,
			COUNT(DISTINCT CASE WHEN jc.reviewed_at IS NOT NULL OR jc.status != 'REVIEW' THEN jc.candidate_id END) as reviewed_candidates,
			COUNT(CASE WHEN jc.status = 'SHORTLISTED' THEN 1 END) as shortlisted_applications
		`)
	baseQuery = applyJobFilters(baseQuery, filter, "jc", "j")
	if filter.From != nil {
		baseQuery = baseQuery.Where("jc.created_at >= ?", *filter.From)
	}
	if filter.To != nil {
		baseQuery = baseQuery.Where("jc.created_at <= ?", *filter.To)
	}

	type baseAgg struct {
		TotalCandidates         int64
		ReviewedCandidates      int64
		ShortlistedApplications int64
	}
	var bRes baseAgg
	if err := baseQuery.Scan(&bRes).Error; err != nil {
		return nil, fmt.Errorf("failed to aggregate funnel base: %w", err)
	}
	funnel.TotalCandidates = bRes.TotalCandidates
	funnel.ReviewedCandidates = bRes.ReviewedCandidates
	funnel.ShortlistedApplications = bRes.ShortlistedApplications

	// 2. Interviewed candidates & completed interview candidates
	itwQuery := r.db.WithContext(ctx).Table("interviews i").
		Joins("JOIN jobs j ON i.job_id = j.id").
		Select(`
			COUNT(DISTINCT i.candidate_id) as interviewed_candidates,
			COUNT(DISTINCT CASE WHEN i.status = 'COMPLETED' THEN i.candidate_id END) as completed_interview_candidates
		`)
	itwQuery = applyJobFilters(itwQuery, filter, "i", "j")
	if filter.From != nil {
		itwQuery = itwQuery.Where("i.scheduled_start >= ?", *filter.From)
	}
	if filter.To != nil {
		itwQuery = itwQuery.Where("i.scheduled_start <= ?", *filter.To)
	}

	type itwAgg struct {
		InterviewedCandidates        int64
		CompletedInterviewCandidates int64
	}
	var iRes itwAgg
	if err := itwQuery.Scan(&iRes).Error; err != nil {
		return nil, fmt.Errorf("failed to aggregate funnel interview counts: %w", err)
	}
	funnel.InterviewedCandidates = iRes.InterviewedCandidates
	funnel.CompletedInterviewCandidates = iRes.CompletedInterviewCandidates

	return funnel, nil
}

func (r *gormAnalyticsRepository) GetPipelineDistribution(ctx context.Context, filter AnalyticsFilter) ([]StatusCountItem, error) {
	query := r.db.WithContext(ctx).Table("job_candidates jc").
		Joins("JOIN jobs j ON jc.job_id = j.id").
		Select("jc.status as key, COUNT(jc.id) as count").
		Group("jc.status").
		Order("count DESC")

	query = applyJobFilters(query, filter, "jc", "j")
	if filter.From != nil {
		query = query.Where("jc.updated_at >= ?", *filter.From)
	}
	if filter.To != nil {
		query = query.Where("jc.updated_at <= ?", *filter.To)
	}

	var results []StatusCountItem
	if err := query.Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("failed to get pipeline distribution: %w", err)
	}
	return results, nil
}

func (r *gormAnalyticsRepository) GetScreeningDistribution(ctx context.Context, filter AnalyticsFilter) ([]StatusCountItem, error) {
	query := r.db.WithContext(ctx).Table("screening_results sr").
		Joins("JOIN jobs j ON sr.job_id = j.id").
		Select("sr.status as key, COUNT(sr.id) as count").
		Group("sr.status").
		Order("count DESC")

	query = applyJobFilters(query, filter, "sr", "j")
	if filter.From != nil {
		query = query.Where("sr.created_at >= ?", *filter.From)
	}
	if filter.To != nil {
		query = query.Where("sr.created_at <= ?", *filter.To)
	}

	var results []StatusCountItem
	if err := query.Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("failed to get screening distribution: %w", err)
	}
	return results, nil
}

func (r *gormAnalyticsRepository) GetScreeningRequirementMatches(ctx context.Context, filter AnalyticsFilter) ([]RequirementMatchCount, error) {
	query := r.db.WithContext(ctx).Table("screening_matches sm").
		Joins("JOIN screening_results sr ON sm.screening_result_id = sr.id").
		Joins("JOIN job_requirements jr ON sm.requirement_id = jr.id").
		Joins("JOIN jobs j ON sr.job_id = j.id").
		Select("jr.importance as importance, sm.status as match_status, COUNT(sm.id) as count").
		Group("jr.importance, sm.status")

	query = applyJobFilters(query, filter, "sr", "j")
	if filter.From != nil {
		query = query.Where("sr.created_at >= ?", *filter.From)
	}
	if filter.To != nil {
		query = query.Where("sr.created_at <= ?", *filter.To)
	}

	var results []RequirementMatchCount
	if err := query.Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("failed to get requirement matches: %w", err)
	}
	return results, nil
}

func (r *gormAnalyticsRepository) GetInterviewStatusDistribution(ctx context.Context, filter AnalyticsFilter) ([]StatusCountItem, error) {
	query := r.db.WithContext(ctx).Table("interviews i").
		Joins("JOIN jobs j ON i.job_id = j.id").
		Select("i.status as key, COUNT(i.id) as count").
		Group("i.status").
		Order("count DESC")

	query = applyJobFilters(query, filter, "i", "j")
	if filter.From != nil {
		query = query.Where("i.scheduled_start >= ?", *filter.From)
	}
	if filter.To != nil {
		query = query.Where("i.scheduled_start <= ?", *filter.To)
	}

	var results []StatusCountItem
	if err := query.Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("failed to get interview status distribution: %w", err)
	}
	return results, nil
}

func (r *gormAnalyticsRepository) GetInterviewTypeDistribution(ctx context.Context, filter AnalyticsFilter) ([]StatusCountItem, error) {
	query := r.db.WithContext(ctx).Table("interviews i").
		Joins("JOIN jobs j ON i.job_id = j.id").
		Select("i.interview_type as key, COUNT(i.id) as count").
		Group("i.interview_type").
		Order("count DESC")

	query = applyJobFilters(query, filter, "i", "j")
	if filter.From != nil {
		query = query.Where("i.scheduled_start >= ?", *filter.From)
	}
	if filter.To != nil {
		query = query.Where("i.scheduled_start <= ?", *filter.To)
	}

	var results []StatusCountItem
	if err := query.Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("failed to get interview type distribution: %w", err)
	}
	return results, nil
}

func (r *gormAnalyticsRepository) GetInterviewStageDistribution(ctx context.Context, filter AnalyticsFilter) ([]StatusCountItem, error) {
	query := r.db.WithContext(ctx).Table("interviews i").
		Joins("JOIN jobs j ON i.job_id = j.id").
		Select("i.stage as key, COUNT(i.id) as count").
		Group("i.stage").
		Order("count DESC")

	query = applyJobFilters(query, filter, "i", "j")
	if filter.From != nil {
		query = query.Where("i.scheduled_start >= ?", *filter.From)
	}
	if filter.To != nil {
		query = query.Where("i.scheduled_start <= ?", *filter.To)
	}

	var results []StatusCountItem
	if err := query.Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("failed to get interview stage distribution: %w", err)
	}
	return results, nil
}

func (r *gormAnalyticsRepository) GetRecruitmentTrends(ctx context.Context, filter AnalyticsFilter, interval string) ([]TrendDataPoint, error) {
	trunc := "day"
	if interval == "week" {
		trunc = "week"
	} else if interval == "month" {
		trunc = "month"
	}

	// Group candidates added by truncated date
	candQuery := r.db.WithContext(ctx).Table("job_candidates jc").
		Joins("JOIN jobs j ON jc.job_id = j.id").
		Select(fmt.Sprintf("DATE_TRUNC('%s', jc.created_at) as period_date, COUNT(DISTINCT jc.candidate_id) as count", trunc)).
		Group("period_date").
		Order("period_date ASC")

	candQuery = applyJobFilters(candQuery, filter, "jc", "j")
	if filter.From != nil {
		candQuery = candQuery.Where("jc.created_at >= ?", *filter.From)
	}
	if filter.To != nil {
		candQuery = candQuery.Where("jc.created_at <= ?", *filter.To)
	}

	type trendRow struct {
		PeriodDate time.Time
		Count      int64
	}
	var candRows []trendRow
	if err := candQuery.Scan(&candRows).Error; err != nil {
		return nil, fmt.Errorf("failed to aggregate candidate trends: %w", err)
	}

	// Group interviews scheduled by truncated date
	itwQuery := r.db.WithContext(ctx).Table("interviews i").
		Joins("JOIN jobs j ON i.job_id = j.id").
		Select(fmt.Sprintf("DATE_TRUNC('%s', i.scheduled_start) as period_date, COUNT(i.id) as count", trunc)).
		Group("period_date").
		Order("period_date ASC")

	itwQuery = applyJobFilters(itwQuery, filter, "i", "j")
	if filter.From != nil {
		itwQuery = itwQuery.Where("i.scheduled_start >= ?", *filter.From)
	}
	if filter.To != nil {
		itwQuery = itwQuery.Where("i.scheduled_start <= ?", *filter.To)
	}

	var itwRows []trendRow
	if err := itwQuery.Scan(&itwRows).Error; err != nil {
		return nil, fmt.Errorf("failed to aggregate interview trends: %w", err)
	}

	// Merge candidate & interview timeline
	pointsMap := make(map[string]*TrendDataPoint)
	var orderedDates []string

	for _, cr := range candRows {
		dateStr := cr.PeriodDate.Format("2006-01-02")
		if _, exists := pointsMap[dateStr]; !exists {
			pointsMap[dateStr] = &TrendDataPoint{
				Period: dateStr,
				Date:   dateStr,
			}
			orderedDates = append(orderedDates, dateStr)
		}
		pointsMap[dateStr].CandidatesCount = cr.Count
	}

	for _, ir := range itwRows {
		dateStr := ir.PeriodDate.Format("2006-01-02")
		if _, exists := pointsMap[dateStr]; !exists {
			pointsMap[dateStr] = &TrendDataPoint{
				Period: dateStr,
				Date:   dateStr,
			}
			orderedDates = append(orderedDates, dateStr)
		}
		pointsMap[dateStr].InterviewsCount = ir.Count
	}

	results := make([]TrendDataPoint, 0, len(orderedDates))
	for _, d := range orderedDates {
		results = append(results, *pointsMap[d])
	}

	return results, nil
}

func (r *gormAnalyticsRepository) GetJobPerformance(ctx context.Context, filter AnalyticsFilter) ([]JobPerformanceRow, error) {
	query := r.db.WithContext(ctx).Table("jobs j").
		Select(`
			j.id as job_id,
			j.code as job_code,
			j.title as job_title,
			j.department as department,
			j.status as status,
			COUNT(DISTINCT jc.candidate_id) as total_candidates,
			COUNT(DISTINCT CASE WHEN jc.reviewed_at IS NOT NULL OR jc.status != 'REVIEW' THEN jc.candidate_id END) as reviewed,
			COUNT(CASE WHEN jc.status = 'SHORTLISTED' THEN 1 END) as shortlisted,
			COUNT(DISTINCT i.id) as interviews,
			COUNT(DISTINCT CASE WHEN i.status = 'COMPLETED' THEN i.id END) as completed_interviews
		`).
		Joins("LEFT JOIN job_candidates jc ON jc.job_id = j.id").
		Joins("LEFT JOIN interviews i ON i.job_id = j.id").
		Group("j.id, j.code, j.title, j.department, j.status").
		Order("total_candidates DESC, j.title ASC")

	if filter.JobID != "" {
		query = query.Where("j.id = ?", filter.JobID)
	}
	if filter.Department != "" {
		query = query.Where("j.department = ?", filter.Department)
	}
	if filter.JobStatus != "" {
		query = query.Where("j.status = ?", filter.JobStatus)
	}
	if filter.RecruiterID != "" {
		query = query.Where("j.created_by = ?", filter.RecruiterID)
	}

	var rows []JobPerformanceRow
	if err := query.Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to aggregate job performance: %w", err)
	}
	return rows, nil
}

func (r *gormAnalyticsRepository) GetRecruiterActivity(ctx context.Context, filter AnalyticsFilter) ([]RecruiterActivityRow, error) {
	query := r.db.WithContext(ctx).Table("users u").
		Where("u.role IN ('ADMIN', 'RECRUITER')").
		Select(`
			u.id as recruiter_id,
			u.name as recruiter_name,
			u.email as recruiter_email,
			u.role as role,
			(SELECT COUNT(jc.id) FROM job_candidates jc WHERE jc.reviewed_by = u.id) as candidates_reviewed,
			(SELECT COUNT(cn.id) FROM candidate_notes cn WHERE cn.author_id = u.id) as notes_created,
			(SELECT COUNT(i.id) FROM interviews i WHERE i.created_by = u.id) as interviews_scheduled,
			(SELECT COUNT(ic.id) FROM interviews ic WHERE ic.completed_by = u.id AND ic.status = 'COMPLETED') as interviews_completed,
			(SELECT MAX(al.created_at) FROM audit_logs al WHERE al.actor_id = u.id) as last_activity_at
		`).
		Order("candidates_reviewed DESC, interviews_scheduled DESC, u.name ASC")

	if filter.RecruiterID != "" {
		query = query.Where("u.id = ?", filter.RecruiterID)
	}

	var rows []RecruiterActivityRow
	if err := query.Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to aggregate recruiter activity: %w", err)
	}
	return rows, nil
}

func (r *gormAnalyticsRepository) GetTimeMetrics(ctx context.Context, filter AnalyticsFilter) (*TimeMetrics, error) {
	metrics := &TimeMetrics{}

	// 1. Avg Assignment to Review (jc.created_at to jc.reviewed_at)
	assignQuery := r.db.WithContext(ctx).Table("job_candidates jc").
		Joins("JOIN jobs j ON jc.job_id = j.id").
		Where("jc.reviewed_at IS NOT NULL AND jc.reviewed_at >= jc.created_at").
		Select("AVG(EXTRACT(EPOCH FROM (jc.reviewed_at - jc.created_at))/3600.0) as avg_hours, COUNT(jc.id) as count")

	assignQuery = applyJobFilters(assignQuery, filter, "jc", "j")
	if filter.From != nil {
		assignQuery = assignQuery.Where("jc.created_at >= ?", *filter.From)
	}
	if filter.To != nil {
		assignQuery = assignQuery.Where("jc.created_at <= ?", *filter.To)
	}

	type avgResult struct {
		AvgHours *float64
		Count    int64
	}
	var res1 avgResult
	if err := assignQuery.Scan(&res1).Error; err == nil && res1.Count > 0 {
		metrics.AvgAssignmentToReviewHours = res1.AvgHours
		metrics.SampleSize = res1.Count
	}

	// 2. Avg Interview scheduled_start to completion
	itwQuery := r.db.WithContext(ctx).Table("interviews i").
		Joins("JOIN jobs j ON i.job_id = j.id").
		Where("i.status = 'COMPLETED' AND i.completed_at IS NOT NULL AND i.completed_at >= i.scheduled_start").
		Select("AVG(EXTRACT(EPOCH FROM (i.completed_at - i.scheduled_start))/3600.0) as avg_hours, COUNT(i.id) as count")

	itwQuery = applyJobFilters(itwQuery, filter, "i", "j")
	if filter.From != nil {
		itwQuery = itwQuery.Where("i.scheduled_start >= ?", *filter.From)
	}
	if filter.To != nil {
		itwQuery = itwQuery.Where("i.scheduled_start <= ?", *filter.To)
	}

	var res2 avgResult
	if err := itwQuery.Scan(&res2).Error; err == nil && res2.Count > 0 {
		metrics.AvgInterviewToCompletionHours = res2.AvgHours
		if res2.Count > metrics.SampleSize {
			metrics.SampleSize = res2.Count
		}
	}

	return metrics, nil
}

func (r *gormAnalyticsRepository) GetDistinctDepartments(ctx context.Context) ([]string, error) {
	var depts []string
	err := r.db.WithContext(ctx).
		Model(&model.Job{}).
		Where("department IS NOT NULL AND TRIM(department) != ''").
		Distinct("department").
		Order("department ASC").
		Pluck("department", &depts).Error
	if err != nil {
		return nil, fmt.Errorf("failed to fetch distinct departments: %w", err)
	}
	return depts, nil
}

func applyJobFilters(query *gorm.DB, filter AnalyticsFilter, entityAlias, jobAlias string) *gorm.DB {
	q := query
	if filter.JobID != "" {
		q = q.Where(fmt.Sprintf("%s.job_id = ?", entityAlias), filter.JobID)
	}
	if filter.Department != "" {
		q = q.Where(fmt.Sprintf("%s.department = ?", jobAlias), filter.Department)
	}
	if filter.JobStatus != "" {
		q = q.Where(fmt.Sprintf("%s.status = ?", jobAlias), strings.ToUpper(filter.JobStatus))
	}
	if filter.RecruiterID != "" {
		q = q.Where(fmt.Sprintf("%s.created_by = ?", jobAlias), filter.RecruiterID)
	}
	return q
}
