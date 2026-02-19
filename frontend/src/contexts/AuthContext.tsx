import { createContext, useContext, useEffect, useState, type ReactNode } from 'react'
import { fetchMe, logout as apiLogout, type AuthUser } from '../api/client'
import { track } from '../analytics'

interface AuthContextType {
  user: AuthUser | null
  loading: boolean
  login: () => void
  logout: () => void
}

const AuthContext = createContext<AuthContextType>({
  user: null,
  loading: true,
  login: () => {},
  logout: () => {},
})

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<AuthUser | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetchMe().then(u => {
      setUser(u)
      if (u) track('login')
      setLoading(false)
    })
  }, [])

  const login = () => {
    window.location.href = '/api/auth/google'
  }

  const logout = async () => {
    track('logout')
    await apiLogout()
    setUser(null)
  }

  return (
    <AuthContext.Provider value={{ user, loading, login, logout }}>
      {children}
    </AuthContext.Provider>
  )
}

export const useAuth = () => useContext(AuthContext)
