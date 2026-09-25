package tests

import (
	"context"
	"encoding/csv"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"hirescope/backend/internal/handler"
	"hirescope/backend/internal/model"
	"hirescope/backend/internal/repository"
	"hirescope/backend/internal/service"

	"github.com/gin-gonic/gin"
)

type memoryAnalyticsRepo struct {
	mu           sync.RWMutex
	summary      *repository.SummaryCounts
	funnel       *repository.FunnelCounts
	pipeline     []repository.StatusCountItem
	screening    []repository.StatusCountItem
	reqMatches   []repository.RequirementMatchCount
	itwStatus    []repository.StatusCountItem
	itwType      []repository.StatusCountItem
	itwStage     []repository.StatusCountItem
	trends       []repository.TrendDataPoint
	jobs         []repository.JobPerformanceRow
	recruiters   []repository.RecruiterActivityRow
	timeMetrics  *repository.TimeMetrics
	departments  []string
	lastFilter   *repository.AnalyticsFilter
	lastInterval string
}

func newMemoryAnalyticsRepo() *memoryAnalyticsRepo {
	return &memoryAnalyticsRepo{
		summary: &repository.SummaryCounts{
			TotalCandidates:         42,
			ActiveJobs:              8,
			TotalInterviews:         25,
			ScheduledInterviews:     10,
			CompletedInterviews:     12,
			CancelledInterviews:     3,
			ShortlistedApplications: 15,
		},
		funnel: &repository.FunnelCounts{
			TotalCandidates:              50,
			ReviewedCandidates:           40,
			ShortlistedApplications:      20,
			InterviewedCandidates:        15,
			CompletedInterviewCandidates: 10,
		},
		pipeline: []repository.StatusCountItem{
			{Key: "REVIEW", Count: 10},
			{Key: "SHORTLISTED", Count: 20},
			{Key: "REJECTED", Count: 12},
		},
		screening: []repository.StatusCountItem{
			{Key: "QUALIFIED", Count: 25},
			{Key: "REVIEW", Count: 12},
			{Key: "NOT_QUALIFIED", Count: 5},
		},
		reqMatches: []repository.RequirementMatchCount{
			{Importance: "MUST_HAVE", MatchStatus: "PASSED", Count: 50},
			{Importance: "MUST_HAVE", MatchStatus: "FAILED", Count: 10},
			{Importance: "NICE_TO_HAVE", MatchStatus: "PASSED", Count: 30},
			{Importance: "NICE_TO_HAVE", MatchStatus: "FAILED", Count: 15},
		},
		itwStatus: []repository.StatusCountItem{
			{Key: "SCHEDULED", Count: 10},
			{Key: "COMPLETED", Count: 12},
			{Key: "CANCELLED", Count: 3},
		},
		itwType: []repository.StatusCountItem{
			{Key: "ONLINE", Count: 18},
			{Key: "ONSITE", Count: 5},
			{Key: "PHONE", Count: 2},
		},
		itwStage: []repository.StatusCountItem{
			{Key: "HR_INTERVIEW", Count: 10},
			{Key: "TECHNICAL", Count: 8},
			{Key: "FINAL", Count: 7},
		},
		trends: []repository.TrendDataPoint{
			{Period: "2026-09-01", Date: "2026-09-01", CandidatesCount: 5, InterviewsCount: 2},
			{Period: "2026-09-02", Date: "2026-09-02", CandidatesCount: 8, InterviewsCount: 4},
		},
		jobs: []repository.JobPerformanceRow{
			{
				JobID:               "job-1",
				JobCode:             "JOB-2026-0001",
				JobTitle:            "Staff Backend Engineer",
				Department:          "Engineering",
				Status:              model.JobStatusOpen,
				TotalCandidates:     20,
				Reviewed:            16,
				Shortlisted:         10,
				Interviews:          8,
				CompletedInterviews: 6,
			},
		},
		recruiters: []repository.RecruiterActivityRow{
			{
				RecruiterID:         "rec-1",
				RecruiterName:       "Sarah Connor",
				RecruiterEmail:      "sarah@example.com",
				Role:                "RECRUITER",
				CandidatesReviewed:  18,
				NotesCreated:        24,
				InterviewsScheduled: 10,
				InterviewsCompleted: 8,
			},
		},
		timeMetrics: &repository.TimeMetrics{
			SampleSize: 15,
		},
		departments: []string{"Engineering", "Product", "Design", "Marketing"},
	}
}

func (r *memoryAnalyticsRepo) GetSummaryCounts(ctx context.Context, filter repository.AnalyticsFilter) (*repository.SummaryCounts, error) {
	r.mu.Lock()
	r.lastFilter = &filter
	r.mu.Unlock()
	return r.summary, nil
}

func (r *memoryAnalyticsRepo) GetFunnelCounts(ctx context.Context, filter repository.AnalyticsFilter) (*repository.FunnelCounts, error) {
	r.mu.Lock()
	r.lastFilter = &filter
	r.mu.Unlock()
	return r.funnel, nil
}

func (r *memoryAnalyticsRepo) GetPipelineDistribution(ctx context.Context, filter repository.AnalyticsFilter) ([]repository.StatusCountItem, error) {
	return r.pipeline, nil
}

func (r *memoryAnalyticsRepo) GetScreeningDistribution(ctx context.Context, filter repository.AnalyticsFilter) ([]repository.StatusCountItem, error) {
	return r.screening, nil
}

func (r *memoryAnalyticsRepo) GetScreeningRequirementMatches(ctx context.Context, filter repository.AnalyticsFilter) ([]repository.RequirementMatchCount, error) {
	return r.reqMatches, nil
}

func (r *memoryAnalyticsRepo) GetInterviewStatusDistribution(ctx context.Context, filter repository.AnalyticsFilter) ([]repository.StatusCountItem, error) {
	return r.itwStatus, nil
}

func (r *memoryAnalyticsRepo) GetInterviewTypeDistribution(ctx context.Context, filter repository.AnalyticsFilter) ([]repository.StatusCountItem, error) {
	return r.itwType, nil
}

func (r *memoryAnalyticsRepo) GetInterviewStageDistribution(ctx context.Context, filter repository.AnalyticsFilter) ([]repository.StatusCountItem, error) {
	return r.itwStage, nil
}

func (r *memoryAnalyticsRepo) GetRecruitmentTrends(ctx context.Context, filter repository.AnalyticsFilter, interval string) ([]repository.TrendDataPoint, error) {
	r.mu.Lock()
	r.lastInterval = interval
	r.mu.Unlock()
	return r.trends, nil
}

func (r *memoryAnalyticsRepo) GetJobPerformance(ctx context.Context, filter repository.AnalyticsFilter) ([]repository.JobPerformanceRow, error) {
	r.mu.Lock()
	r.lastFilter = &filter
	r.mu.Unlock()
	return r.jobs, nil
}

func (r *memoryAnalyticsRepo) GetRecruiterActivity(ctx context.Context, filter repository.AnalyticsFilter) ([]repository.RecruiterActivityRow, error) {
	r.mu.Lock()
	r.lastFilter = &filter
	r.mu.Unlock()
	return r.recruiters, nil
}

func (r *memoryAnalyticsRepo) GetTimeMetrics(ctx context.Context, filter repository.AnalyticsFilter) (*repository.TimeMetrics, error) {
	return r.timeMetrics, nil
}

func (r *memoryAnalyticsRepo) GetDistinctDepartments(ctx context.Context) ([]string, error) {
	return r.departments, nil
}

type memoryAuditRepo struct {
	mu   sync.RWMutex
	logs []*model.AuditLog
}

func newMemoryAuditRepo() *memoryAuditRepo {
	return &memoryAuditRepo{logs: make([]*model.AuditLog, 0)}
}

func (a *memoryAuditRepo) Create(ctx context.Context, log *model.AuditLog) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.logs = append(a.logs, log)
	return nil
}

func setupTestAnalyticsRouter(svc service.AnalyticsService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := handler.NewAnalyticsHandler(svc)

	v1 := r.Group("/api/v1/analytics")
	{
		v1.GET("/overview", h.GetOverview)
		v1.GET("/jobs", h.GetJobPerformance)
		v1.GET("/recruiters", h.GetRecruiterActivity)
		v1.GET("/departments", h.GetDepartments)
		v1.GET("/export", func(c *gin.Context) {
			c.Set("userID", "user-admin-1")
			h.ExportCSV(c)
		})
	}
	return r
}

// 1. Summary counts calculation
func TestSummaryCountsCalculation(t *testing.T) {
	repo := newMemoryAnalyticsRepo()
	audit := newMemoryAuditRepo()
	svc := service.NewAnalyticsService(repo, audit)

	resp, err := svc.GetOverview(context.Background(), service.AnalyticsQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Summary.TotalCandidates != 42 {
		t.Errorf("expected TotalCandidates 42, got %d", resp.Summary.TotalCandidates)
	}
	if resp.Summary.ActiveJobs != 8 {
		t.Errorf("expected ActiveJobs 8, got %d", resp.Summary.ActiveJobs)
	}
	if resp.Summary.TotalInterviews != 25 {
		t.Errorf("expected TotalInterviews 25, got %d", resp.Summary.TotalInterviews)
	}
	if resp.Summary.ScheduledInterviews != 10 {
		t.Errorf("expected ScheduledInterviews 10, got %d", resp.Summary.ScheduledInterviews)
	}
	if resp.Summary.CompletedInterviews != 12 {
		t.Errorf("expected CompletedInterviews 12, got %d", resp.Summary.CompletedInterviews)
	}
	if resp.Summary.CancelledInterviews != 3 {
		t.Errorf("expected CancelledInterviews 3, got %d", resp.Summary.CancelledInterviews)
	}
	if resp.Summary.ShortlistedApplications != 15 {
		t.Errorf("expected ShortlistedApplications 15, got %d", resp.Summary.ShortlistedApplications)
	}
}

// 2. Pipeline funnel progression
func TestPipelineFunnelProgression(t *testing.T) {
	repo := newMemoryAnalyticsRepo()
	audit := newMemoryAuditRepo()
	svc := service.NewAnalyticsService(repo, audit)

	resp, err := svc.GetOverview(context.Background(), service.AnalyticsQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	funnel := resp.Funnel
	if funnel.TotalCandidates != 50 {
		t.Errorf("expected 50 total candidates, got %d", funnel.TotalCandidates)
	}
	if funnel.ReviewedCandidates != 40 {
		t.Errorf("expected 40 reviewed candidates, got %d", funnel.ReviewedCandidates)
	}
	// Review rate: 40/50 = 80.0%
	if funnel.ReviewRate != 80.0 {
		t.Errorf("expected review rate 80.0, got %f", funnel.ReviewRate)
	}
	// Shortlist rate: 20/40 = 50.0%
	if funnel.ShortlistRate != 50.0 {
		t.Errorf("expected shortlist rate 50.0, got %f", funnel.ShortlistRate)
	}
	// Interview rate: 15/20 = 75.0%
	if funnel.InterviewRate != 75.0 {
		t.Errorf("expected interview rate 75.0, got %f", funnel.InterviewRate)
	}
	// Completion rate: 10/15 = 66.7%
	if funnel.CompletionRate != 66.7 {
		t.Errorf("expected completion rate 66.7, got %f", funnel.CompletionRate)
	}
	if len(funnel.Stages) != 5 {
		t.Errorf("expected 5 funnel stages, got %d", len(funnel.Stages))
	}
}

// 3. Screening status breakdown
func TestScreeningStatusBreakdown(t *testing.T) {
	repo := newMemoryAnalyticsRepo()
	svc := service.NewAnalyticsService(repo, nil)

	resp, err := svc.GetOverview(context.Background(), service.AnalyticsQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Screening.Total != 42 {
		t.Errorf("expected total screening 42, got %d", resp.Screening.Total)
	}
	if len(resp.Screening.ByStatus) != 3 {
		t.Fatalf("expected 3 status items, got %d", len(resp.Screening.ByStatus))
	}
	// Qualified: 25 / 42 * 100 = 59.5%
	if resp.Screening.ByStatus[0].Key != "QUALIFIED" || resp.Screening.ByStatus[0].Percentage != 59.5 {
		t.Errorf("expected QUALIFIED 59.5%%, got %s %f", resp.Screening.ByStatus[0].Key, resp.Screening.ByStatus[0].Percentage)
	}
}

// 4. Screening requirement match distribution
func TestScreeningRequirementMatchDistribution(t *testing.T) {
	repo := newMemoryAnalyticsRepo()
	svc := service.NewAnalyticsService(repo, nil)

	resp, err := svc.GetOverview(context.Background(), service.AnalyticsQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Screening.RequirementMatches) != 4 {
		t.Fatalf("expected 4 requirement match items, got %d", len(resp.Screening.RequirementMatches))
	}
	if resp.Screening.RequirementMatches[0].Importance != "MUST_HAVE" || resp.Screening.RequirementMatches[0].Count != 50 {
		t.Errorf("unexpected match item: %+v", resp.Screening.RequirementMatches[0])
	}
}

// 5. Interview distribution by status
func TestInterviewDistributionByStatus(t *testing.T) {
	repo := newMemoryAnalyticsRepo()
	svc := service.NewAnalyticsService(repo, nil)

	resp, err := svc.GetOverview(context.Background(), service.AnalyticsQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Interviews.ByStatus) != 3 {
		t.Fatalf("expected 3 interview status items, got %d", len(resp.Interviews.ByStatus))
	}
	if resp.Interviews.ByStatus[0].Key != "SCHEDULED" || resp.Interviews.ByStatus[0].Count != 10 {
		t.Errorf("unexpected status item: %+v", resp.Interviews.ByStatus[0])
	}
}

// 6. Interview distribution by modality / type
func TestInterviewDistributionByType(t *testing.T) {
	repo := newMemoryAnalyticsRepo()
	svc := service.NewAnalyticsService(repo, nil)

	resp, err := svc.GetOverview(context.Background(), service.AnalyticsQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Interviews.ByType) != 3 {
		t.Fatalf("expected 3 interview types, got %d", len(resp.Interviews.ByType))
	}
	if resp.Interviews.ByType[0].Key != "ONLINE" || resp.Interviews.ByType[0].Count != 18 {
		t.Errorf("unexpected type item: %+v", resp.Interviews.ByType[0])
	}
}

// 7. Interview distribution by stage
func TestInterviewDistributionByStage(t *testing.T) {
	repo := newMemoryAnalyticsRepo()
	svc := service.NewAnalyticsService(repo, nil)

	resp, err := svc.GetOverview(context.Background(), service.AnalyticsQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Interviews.ByStage) != 3 {
		t.Fatalf("expected 3 interview stages, got %d", len(resp.Interviews.ByStage))
	}
	if resp.Interviews.ByStage[0].Key != "HR_INTERVIEW" || resp.Interviews.ByStage[0].Count != 10 {
		t.Errorf("unexpected stage item: %+v", resp.Interviews.ByStage[0])
	}
}

// 8. Recruitment trend time-series aggregation by day
func TestRecruitmentTrendsByDay(t *testing.T) {
	repo := newMemoryAnalyticsRepo()
	svc := service.NewAnalyticsService(repo, nil)

	resp, err := svc.GetOverview(context.Background(), service.AnalyticsQuery{Interval: "day"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Trends) != 2 {
		t.Fatalf("expected 2 trend data points, got %d", len(resp.Trends))
	}
	if repo.lastInterval != "day" {
		t.Errorf("expected interval day, got %s", repo.lastInterval)
	}
}

// 9. Recruitment trend time-series aggregation by week
func TestRecruitmentTrendsByWeek(t *testing.T) {
	repo := newMemoryAnalyticsRepo()
	svc := service.NewAnalyticsService(repo, nil)

	_, err := svc.GetOverview(context.Background(), service.AnalyticsQuery{Interval: "week"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.lastInterval != "week" {
		t.Errorf("expected interval week, got %s", repo.lastInterval)
	}
}

// 10. Recruitment trend time-series aggregation by month
func TestRecruitmentTrendsByMonth(t *testing.T) {
	repo := newMemoryAnalyticsRepo()
	svc := service.NewAnalyticsService(repo, nil)

	_, err := svc.GetOverview(context.Background(), service.AnalyticsQuery{Interval: "month"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.lastInterval != "month" {
		t.Errorf("expected interval month, got %s", repo.lastInterval)
	}
}

// 11. Time-to-review calculation
func TestTimeToReviewCalculation(t *testing.T) {
	repo := newMemoryAnalyticsRepo()
	val := 28.5
	repo.timeMetrics.AvgAssignmentToReviewHours = &val
	svc := service.NewAnalyticsService(repo, nil)

	resp, err := svc.GetOverview(context.Background(), service.AnalyticsQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.TimeMetrics == nil || resp.TimeMetrics.AvgAssignmentToReviewHours == nil || *resp.TimeMetrics.AvgAssignmentToReviewHours != 28.5 {
		t.Errorf("expected AvgAssignmentToReviewHours 28.5, got %v", resp.TimeMetrics)
	}
}

// 12. Time-to-interview completion calculation
func TestTimeToInterviewCompletionCalculation(t *testing.T) {
	repo := newMemoryAnalyticsRepo()
	val := 1.2
	repo.timeMetrics.AvgInterviewToCompletionHours = &val
	svc := service.NewAnalyticsService(repo, nil)

	resp, err := svc.GetOverview(context.Background(), service.AnalyticsQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.TimeMetrics == nil || resp.TimeMetrics.AvgInterviewToCompletionHours == nil || *resp.TimeMetrics.AvgInterviewToCompletionHours != 1.2 {
		t.Errorf("expected AvgInterviewToCompletionHours 1.2, got %v", resp.TimeMetrics)
	}
}

// 13. Job performance table aggregation
func TestJobPerformanceTableAggregation(t *testing.T) {
	repo := newMemoryAnalyticsRepo()
	svc := service.NewAnalyticsService(repo, nil)

	resp, err := svc.GetJobPerformance(context.Background(), service.AnalyticsQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Total != 1 {
		t.Fatalf("expected 1 job, got %d", resp.Total)
	}
	job := resp.Jobs[0]
	if job.JobCode != "JOB-2026-0001" || job.TotalCandidates != 20 || job.Reviewed != 16 {
		t.Errorf("unexpected job metrics: %+v", job)
	}
}

// 14. Job performance conversion rates
func TestJobPerformanceConversionRates(t *testing.T) {
	repo := newMemoryAnalyticsRepo()
	svc := service.NewAnalyticsService(repo, nil)

	resp, err := svc.GetJobPerformance(context.Background(), service.AnalyticsQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	job := resp.Jobs[0]
	// Review rate: 16 / 20 = 80.0%
	if job.ReviewRate != 80.0 {
		t.Errorf("expected review rate 80.0, got %f", job.ReviewRate)
	}
	// Shortlist rate: 10 / 20 = 50.0%
	if job.ShortlistRate != 50.0 {
		t.Errorf("expected shortlist rate 50.0, got %f", job.ShortlistRate)
	}
	// Interview rate: 8 / 20 = 40.0%
	if job.InterviewRate != 40.0 {
		t.Errorf("expected interview rate 40.0, got %f", job.InterviewRate)
	}
	// Completion rate: 6 / 8 = 75.0%
	if job.CompletionRate != 75.0 {
		t.Errorf("expected completion rate 75.0, got %f", job.CompletionRate)
	}
}

// 15. Recruiter operational activity aggregation
func TestRecruiterOperationalActivity(t *testing.T) {
	repo := newMemoryAnalyticsRepo()
	svc := service.NewAnalyticsService(repo, nil)

	resp, err := svc.GetRecruiterActivity(context.Background(), service.AnalyticsQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Total != 1 {
		t.Fatalf("expected 1 recruiter, got %d", resp.Total)
	}
	rec := resp.Recruiters[0]
	if rec.RecruiterName != "Sarah Connor" || rec.CandidatesReviewed != 18 || rec.NotesCreated != 24 {
		t.Errorf("unexpected recruiter activity: %+v", rec)
	}
}

// 16. Filter by date range
func TestFilterByDateRange(t *testing.T) {
	repo := newMemoryAnalyticsRepo()
	svc := service.NewAnalyticsService(repo, nil)

	_, err := svc.GetOverview(context.Background(), service.AnalyticsQuery{
		From: "2026-09-01",
		To:   "2026-09-20",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.lastFilter == nil || repo.lastFilter.From == nil || repo.lastFilter.To == nil {
		t.Fatalf("expected date filter to be populated")
	}
	if repo.lastFilter.From.Format("2006-01-02") != "2026-09-01" {
		t.Errorf("expected from 2026-09-01, got %v", repo.lastFilter.From)
	}
}

// 17. Filter by job vacancy
func TestFilterByJobVacancy(t *testing.T) {
	repo := newMemoryAnalyticsRepo()
	svc := service.NewAnalyticsService(repo, nil)

	_, err := svc.GetOverview(context.Background(), service.AnalyticsQuery{
		JobID: "job-123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.lastFilter.JobID != "job-123" {
		t.Errorf("expected JobID job-123, got %s", repo.lastFilter.JobID)
	}
}

// 18. Filter by department
func TestFilterByDepartment(t *testing.T) {
	repo := newMemoryAnalyticsRepo()
	svc := service.NewAnalyticsService(repo, nil)

	_, err := svc.GetOverview(context.Background(), service.AnalyticsQuery{
		Department: "Engineering",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.lastFilter.Department != "Engineering" {
		t.Errorf("expected Department Engineering, got %s", repo.lastFilter.Department)
	}
}

// 19. Filter by job status
func TestFilterByJobStatus(t *testing.T) {
	repo := newMemoryAnalyticsRepo()
	svc := service.NewAnalyticsService(repo, nil)

	_, err := svc.GetOverview(context.Background(), service.AnalyticsQuery{
		JobStatus: "OPEN",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.lastFilter.JobStatus != "OPEN" {
		t.Errorf("expected JobStatus OPEN, got %s", repo.lastFilter.JobStatus)
	}
}

// 20. Filter by recruiter
func TestFilterByRecruiter(t *testing.T) {
	repo := newMemoryAnalyticsRepo()
	svc := service.NewAnalyticsService(repo, nil)

	_, err := svc.GetOverview(context.Background(), service.AnalyticsQuery{
		RecruiterID: "rec-sarah",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.lastFilter.RecruiterID != "rec-sarah" {
		t.Errorf("expected RecruiterID rec-sarah, got %s", repo.lastFilter.RecruiterID)
	}
}

// 21. Combined multi-filter
func TestCombinedMultiFilter(t *testing.T) {
	repo := newMemoryAnalyticsRepo()
	svc := service.NewAnalyticsService(repo, nil)

	_, err := svc.GetOverview(context.Background(), service.AnalyticsQuery{
		From:       "2026-09-01",
		To:         "2026-09-15",
		Department: "Engineering",
		JobStatus:  "OPEN",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.lastFilter.Department != "Engineering" || repo.lastFilter.JobStatus != "OPEN" || repo.lastFilter.From == nil {
		t.Errorf("expected all filters preserved in repository call: %+v", repo.lastFilter)
	}
}

// 22. Empty result handling (zero counts, no division by zero)
func TestEmptyResultHandling(t *testing.T) {
	repo := newMemoryAnalyticsRepo()
	repo.summary = &repository.SummaryCounts{}
	repo.funnel = &repository.FunnelCounts{}
	repo.pipeline = []repository.StatusCountItem{}
	repo.screening = []repository.StatusCountItem{}
	repo.itwStatus = []repository.StatusCountItem{}
	repo.jobs = []repository.JobPerformanceRow{
		{
			JobID:           "job-empty",
			JobTitle:        "Empty Position",
			TotalCandidates: 0,
			Reviewed:        0,
			Shortlisted:     0,
			Interviews:      0,
		},
	}
	svc := service.NewAnalyticsService(repo, nil)

	resp, err := svc.GetOverview(context.Background(), service.AnalyticsQuery{})
	if err != nil {
		t.Fatalf("unexpected error on empty metrics: %v", err)
	}

	if resp.Funnel.ReviewRate != 0.0 || resp.Funnel.ShortlistRate != 0.0 || resp.Funnel.CompletionRate != 0.0 {
		t.Errorf("expected 0.0 conversion rates, got %+v", resp.Funnel)
	}

	perf, err := svc.GetJobPerformance(context.Background(), service.AnalyticsQuery{})
	if err != nil {
		t.Fatalf("unexpected error on empty job performance: %v", err)
	}
	if perf.Jobs[0].ReviewRate != 0.0 || perf.Jobs[0].CompletionRate != 0.0 {
		t.Errorf("expected 0.0 job conversion rates, got %+v", perf.Jobs[0])
	}
}

// 23. Date range validation (from > to returns 400 Bad Request)
func TestDateRangeValidation(t *testing.T) {
	repo := newMemoryAnalyticsRepo()
	svc := service.NewAnalyticsService(repo, nil)
	router := setupTestAnalyticsRouter(svc)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/analytics/overview?from=2026-09-20&to=2026-09-01", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected HTTP 400 Bad Request, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "INVALID_QUERY_PARAMS") {
		t.Errorf("expected INVALID_QUERY_PARAMS in error response, got %s", w.Body.String())
	}
}

// 24. Max date span enforcement (> 366 days returns 400 Bad Request)
func TestMaxDateSpanEnforcement(t *testing.T) {
	repo := newMemoryAnalyticsRepo()
	svc := service.NewAnalyticsService(repo, nil)
	router := setupTestAnalyticsRouter(svc)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/analytics/overview?from=2024-01-01&to=2026-01-01", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected HTTP 400 Bad Request for > 366 days, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "INVALID_QUERY_PARAMS") {
		t.Errorf("expected INVALID_QUERY_PARAMS in error response, got %s", w.Body.String())
	}
}

// 25. CSV export format
func TestCSVExportFormat(t *testing.T) {
	repo := newMemoryAnalyticsRepo()
	audit := newMemoryAuditRepo()
	svc := service.NewAnalyticsService(repo, audit)
	router := setupTestAnalyticsRouter(svc)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/analytics/export?from=2026-09-01&to=2026-09-20", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/csv") {
		t.Errorf("expected text/csv Content-Type, got %s", contentType)
	}

	contentDisposition := w.Header().Get("Content-Disposition")
	if !strings.Contains(contentDisposition, "attachment; filename=hirescope-analytics-") {
		t.Errorf("expected attachment filename header, got %s", contentDisposition)
	}

	reader := csv.NewReader(strings.NewReader(w.Body.String()))
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("failed to parse generated CSV: %v", err)
	}

	if len(records) < 10 {
		t.Errorf("expected at least 10 lines in CSV report, got %d", len(records))
	}

	// Verify audit log creation
	if len(audit.logs) != 1 {
		t.Fatalf("expected 1 audit log created, got %d", len(audit.logs))
	}
	if audit.logs[0].Action != model.AuditActionAnalyticsExported {
		t.Errorf("expected action ANALYTICS_EXPORTED, got %s", audit.logs[0].Action)
	}
}

// 26. CSV formula injection protection
func TestCSVFormulaInjectionProtection(t *testing.T) {
	// Direct unit test of SanitizeCSVValue
	testCases := []struct {
		input    string
		expected string
	}{
		{"=cmd|' /C calc'!A0", "'=cmd|' /C calc'!A0"},
		{"+1+1", "'+1+1"},
		{"-SUM(A1:A10)", "'-SUM(A1:A10)"},
		{"@SUM(A1:A10)", "'@SUM(A1:A10)"},
		{"Engineering", "Engineering"},
		{"Normal Title", "Normal Title"},
		{"  =evil", "'=evil"},
	}

	for _, tc := range testCases {
		actual := service.SanitizeCSVValue(tc.input)
		if actual != tc.expected {
			t.Errorf("SanitizeCSVValue(%q) = %q, expected %q", tc.input, actual, tc.expected)
		}
	}

	// End-to-end CSV export verification with malicious values
	repo := newMemoryAnalyticsRepo()
	repo.jobs = []repository.JobPerformanceRow{
		{
			JobID:               "job-malicious",
			JobCode:             "=1+1",
			JobTitle:            "+Malicious Title",
			Department:          "@Department",
			Status:              model.JobStatusOpen,
			TotalCandidates:     10,
			Reviewed:            5,
			Shortlisted:         2,
			Interviews:          1,
			CompletedInterviews: 1,
		},
	}
	repo.recruiters = []repository.RecruiterActivityRow{
		{
			RecruiterID:    "rec-malicious",
			RecruiterName:  "-Injected Recruiter",
			RecruiterEmail: "=evil@attacker.com",
			Role:           "RECRUITER",
		},
	}

	svc := service.NewAnalyticsService(repo, nil)
	csvBytes, _, err := svc.ExportCSV(context.Background(), service.AnalyticsQuery{}, "user-1")
	if err != nil {
		t.Fatalf("unexpected export error: %v", err)
	}

	content := string(csvBytes)
	if strings.Contains(content, "\n=1+1") || strings.Contains(content, ",=1+1") {
		t.Errorf("unquoted formula injection found for '=1+1'")
	}
	if strings.Contains(content, "\n+Malicious") || strings.Contains(content, ",+Malicious") {
		t.Errorf("unquoted formula injection found for '+Malicious'")
	}
	if strings.Contains(content, "\n@Department") || strings.Contains(content, ",@Department") {
		t.Errorf("unquoted formula injection found for '@Department'")
	}
	if strings.Contains(content, "\n-Injected") || strings.Contains(content, ",-Injected") {
		t.Errorf("unquoted formula injection found for '-Injected'")
	}

	// Must contain the sanitized single quotes
	if !strings.Contains(content, "'=1+1") {
		t.Errorf("expected sanitized ''=1+1' in output")
	}
	if !strings.Contains(content, "'+Malicious Title") {
		t.Errorf("expected sanitized ''+Malicious Title' in output")
	}
	if !strings.Contains(content, "'@Department") {
		t.Errorf("expected sanitized ''@Department' in output")
	}
	if !strings.Contains(content, "'-Injected Recruiter") {
		t.Errorf("expected sanitized ''-Injected Recruiter' in output")
	}
}
