import axios, { AxiosError, AxiosRequestConfig } from 'axios'
import type { ApiError } from '@/types'

const BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1'

export const apiClient = axios.create({
  baseURL: BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
  timeout: 15000,
})

// Request interceptor — inject token
apiClient.interceptors.request.use((config) => {
  const token = localStorage.getItem('access_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// Response interceptor — handle 401 globally
apiClient.interceptors.response.use(
  (response) => response,
  (error: AxiosError<ApiError>) => {
    if (error.response?.status === 401) {
      // Do not redirect when the user is already on the login page or failing a login attempt
      const isLoginRequest = error.config?.url?.includes('/auth/login')
      const isAlreadyOnLoginPage = typeof window !== 'undefined' && window.location.pathname === '/login'
      if (!isLoginRequest && !isAlreadyOnLoginPage) {
        localStorage.removeItem('access_token')
        localStorage.removeItem('user')
        window.location.href = '/login'
      }
    }
    return Promise.reject(error)
  }
)

export function getApiError(error: unknown): string {
  if (axios.isAxiosError(error)) {
    const data = error.response?.data as ApiError | undefined
    if (data?.error?.message) return data.error.message
    if (error.message) return error.message
  }
  return 'An unexpected error occurred'
}

export type { AxiosRequestConfig }
