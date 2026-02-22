import { useState, useEffect, useCallback } from 'react'
import type { ReactNode } from 'react'
import { getMe, logout as apiLogout, type AuthUser } from './api'
import { AuthContext } from './authState'
import type { AuthState } from './authState'

export function AuthProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<AuthState>({ status: 'loading' })

  useEffect(() => {
    getMe()
      .then((user) => setState({ status: 'authenticated', user }))
      .catch(() => setState({ status: 'unauthenticated' }))
  }, [])

  const setUser = useCallback((user: AuthUser) => {
    setState({ status: 'authenticated', user })
  }, [])

  const logout = useCallback(async () => {
    await apiLogout()
    setState({ status: 'unauthenticated' })
  }, [])

  return (
    <AuthContext.Provider value={{ ...state, setUser, logout }}>
      {children}
    </AuthContext.Provider>
  )
}
