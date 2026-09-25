import { apiClient } from './client'
import type {
  ApiSuccess,
  CalendarConnectionDTO,
  CalendarProviderType,
  InterviewCalendarEventDTO,
} from '@/types'

export const calendarApi = {
  // Get active connections for authenticated user
  getConnections: async (): Promise<CalendarConnectionDTO[]> => {
    const res = await apiClient.get<ApiSuccess<CalendarConnectionDTO[]>>('/calendar/connections')
    return res.data.data
  },

  // Initiate Google OAuth connection
  connectGoogle: async (): Promise<{ auth_url: string }> => {
    const res = await apiClient.get<ApiSuccess<{ auth_url: string }>>('/calendar/google/connect')
    return res.data.data
  },

  // Disconnect Google Calendar
  disconnectGoogle: async (): Promise<void> => {
    await apiClient.delete<ApiSuccess<{ message: string }>>('/calendar/google')
  },

  // Initiate Microsoft OAuth connection
  connectMicrosoft: async (): Promise<{ auth_url: string }> => {
    const res = await apiClient.get<ApiSuccess<{ auth_url: string }>>('/calendar/microsoft/connect')
    return res.data.data
  },

  // Disconnect Microsoft Calendar
  disconnectMicrosoft: async (): Promise<void> => {
    await apiClient.delete<ApiSuccess<{ message: string }>>('/calendar/microsoft')
  },

  // Get external calendar events synced for an interview
  getInterviewEvents: async (interviewId: string): Promise<InterviewCalendarEventDTO[]> => {
    const res = await apiClient.get<ApiSuccess<InterviewCalendarEventDTO[]>>(
      `/interviews/${interviewId}/calendar/events`
    )
    return res.data.data
  },

  // Manually sync an interview to an external calendar provider
  syncInterview: async (
    interviewId: string,
    provider: CalendarProviderType
  ): Promise<InterviewCalendarEventDTO> => {
    const provSlug = provider === 'GOOGLE' ? 'google' : 'microsoft'
    const res = await apiClient.post<ApiSuccess<InterviewCalendarEventDTO>>(
      `/interviews/${interviewId}/calendar/sync/${provSlug}`
    )
    return res.data.data
  },

  // Retry failed sync for an interview
  retrySync: async (
    interviewId: string,
    provider: CalendarProviderType
  ): Promise<InterviewCalendarEventDTO> => {
    const provSlug = provider === 'GOOGLE' ? 'google' : 'microsoft'
    const res = await apiClient.post<ApiSuccess<InterviewCalendarEventDTO>>(
      `/interviews/${interviewId}/calendar/retry/${provSlug}`
    )
    return res.data.data
  },
}
