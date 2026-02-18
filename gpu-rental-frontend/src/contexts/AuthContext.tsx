import { createContext, useContext, useState, useEffect, ReactNode } from 'react'

interface User {
  id: string
  email: string
  name: string
  role: string
  avatar?: string
  balance: number
  verified?: boolean
}

interface AuthContextType {
  user: User | null
  login: (email: string, password: string) => Promise<void>
  register: (email: string, password: string, name: string) => Promise<void>
  logout: () => void
  loading: boolean
}

const AuthContext = createContext<AuthContextType | undefined>(undefined)

export function useAuth() {
  const context = useContext(AuthContext)
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider')
  }
  return context
}

interface AuthProviderProps {
  children: ReactNode
}

export function AuthProvider({ children }: AuthProviderProps) {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    // Check if user is logged in on app start
    const token = localStorage.getItem('auth_token')
    if (token) {
      // Validate token and get user info
      fetchUserProfile(token)
    } else {
      setLoading(false)
    }
  }, [])

  const fetchUserProfile = async (token: string) => {
    try {
      const apiBaseUrl = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8090'
      const response = await fetch(`${apiBaseUrl}/profile`, {
        headers: {
          'Authorization': `Bearer ${token}`
        }
      })

      if (response.ok) {
        const result = await response.json()
        if (result.success && result.user) {
          setUser(result.user)
        } else {
          localStorage.removeItem('auth_token')
        }
      } else {
        localStorage.removeItem('auth_token')
      }
    } catch (error) {
      console.error('Failed to fetch user profile:', error)
      localStorage.removeItem('auth_token')
    } finally {
      setLoading(false)
    }
  }

  const login = async (email: string, password: string) => {
    const apiBaseUrl = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8090'
    const response = await fetch(`${apiBaseUrl}/login`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ email, password })
    })

    if (!response.ok) {
      const error = await response.json()
      throw new Error(error.error || 'Login failed')
    }

    const data = await response.json()
    if (!data.success) {
      throw new Error(data.error || 'Login failed')
    }

    localStorage.setItem('auth_token', data.token)
    setUser({
      id: data.user.id,
      name: data.user.name,
      email: data.user.email,
      role: 'user',
      balance: data.user.balance || 0,
      avatar: data.user.avatar,
      verified: data.user.verified
    })
  }

  const register = async (email: string, password: string, name: string) => {
    const apiBaseUrl = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8090'
    const response = await fetch(`${apiBaseUrl}/register`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ email, password, name })
    })

    if (!response.ok) {
      const error = await response.json()
      throw new Error(error.error || 'Registration failed')
    }

    const data = await response.json()
    if (!data.success) {
      throw new Error(data.error || 'Registration failed')
    }

    localStorage.setItem('auth_token', data.token)
    setUser({
      id: data.user.id,
      name: data.user.name,
      email: data.user.email,
      role: 'user',
      balance: data.user.balance || 0,
      avatar: data.user.avatar,
      verified: data.user.verified
    })
  }

  const logout = () => {
    localStorage.removeItem('auth_token')
    setUser(null)
  }

  const value = {
    user,
    login,
    register,
    logout,
    loading
  }

  return (
    <AuthContext.Provider value={value}>
      {children}
    </AuthContext.Provider>
  )
}
