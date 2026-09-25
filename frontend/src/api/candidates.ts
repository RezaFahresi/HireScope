import { apiClient } from './client'
import type {
  Candidate,
  CandidateListParams,
  CreateCandidateRequest,
  ApiSuccess,
  PaginatedResponse,
} from '@/types'

export const candidatesApi = {
  list: async (params?: CandidateListParams): Promise<PaginatedResponse<Candidate>['data']> => {
    const res = await apiClient.get<PaginatedResponse<Candidate>>('/candidates', { params })
    return res.data.data
  },

  get: async (id: string): Promise<Candidate> => {
    const res = await apiClient.get<ApiSuccess<Candidate>>(`/candidates/${id}`)
    return res.data.data
  },

  create: async (data: CreateCandidateRequest): Promise<Candidate> => {
    const res = await apiClient.post<ApiSuccess<Candidate>>('/candidates', data)
    return res.data.data
  },
}
