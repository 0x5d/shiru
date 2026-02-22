import { createContext } from 'react'
import type { AuthUser } from './api'

type AuthState =
  | { status: 'loading' }
  | { status: 'unauthenticated' }
  | { status: 'authenticated'; user: AuthUser }

export type { AuthState }

export type AuthContextValue = AuthState & {
  setUser: (user: AuthUser) => void
  logout: () => Promise<void>
}

export const AuthContext = createContext<AuthContextValue | null>(null)
