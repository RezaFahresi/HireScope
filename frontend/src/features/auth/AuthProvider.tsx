import React, { createContext, useContext, useCallback } from 'react'
import type { User } from '@/types'
import { authApi } from '@/api/auth'

interface AuthContextValue {
  user: User | null
  token: string | null
  isAuthenticated: boolean
  isLoading: boolean
  login: (token: string, user: User) => void
  logout: () => Promise<void>
}

const AuthContext = createContext<AuthContextValue | undefined>(undefined)

// Initialize auth state synchronously from localStorage (no useEffect needed)
function readAuthStorage(): { token: string | null; user: User | null } {
  try {
    const token = localStorage.getItem('access_token')
    const raw = localStorage.getItem('user')
    if (token && raw) {
      return { token, user: JSON.parse(raw) as User }
    }
  } catch {
    localStorage.removeItem('access_token')
    localStorage.removeItem('user')
  }
  return { token: null, user: null }
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const initial = readAuthStorage()

  const [user, setUser] = React.useState<User | null>(initial.user)
  const [token, setToken] = React.useState<string | null>(initial.token)

  const login = useCallback((newToken: string, newUser: User) => {
    localStorage.setItem('access_token', newToken)
    localStorage.setItem('user', JSON.stringify(newUser))
    setToken(newToken)
    setUser(newUser)
  }, [])

  const logout = useCallback(async () => {
    try {
      await authApi.logout()
    } catch {
      // ignore errors on logout
    } finally {
      localStorage.removeItem('access_token')
      localStorage.removeItem('user')
      setToken(null)
      setUser(null)
    }
  }, [])

  return (
    <AuthContext.Provider
      value={{
        user,
        token,
        isAuthenticated: !!token && !!user,
        isLoading: false, // sync init — never loading
        login,
        logout,
      }}
    >
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}
