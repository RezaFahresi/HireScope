import { apiClient } from './client'
import type {
  Interview,
  InterviewListParams,
  CreateInterviewRequest,
  UpdateInterviewRequest,
  RescheduleInterviewRequest,
  CancelInterviewRequest,
  CompleteInterviewRequest,
  Interviewer,
  ApiSuccess,
  PaginatedData,
  InterviewEmailDelivery,
  SendInvitationRequest,
} from '@/types'

export const interviewsApi = {
  create: async (data: CreateInterviewRequest): Promise<Interview> => {
    const res = await apiClient.post<ApiSuccess<Interview>>('/interviews', data)
    return res.data.data
  },

  get: async (id: string): Promise<Interview> => {
    const res = await apiClient.get<ApiSuccess<Interview>>(`/interviews/${id}`)
    return res.data.data
  },

  list: async (params?: InterviewListParams): Promise<PaginatedData<Interview>> => {
    const res = await apiClient.get<ApiSuccess<PaginatedData<Interview>>>('/interviews', {
      params,
    })
    return res.data.data
  },

  update: async (id: string, data: UpdateInterviewRequest): Promise<Interview> => {
    const res = await apiClient.put<ApiSuccess<Interview>>(`/interviews/${id}`, data)
    return res.data.data
  },

  reschedule: async (id: string, data: RescheduleInterviewRequest): Promise<Interview> => {
    const res = await apiClient.patch<ApiSuccess<Interview>>(`/interviews/${id}/reschedule`, data)
    return res.data.data
  },

  cancel: async (id: string, data: CancelInterviewRequest): Promise<Interview> => {
    const res = await apiClient.patch<ApiSuccess<Interview>>(`/interviews/${id}/cancel`, data)
    return res.data.data
  },

  complete: async (id: string, data: CompleteInterviewRequest): Promise<Interview> => {
    const res = await apiClient.post<ApiSuccess<Interview>>(`/interviews/${id}/complete`, data)
    return res.data.data
  },

  getCandidateInterviews: async (candidateId: string, jobId?: string): Promise<Interview[]> => {
    const res = await apiClient.get<ApiSuccess<Interview[]>>(
      `/candidates/${candidateId}/interviews`,
      { params: { job_id: jobId } }
    )
    return res.data.data
  },

  getCandidateNextInterview: async (
    candidateId: string,
    jobId?: string
  ): Promise<Interview | null> => {
    const res = await apiClient.get<ApiSuccess<Interview | null>>(
      `/candidates/${candidateId}/next-interview`,
      { params: { job_id: jobId } }
    )
    return res.data.data
  },

  getUsers: async (): Promise<Interviewer[]> => {
    const res = await apiClient.get<ApiSuccess<Interviewer[]>>('/users')
    return res.data.data
  },

  downloadIcs: async (id: string, defaultFilename?: string): Promise<void> => {
    const res = await apiClient.get(`/interviews/${id}/ics`, {
      responseType: 'blob',
    })
    const blob = new Blob([res.data], { type: 'text/calendar;charset=utf-8' })
    const url = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url

    const disposition = res.headers['content-disposition'] as string | undefined
    let filename = defaultFilename || `interview-${id}.ics`
    if (disposition && disposition.includes('filename=')) {
      const match = disposition.match(/filename="?([^";]+)"?/)
      if (match && match[1]) {
        filename = match[1].trim()
      }
    }
    link.setAttribute('download', filename)
    document.body.appendChild(link)
    link.click()
    link.remove()
    window.URL.revokeObjectURL(url)
  },

  getEmailDeliveries: async (id: string): Promise<InterviewEmailDelivery[]> => {
    const res = await apiClient.get<ApiSuccess<InterviewEmailDelivery[]>>(
      `/interviews/${id}/email-deliveries`
    )
    return res.data.data
  },

  sendInvitations: async (
    id: string,
    data: SendInvitationRequest
  ): Promise<InterviewEmailDelivery[]> => {
    const res = await apiClient.post<ApiSuccess<InterviewEmailDelivery[]>>(
      `/interviews/${id}/send-invitation`,
      data
    )
    return res.data.data
  },

  retryEmailDelivery: async (
    interviewId: string,
    deliveryId: string
  ): Promise<InterviewEmailDelivery> => {
    const res = await apiClient.post<ApiSuccess<InterviewEmailDelivery>>(
      `/interviews/${interviewId}/email-deliveries/${deliveryId}/retry`
    )
    return res.data.data
  },
}
