package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"hirescope/backend/internal/model"
	"hirescope/backend/internal/repository"
)

// Standard errors for analytics validation.
var (
	ErrDateRangeTooLong = ErrDateRangeTooLarge
	ErrInvalidInterval  = errors.New("invalid trend interval, allowed values: day, week, month")
)

// AnalyticsQuery contains parameters supplied via query strings.
type AnalyticsQuery struct {
	From        string `json:"from"`
	To          string `json:"to"`
	JobID       string `json:"job_id"`
	Department  string `json:"department"`
	JobStatus   string `json:"job_status"`
	RecruiterID string `json:"recruiter_id"`
	Interval    string `json:"interval"` // "day", "week", "month"
}

// FunnelStage represents one stage in the recruitment funnel with volume and conversion rate.
type FunnelStage struct {
	Stage          string  `json:"stage"`
	Count          int64   `json:"count"`
	ConversionRate float64 `json:"conversion_rate"` // percentage relative to preceding stage (or base)
	OverallRate    float64 `json:"overall_rate"`    // percentage relative to total candidates
}

// FunnelData encapsulates the recruitment funnel metrics.
type FunnelData struct {
	Stages                       []FunnelStage `json:"stages"`
	TotalCandidates              int64         `json:"total_candidates"`
	ReviewedCandidates           int64         `json:"reviewed_candidates"`
	ShortlistedApplications      int64         `json:"shortlisted_applications"`
	InterviewedCandidates        int64         `json:"interviewed_candidates"`
	CompletedInterviewCandidates int64         `json:"completed_interview_candidates"`
	ReviewRate                   float64       `json:"review_rate"`
	ShortlistRate                float64       `json:"shortlist_rate"`
	InterviewRate                float64       `json:"interview_rate"`
	CompletionRate               float64       `json:"completion_rate"`
}

// DistributionItem represents a categorical key-value metric with percentage.
type DistributionItem struct {
	Key        string  `json:"key"`
	Count      int64   `json:"count"`
	Percentage float64 `json:"percentage"`
}

// InterviewAnalytics encapsulates status, modality, and stage breakdowns.
type InterviewAnalytics struct {
	Total     int64              `json:"total"`
	ByStatus  []DistributionItem `json:"by_status"`
	ByType    []DistributionItem `json:"by_type"`
	ByStage   []DistributionItem `json:"by_stage"`
}

// ScreeningAnalytics encapsulates screening status and requirement match distributions.
type ScreeningAnalytics struct {
	Total              int64                              `json:"total"`
	ByStatus           []DistributionItem                 `json:"by_status"`
	RequirementMatches []repository.RequirementMatchCount `json:"requirement_matches"`
}

// AnalyticsOverviewResponse aggregates all operational metrics for the /analytics view.
type AnalyticsOverviewResponse struct {
	Summary     repository.SummaryCounts     `json:"summary"`
	Funnel      FunnelData                   `json:"funnel"`
	Pipeline    []DistributionItem           `json:"pipeline"`
	Screening   ScreeningAnalytics           `json:"screening"`
	Interviews  InterviewAnalytics           `json:"interviews"`
	Trends      []repository.TrendDataPoint  `json:"trends"`
	TimeMetrics *repository.TimeMetrics      `json:"time_metrics,omitempty"`
	FilterMeta  AnalyticsFilterMetadata      `json:"filter_meta"`
}

// AnalyticsFilterMetadata describes the active parsed filter boundaries.
type AnalyticsFilterMetadata struct {
	From        *time.Time `json:"from,omitempty"`
	To          *time.Time `json:"to,omitempty"`
	JobID       string     `json:"job_id,omitempty"`
	Department  string     `json:"department,omitempty"`
	JobStatus   string     `json:"job_status,omitempty"`
	RecruiterID string     `json:"recruiter_id,omitempty"`
	Interval    string     `json:"interval"`
}

// JobPerformanceItem augments JobPerformanceRow with conversion rates.
type JobPerformanceItem struct {
	repository.JobPerformanceRow
	ReviewRate      float64 `json:"review_rate"`
	ShortlistRate   float64 `json:"shortlist_rate"`
	InterviewRate   float64 `json:"interview_rate"`
	CompletionRate  float64 `json:"completion_rate"`
}

// JobPerformanceResponse contains the list of job performance records.
type JobPerformanceResponse struct {
	Jobs  []JobPerformanceItem `json:"jobs"`
	Total int                  `json:"total"`
}

// RecruiterActivityResponse contains the list of recruiter workload records.
type RecruiterActivityResponse struct {
	Recruiters []repository.RecruiterActivityRow `json:"recruiters"`
	Total      int                               `json:"total"`
}

// AnalyticsService defines business logic for deterministic recruitment analytics.
type AnalyticsService interface {
	GetOverview(ctx context.Context, query AnalyticsQuery) (*AnalyticsOverviewResponse, error)
	GetJobPerformance(ctx context.Context, query AnalyticsQuery) (*JobPerformanceResponse, error)
	GetRecruiterActivity(ctx context.Context, query AnalyticsQuery) (*RecruiterActivityResponse, error)
	GetDepartments(ctx context.Context) ([]string, error)
	ExportCSV(ctx context.Context, query AnalyticsQuery, actorID string) ([]byte, string, error)
}

type analyticsService struct {
	analyticsRepo repository.AnalyticsRepository
	auditRepo     repository.AuditLogRepository
}

// NewAnalyticsService constructs a new AnalyticsService.
func NewAnalyticsService(
	analyticsRepo repository.AnalyticsRepository,
	auditRepo repository.AuditLogRepository,
) AnalyticsService {
	return &analyticsService{
		analyticsRepo: analyticsRepo,
		auditRepo:     auditRepo,
	}
}

// parseAndValidateFilter parses string date filters and enforces business rules.
func parseAndValidateFilter(query AnalyticsQuery) (repository.AnalyticsFilter, string, error) {
	filter := repository.AnalyticsFilter{
		JobID:       strings.TrimSpace(query.JobID),
		Department:  strings.TrimSpace(query.Department),
		JobStatus:   strings.TrimSpace(query.JobStatus),
		RecruiterID: strings.TrimSpace(query.RecruiterID),
	}

	interval := strings.ToLower(strings.TrimSpace(query.Interval))
	if interval == "" {
		interval = "day"
	}
	if interval != "day" && interval != "week" && interval != "month" {
		return filter, interval, ErrInvalidInterval
	}

	var fromTime, toTime *time.Time

	if query.From != "" {
		t, err := parseAnalyticsDate(query.From, false)
		if err != nil {
			return filter, interval, fmt.Errorf("invalid from date format: %w", err)
		}
		fromTime = &t
	}

	if query.To != "" {
		t, err := parseAnalyticsDate(query.To, true)
		if err != nil {
			return filter, interval, fmt.Errorf("invalid to date format: %w", err)
		}
		toTime = &t
	}

	// Validate date relationship if both provided
	if fromTime != nil && toTime != nil {
		if fromTime.After(*toTime) {
			return filter, interval, ErrInvalidDateRange
		}
		// Max date span is 366 days
		diff := toTime.Sub(*fromTime)
		if diff > 366*24*time.Hour {
			return filter, interval, ErrDateRangeTooLong
		}
	}

	filter.From = fromTime
	filter.To = toTime

	return filter, interval, nil
}

// parseAnalyticsDate accepts either YYYY-MM-DD or RFC3339 formats.
func parseAnalyticsDate(val string, endOfDay bool) (time.Time, error) {
	val = strings.TrimSpace(val)
	// Try RFC3339 first
	if t, err := time.Parse(time.RFC3339, val); err == nil {
		return t.UTC(), nil
	}

	// Try YYYY-MM-DD
	if t, err := time.Parse("2006-01-02", val); err == nil {
		if endOfDay {
			return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, time.UTC), nil
		}
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), nil
	}

	return time.Time{}, errors.New("date must be in YYYY-MM-DD or RFC3339 format")
}

// safePercentage calculates a safe percentage rounded to 1 decimal place.
func safePercentage(numerator, denominator int64) float64 {
	if denominator <= 0 {
		return 0.0
	}
	pct := (float64(numerator) / float64(denominator)) * 100.0
	return math.Round(pct*10) / 10
}

func calculateDistribution(items []repository.StatusCountItem) ([]DistributionItem, int64) {
	var total int64
	for _, it := range items {
		total += it.Count
	}

	result := make([]DistributionItem, len(items))
	for i, it := range items {
		result[i] = DistributionItem{
			Key:        it.Key,
			Count:      it.Count,
			Percentage: safePercentage(it.Count, total),
		}
	}
	return result, total
}

func (s *analyticsService) GetOverview(ctx context.Context, query AnalyticsQuery) (*AnalyticsOverviewResponse, error) {
	filter, interval, err := parseAndValidateFilter(query)
	if err != nil {
		return nil, err
	}

	summary, err := s.analyticsRepo.GetSummaryCounts(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get summary counts: %w", err)
	}

	funnelCounts, err := s.analyticsRepo.GetFunnelCounts(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get funnel counts: %w", err)
	}

	pipelineItems, err := s.analyticsRepo.GetPipelineDistribution(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get pipeline distribution: %w", err)
	}
	pipelineDist, _ := calculateDistribution(pipelineItems)

	screeningItems, err := s.analyticsRepo.GetScreeningDistribution(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get screening distribution: %w", err)
	}
	screeningDist, screeningTotal := calculateDistribution(screeningItems)

	screeningMatches, err := s.analyticsRepo.GetScreeningRequirementMatches(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get screening matches: %w", err)
	}

	interviewStatuses, err := s.analyticsRepo.GetInterviewStatusDistribution(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get interview status distribution: %w", err)
	}
	statusDist, interviewTotal := calculateDistribution(interviewStatuses)

	interviewTypes, err := s.analyticsRepo.GetInterviewTypeDistribution(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get interview type distribution: %w", err)
	}
	typeDist, _ := calculateDistribution(interviewTypes)

	interviewStages, err := s.analyticsRepo.GetInterviewStageDistribution(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get interview stage distribution: %w", err)
	}
	stageDist, _ := calculateDistribution(interviewStages)

	trends, err := s.analyticsRepo.GetRecruitmentTrends(ctx, filter, interval)
	if err != nil {
		return nil, fmt.Errorf("failed to get recruitment trends: %w", err)
	}

	timeMetrics, err := s.analyticsRepo.GetTimeMetrics(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get time metrics: %w", err)
	}

	// Calculate funnel rates safely
	totalCand := funnelCounts.TotalCandidates
	reviewRate := safePercentage(funnelCounts.ReviewedCandidates, totalCand)
	shortlistRate := safePercentage(funnelCounts.ShortlistedApplications, funnelCounts.ReviewedCandidates)
	interviewRate := safePercentage(funnelCounts.InterviewedCandidates, funnelCounts.ShortlistedApplications)
	completionRate := safePercentage(funnelCounts.CompletedInterviewCandidates, funnelCounts.InterviewedCandidates)

	funnelStages := []FunnelStage{
		{
			Stage:          "Total Candidates",
			Count:          funnelCounts.TotalCandidates,
			ConversionRate: 100.0,
			OverallRate:    100.0,
		},
		{
			Stage:          "Reviewed",
			Count:          funnelCounts.ReviewedCandidates,
			ConversionRate: safePercentage(funnelCounts.ReviewedCandidates, funnelCounts.TotalCandidates),
			OverallRate:    safePercentage(funnelCounts.ReviewedCandidates, funnelCounts.TotalCandidates),
		},
		{
			Stage:          "Shortlisted",
			Count:          funnelCounts.ShortlistedApplications,
			ConversionRate: safePercentage(funnelCounts.ShortlistedApplications, funnelCounts.ReviewedCandidates),
			OverallRate:    safePercentage(funnelCounts.ShortlistedApplications, funnelCounts.TotalCandidates),
		},
		{
			Stage:          "Interviewed",
			Count:          funnelCounts.InterviewedCandidates,
			ConversionRate: safePercentage(funnelCounts.InterviewedCandidates, funnelCounts.ShortlistedApplications),
			OverallRate:    safePercentage(funnelCounts.InterviewedCandidates, funnelCounts.TotalCandidates),
		},
		{
			Stage:          "Completed",
			Count:          funnelCounts.CompletedInterviewCandidates,
			ConversionRate: safePercentage(funnelCounts.CompletedInterviewCandidates, funnelCounts.InterviewedCandidates),
			OverallRate:    safePercentage(funnelCounts.CompletedInterviewCandidates, funnelCounts.TotalCandidates),
		},
	}

	funnel := FunnelData{
		Stages:                       funnelStages,
		TotalCandidates:              funnelCounts.TotalCandidates,
		ReviewedCandidates:           funnelCounts.ReviewedCandidates,
		ShortlistedApplications:      funnelCounts.ShortlistedApplications,
		InterviewedCandidates:        funnelCounts.InterviewedCandidates,
		CompletedInterviewCandidates: funnelCounts.CompletedInterviewCandidates,
		ReviewRate:                   reviewRate,
		ShortlistRate:                shortlistRate,
		InterviewRate:                interviewRate,
		CompletionRate:               completionRate,
	}

	resp := &AnalyticsOverviewResponse{
		Summary:  *summary,
		Funnel:   funnel,
		Pipeline: pipelineDist,
		Screening: ScreeningAnalytics{
			Total:              screeningTotal,
			ByStatus:           screeningDist,
			RequirementMatches: screeningMatches,
		},
		Interviews: InterviewAnalytics{
			Total:    interviewTotal,
			ByStatus: statusDist,
			ByType:   typeDist,
			ByStage:  stageDist,
		},
		Trends:      trends,
		TimeMetrics: timeMetrics,
		FilterMeta: AnalyticsFilterMetadata{
			From:        filter.From,
			To:          filter.To,
			JobID:       filter.JobID,
			Department:  filter.Department,
			JobStatus:   filter.JobStatus,
			RecruiterID: filter.RecruiterID,
			Interval:    interval,
		},
	}

	return resp, nil
}

func (s *analyticsService) GetJobPerformance(ctx context.Context, query AnalyticsQuery) (*JobPerformanceResponse, error) {
	filter, _, err := parseAndValidateFilter(query)
	if err != nil {
		return nil, err
	}

	rows, err := s.analyticsRepo.GetJobPerformance(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get job performance: %w", err)
	}

	items := make([]JobPerformanceItem, len(rows))
	for i, r := range rows {
		items[i] = JobPerformanceItem{
			JobPerformanceRow: r,
			ReviewRate:        safePercentage(r.Reviewed, r.TotalCandidates),
			ShortlistRate:     safePercentage(r.Shortlisted, r.TotalCandidates),
			InterviewRate:     safePercentage(r.Interviews, r.TotalCandidates),
			CompletionRate:    safePercentage(r.CompletedInterviews, r.Interviews),
		}
	}

	return &JobPerformanceResponse{
		Jobs:  items,
		Total: len(items),
	}, nil
}

func (s *analyticsService) GetRecruiterActivity(ctx context.Context, query AnalyticsQuery) (*RecruiterActivityResponse, error) {
	filter, _, err := parseAndValidateFilter(query)
	if err != nil {
		return nil, err
	}

	rows, err := s.analyticsRepo.GetRecruiterActivity(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get recruiter activity: %w", err)
	}

	return &RecruiterActivityResponse{
		Recruiters: rows,
		Total:      len(rows),
	}, nil
}

func (s *analyticsService) GetDepartments(ctx context.Context) ([]string, error) {
	return s.analyticsRepo.GetDistinctDepartments(ctx)
}

// SanitizeCSVValue prevents CSV Formula Injection (CWE-1236).
// If a cell string starts with '=', '+', '-', or '@', it is prepended with a single quote.
func SanitizeCSVValue(val string) string {
	trimmed := strings.TrimSpace(val)
	if len(trimmed) > 0 {
		firstChar := trimmed[0]
		if firstChar == '=' || firstChar == '+' || firstChar == '-' || firstChar == '@' {
			return "'" + trimmed
		}
	}
	return val
}

func (s *analyticsService) ExportCSV(ctx context.Context, query AnalyticsQuery, actorID string) ([]byte, string, error) {
	filter, _, err := parseAndValidateFilter(query)
	if err != nil {
		return nil, "", err
	}

	summary, err := s.analyticsRepo.GetSummaryCounts(ctx, filter)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch summary counts for export: %w", err)
	}

	funnelCounts, err := s.analyticsRepo.GetFunnelCounts(ctx, filter)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch funnel counts for export: %w", err)
	}

	jobPerf, err := s.analyticsRepo.GetJobPerformance(ctx, filter)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch job performance for export: %w", err)
	}

	recruiterAct, err := s.analyticsRepo.GetRecruiterActivity(ctx, filter)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch recruiter activity for export: %w", err)
	}

	timeMetrics, _ := s.analyticsRepo.GetTimeMetrics(ctx, filter)

	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	now := time.Now().UTC()
	dateStr := now.Format("2006-01-02")
	filename := fmt.Sprintf("hirescope-analytics-%s.csv", dateStr)

	// Header section
	_ = writer.Write([]string{"HireScope Recruitment Analytics & Operations Report"})
	_ = writer.Write([]string{"Generated At (UTC)", SanitizeCSVValue(now.Format(time.RFC3339))})
	_ = writer.Write([]string{"Filter Job ID", SanitizeCSVValue(filter.JobID)})
	_ = writer.Write([]string{"Filter Department", SanitizeCSVValue(filter.Department)})
	_ = writer.Write([]string{"Filter Job Status", SanitizeCSVValue(filter.JobStatus)})
	_ = writer.Write([]string{"Filter Recruiter ID", SanitizeCSVValue(filter.RecruiterID)})
	if filter.From != nil {
		_ = writer.Write([]string{"Filter Date From", SanitizeCSVValue(filter.From.Format(time.RFC3339))})
	}
	if filter.To != nil {
		_ = writer.Write([]string{"Filter Date To", SanitizeCSVValue(filter.To.Format(time.RFC3339))})
	}
	_ = writer.Write([]string{""})

	// Section 1: Executive KPI Summary
	_ = writer.Write([]string{"=== EXECUTIVE SUMMARY METRICS ==="})
	_ = writer.Write([]string{"Metric", "Value"})
	_ = writer.Write([]string{"Total Candidates (Distinct)", strconv.FormatInt(summary.TotalCandidates, 10)})
	_ = writer.Write([]string{"Active Jobs (OPEN)", strconv.FormatInt(summary.ActiveJobs, 10)})
	_ = writer.Write([]string{"Total Interviews", strconv.FormatInt(summary.TotalInterviews, 10)})
	_ = writer.Write([]string{"Scheduled Interviews", strconv.FormatInt(summary.ScheduledInterviews, 10)})
	_ = writer.Write([]string{"Completed Interviews", strconv.FormatInt(summary.CompletedInterviews, 10)})
	_ = writer.Write([]string{"Cancelled Interviews", strconv.FormatInt(summary.CancelledInterviews, 10)})
	_ = writer.Write([]string{"Shortlisted Applications", strconv.FormatInt(summary.ShortlistedApplications, 10)})
	if timeMetrics != nil {
		if timeMetrics.AvgAssignmentToReviewHours != nil {
			_ = writer.Write([]string{"Avg Assignment to Review (Hours)", fmt.Sprintf("%.1f", *timeMetrics.AvgAssignmentToReviewHours)})
		}
		if timeMetrics.AvgReviewToInterviewHours != nil {
			_ = writer.Write([]string{"Avg Review to Interview (Hours)", fmt.Sprintf("%.1f", *timeMetrics.AvgReviewToInterviewHours)})
		}
		if timeMetrics.AvgInterviewToCompletionHours != nil {
			_ = writer.Write([]string{"Avg Interview to Completion (Hours)", fmt.Sprintf("%.1f", *timeMetrics.AvgInterviewToCompletionHours)})
		}
	}
	_ = writer.Write([]string{""})

	// Section 2: Recruitment Pipeline Funnel
	_ = writer.Write([]string{"=== RECRUITMENT PIPELINE FUNNEL ==="})
	_ = writer.Write([]string{"Funnel Stage", "Count", "Stage Conversion Rate (%)", "Overall Candidate Rate (%)"})
	_ = writer.Write([]string{
		"1. Registered Candidates",
		strconv.FormatInt(funnelCounts.TotalCandidates, 10),
		"100.0%",
		"100.0%",
	})
	_ = writer.Write([]string{
		"2. Reviewed Candidates",
		strconv.FormatInt(funnelCounts.ReviewedCandidates, 10),
		fmt.Sprintf("%.1f%%", safePercentage(funnelCounts.ReviewedCandidates, funnelCounts.TotalCandidates)),
		fmt.Sprintf("%.1f%%", safePercentage(funnelCounts.ReviewedCandidates, funnelCounts.TotalCandidates)),
	})
	_ = writer.Write([]string{
		"3. Shortlisted Applications",
		strconv.FormatInt(funnelCounts.ShortlistedApplications, 10),
		fmt.Sprintf("%.1f%%", safePercentage(funnelCounts.ShortlistedApplications, funnelCounts.ReviewedCandidates)),
		fmt.Sprintf("%.1f%%", safePercentage(funnelCounts.ShortlistedApplications, funnelCounts.TotalCandidates)),
	})
	_ = writer.Write([]string{
		"4. Interviewed Candidates",
		strconv.FormatInt(funnelCounts.InterviewedCandidates, 10),
		fmt.Sprintf("%.1f%%", safePercentage(funnelCounts.InterviewedCandidates, funnelCounts.ShortlistedApplications)),
		fmt.Sprintf("%.1f%%", safePercentage(funnelCounts.InterviewedCandidates, funnelCounts.TotalCandidates)),
	})
	_ = writer.Write([]string{
		"5. Completed Interview Candidates",
		strconv.FormatInt(funnelCounts.CompletedInterviewCandidates, 10),
		fmt.Sprintf("%.1f%%", safePercentage(funnelCounts.CompletedInterviewCandidates, funnelCounts.InterviewedCandidates)),
		fmt.Sprintf("%.1f%%", safePercentage(funnelCounts.CompletedInterviewCandidates, funnelCounts.TotalCandidates)),
	})
	_ = writer.Write([]string{""})

	// Section 3: Job Performance Table
	_ = writer.Write([]string{"=== JOB REQUISITION PERFORMANCE ==="})
	_ = writer.Write([]string{
		"Job Code",
		"Job Title",
		"Department",
		"Status",
		"Total Candidates",
		"Reviewed",
		"Review Rate (%)",
		"Shortlisted",
		"Shortlist Rate (%)",
		"Interviews",
		"Interview Rate (%)",
		"Completed Interviews",
	})
	for _, j := range jobPerf {
		_ = writer.Write([]string{
			SanitizeCSVValue(j.JobCode),
			SanitizeCSVValue(j.JobTitle),
			SanitizeCSVValue(j.Department),
			SanitizeCSVValue(string(j.Status)),
			strconv.FormatInt(j.TotalCandidates, 10),
			strconv.FormatInt(j.Reviewed, 10),
			fmt.Sprintf("%.1f%%", safePercentage(j.Reviewed, j.TotalCandidates)),
			strconv.FormatInt(j.Shortlisted, 10),
			fmt.Sprintf("%.1f%%", safePercentage(j.Shortlisted, j.TotalCandidates)),
			strconv.FormatInt(j.Interviews, 10),
			fmt.Sprintf("%.1f%%", safePercentage(j.Interviews, j.TotalCandidates)),
			strconv.FormatInt(j.CompletedInterviews, 10),
		})
	}
	_ = writer.Write([]string{""})

	// Section 4: Recruiter Operational Activity
	_ = writer.Write([]string{"=== RECRUITER OPERATIONAL ACTIVITY ==="})
	_ = writer.Write([]string{
		"Recruiter Name",
		"Email",
		"Role",
		"Candidates Reviewed",
		"Notes Created",
		"Interviews Scheduled",
		"Interviews Completed",
		"Last Activity (UTC)",
	})
	for _, r := range recruiterAct {
		lastAct := "N/A"
		if r.LastActivityAt != nil {
			lastAct = r.LastActivityAt.UTC().Format(time.RFC3339)
		}
		_ = writer.Write([]string{
			SanitizeCSVValue(r.RecruiterName),
			SanitizeCSVValue(r.RecruiterEmail),
			SanitizeCSVValue(r.Role),
			strconv.FormatInt(r.CandidatesReviewed, 10),
			strconv.FormatInt(r.NotesCreated, 10),
			strconv.FormatInt(r.InterviewsScheduled, 10),
			strconv.FormatInt(r.InterviewsCompleted, 10),
			SanitizeCSVValue(lastAct),
		})
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, "", fmt.Errorf("failed to generate CSV: %w", err)
	}

	// Record audit event
	if actorID != "" && s.auditRepo != nil {
		meta, _ := json.Marshal(map[string]interface{}{
			"filename": filename,
			"filter":   filter,
		})
		_ = s.auditRepo.Create(ctx, &model.AuditLog{
			Action:   model.AuditActionAnalyticsExported,
			ActorID:  actorID,
			Metadata: string(meta),
		})
	}

	return buf.Bytes(), filename, nil
}
