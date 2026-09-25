import { apiClient } from './client'
import type {
  JobCandidate,
  CandidateNote,
  ApiSuccess,
  PaginatedResponse,
  WorkflowStatus,
  ReviewDetailResponse
} from '@/types'

export interface JobCandidatesFilter {
  page?: number
  limit?: number
  status?: WorkflowStatus | ''
  search?: string
  sort?: string
  order?: 'asc' | 'desc'
}

export const reviewsApi = {
  // List candidates for a specific job
  listCandidatesByJob: async (jobId: string, params?: JobCandidatesFilter): Promise<PaginatedResponse<JobCandidate>['data']> => {
    const res = await apiClient.get<PaginatedResponse<JobCandidate>>(`/jobs/${jobId}/candidates`, { params })
    return res.data.data
  },

  // Get full review details for a specific candidate on a job
  getReviewDetail: async (jobId: string, candidateId: string): Promise<ReviewDetailResponse> => {
    const res = await apiClient.get<ApiSuccess<ReviewDetailResponse>>(`/jobs/${jobId}/candidates/${candidateId}/review`)
    return res.data.data
  },

  // Update candidate workflow status
  updateStatus: async (jobId: string, candidateId: string, status: WorkflowStatus): Promise<JobCandidate> => {
    const res = await apiClient.patch<ApiSuccess<JobCandidate>>(`/jobs/${jobId}/candidates/${candidateId}/status`, { status })
    return res.data.data
  },

  // List recruiter notes
  listNotes: async (jobId: string, candidateId: string, params?: { page?: number; limit?: number }): Promise<PaginatedResponse<CandidateNote>['data']> => {
    const res = await apiClient.get<PaginatedResponse<CandidateNote>>(`/jobs/${jobId}/candidates/${candidateId}/notes`, { params })
    return res.data.data
  },

  // Create a note
  createNote: async (jobId: string, candidateId: string, content: string): Promise<CandidateNote> => {
    const res = await apiClient.post<ApiSuccess<CandidateNote>>(`/jobs/${jobId}/candidates/${candidateId}/notes`, { content })
    return res.data.data
  },

  // Update a note
  updateNote: async (jobId: string, candidateId: string, noteId: string, content: string): Promise<CandidateNote> => {
    const res = await apiClient.put<ApiSuccess<CandidateNote>>(`/jobs/${jobId}/candidates/${candidateId}/notes/${noteId}`, { content })
    return res.data.data
  },

  // Delete a note
  deleteNote: async (jobId: string, candidateId: string, noteId: string): Promise<void> => {
    await apiClient.delete(`/jobs/${jobId}/candidates/${candidateId}/notes/${noteId}`)
  },
}
