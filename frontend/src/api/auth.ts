import { apiClient } from './client'
import type { LoginRequest, LoginResponse, User, ApiSuccess } from '@/types'

export const authApi = {
  login: async (data: LoginRequest): Promise<LoginResponse> => {
    const res = await apiClient.post<ApiSuccess<LoginResponse>>('/auth/login', data)
    return res.data.data
  },

  logout: async (): Promise<void> => {
    await apiClient.post('/auth/logout')
  },

  me: async (): Promise<User> => {
    const res = await apiClient.get<ApiSuccess<User>>('/auth/me')
    return res.data.data
  },
}
