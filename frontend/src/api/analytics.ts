import { apiClient } from './client'
import type { ApiSuccess } from '@/types'

export interface AnalyticsFilterParams {
  from?: string
  to?: string
  job_id?: string
  department?: string
  job_status?: string
  recruiter_id?: string
  interval?: 'day' | 'week' | 'month'
}

export interface AnalyticsSummaryCounts {
  total_candidates: number
  active_jobs: number
  total_interviews: number
  scheduled_interviews: number
  completed_interviews: number
  cancelled_interviews: number
  shortlisted_applications: number
}

export interface FunnelStage {
  stage: string
  count: number
  conversion_rate: number
  overall_rate: number
}

export interface FunnelData {
  stages: FunnelStage[]
  total_candidates: number
  reviewed_candidates: number
  shortlisted_applications: number
  interviewed_candidates: number
  completed_interview_candidates: number
  review_rate: number
  shortlist_rate: number
  interview_rate: number
  completion_rate: number
}

export interface DistributionItem {
  key: string
  count: number
  percentage: number
}

export interface RequirementMatchCount {
  importance: string
  match_status: string
  count: number
}

export interface ScreeningAnalytics {
  total: number
  by_status: DistributionItem[]
  requirement_matches: RequirementMatchCount[]
}

export interface InterviewAnalytics {
  total: number
  by_status: DistributionItem[]
  by_type: DistributionItem[]
  by_stage: DistributionItem[]
}

export interface TrendDataPoint {
  period: string
  date: string
  candidates_count: number
  interviews_count: number
}

export interface TimeMetrics {
  avg_assignment_to_review_hours?: number | null
  avg_review_to_interview_hours?: number | null
  avg_interview_to_completion_hours?: number | null
  sample_size: number
}

export interface AnalyticsOverviewResponse {
  summary: AnalyticsSummaryCounts
  funnel: FunnelData
  pipeline: DistributionItem[]
  screening: ScreeningAnalytics
  interviews: InterviewAnalytics
  trends: TrendDataPoint[]
  time_metrics?: TimeMetrics
  filter_meta: {
    from?: string
    to?: string
    job_id?: string
    department?: string
    job_status?: string
    recruiter_id?: string
    interval: string
  }
}

export interface JobPerformanceItem {
  job_id: string
  job_code: string
  job_title: string
  department: string
  status: string
  total_candidates: number
  reviewed: number
  review_rate: number
  shortlisted: number
  shortlist_rate: number
  interviews: number
  interview_rate: number
  completed_interviews: number
  completion_rate: number
}

export interface JobPerformanceResponse {
  jobs: JobPerformanceItem[]
  total: number
}

export interface RecruiterActivityRow {
  recruiter_id: string
  recruiter_name: string
  recruiter_email: string
  role: string
  candidates_reviewed: number
  notes_created: number
  interviews_scheduled: number
  interviews_completed: number
  last_activity_at?: string
}

export interface RecruiterActivityResponse {
  recruiters: RecruiterActivityRow[]
  total: number
}

export const analyticsApi = {
  getOverview: async (params: AnalyticsFilterParams = {}): Promise<AnalyticsOverviewResponse> => {
    const res = await apiClient.get<ApiSuccess<AnalyticsOverviewResponse>>('/analytics/overview', {
      params,
    })
    return res.data.data
  },

  getJobPerformance: async (params: AnalyticsFilterParams = {}): Promise<JobPerformanceResponse> => {
    const res = await apiClient.get<ApiSuccess<JobPerformanceResponse>>('/analytics/jobs', {
      params,
    })
    return res.data.data
  },

  getRecruiterActivity: async (params: AnalyticsFilterParams = {}): Promise<RecruiterActivityResponse> => {
    const res = await apiClient.get<ApiSuccess<RecruiterActivityResponse>>('/analytics/recruiters', {
      params,
    })
    return res.data.data
  },

  getDepartments: async (): Promise<string[]> => {
    const res = await apiClient.get<ApiSuccess<{ departments: string[] }>>('/analytics/departments')
    return res.data.data.departments || []
  },

  exportCSV: async (params: AnalyticsFilterParams = {}): Promise<void> => {
    const res = await apiClient.get('/analytics/export', {
      params,
      responseType: 'blob',
    })

    // Extract filename from header or fallback
    const disposition = res.headers['content-disposition'] as string | undefined
    let filename = `hirescope-analytics-${new Date().toISOString().split('T')[0]}.csv`
    if (disposition && disposition.includes('filename=')) {
      const parts = disposition.split('filename=')
      if (parts[1]) {
        filename = parts[1].replace(/"/g, '').trim()
      }
    }

    // Trigger download in browser
    const blob = new Blob([res.data], { type: 'text/csv;charset=utf-8;' })
    const url = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.setAttribute('download', filename)
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    window.URL.revokeObjectURL(url)
  },
}
