import { apiClient } from './client'
import type {
  Job,
  JobRequirement,
  CreateJobRequest,
  UpdateJobRequest,
  UpdateJobStatusRequest,
  CreateRequirementRequest,
  JobListParams,
  ApiSuccess,
  PaginatedResponse,
} from '@/types'

export const jobsApi = {
  list: async (params?: JobListParams): Promise<PaginatedResponse<Job>['data']> => {
    const res = await apiClient.get<PaginatedResponse<Job>>('/jobs', { params })
    return res.data.data
  },

  get: async (id: string): Promise<Job> => {
    const res = await apiClient.get<ApiSuccess<Job>>(`/jobs/${id}`)
    return res.data.data
  },

  create: async (data: CreateJobRequest): Promise<Job> => {
    const res = await apiClient.post<ApiSuccess<Job>>('/jobs', data)
    return res.data.data
  },

  update: async (id: string, data: UpdateJobRequest): Promise<Job> => {
    const res = await apiClient.put<ApiSuccess<Job>>(`/jobs/${id}`, data)
    return res.data.data
  },

  updateStatus: async (id: string, data: UpdateJobStatusRequest): Promise<Job> => {
    const res = await apiClient.patch<ApiSuccess<Job>>(`/jobs/${id}/status`, data)
    return res.data.data
  },

  // Requirements
  addRequirement: async (jobId: string, data: CreateRequirementRequest): Promise<JobRequirement> => {
    const res = await apiClient.post<ApiSuccess<JobRequirement>>(`/jobs/${jobId}/requirements`, data)
    return res.data.data
  },

  updateRequirement: async (jobId: string, reqId: string, data: Partial<CreateRequirementRequest>): Promise<JobRequirement> => {
    const res = await apiClient.put<ApiSuccess<JobRequirement>>(`/jobs/${jobId}/requirements/${reqId}`, data)
    return res.data.data
  },

  deleteRequirement: async (jobId: string, reqId: string): Promise<void> => {
    await apiClient.delete(`/jobs/${jobId}/requirements/${reqId}`)
  },
}
