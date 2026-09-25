import { apiClient } from './client'
import type {
  DashboardSummary,
  DashboardRecentCandidate,
  DashboardOpenJob,
  DashboardActivity,
  ApiSuccess,
} from '@/types'

export const dashboardApi = {
  // Aggregate KPI summary, job status counts, pipeline distribution, and needs attention metrics
  getSummary: async (): Promise<DashboardSummary> => {
    const res = await apiClient.get<ApiSuccess<DashboardSummary>>('/dashboard/summary')
    return res.data.data
  },

  // Recent candidate applications or assignments
  getRecentCandidates: async (limit = 5): Promise<DashboardRecentCandidate[]> => {
    const res = await apiClient.get<ApiSuccess<DashboardRecentCandidate[]>>(
      '/dashboard/recent-candidates',
      { params: { limit } }
    )
    return res.data.data
  },

  // Active / open jobs with real candidate count per job
  getOpenJobs: async (limit = 5): Promise<DashboardOpenJob[]> => {
    const res = await apiClient.get<ApiSuccess<DashboardOpenJob[]>>(
      '/dashboard/open-jobs',
      { params: { limit } }
    )
    return res.data.data
  },

  // Recruiter workflow audit activity timeline
  getActivity: async (limit = 10): Promise<DashboardActivity[]> => {
    const res = await apiClient.get<ApiSuccess<DashboardActivity[]>>(
      '/dashboard/activity',
      { params: { limit } }
    )
    return res.data.data
  },
}
