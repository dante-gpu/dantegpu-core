import { createContext, useContext, useState, useEffect, ReactNode } from 'react'

interface User {
  id: string
  email: string
  name: string
  avatar?: string
  balance: number
  verified: boolean
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
      const response = await fetch('/api/auth/profile', {
        headers: {
          'Authorization': `Bearer ${token}`
        }
      })
      
      if (response.ok) {
        const userData = await response.json()
        setUser(userData)
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
    const response = await fetch('/api/auth/login', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ email, password })
    })

    if (!response.ok) {
      const error = await response.json()
      throw new Error(error.message || 'Login failed')
    }

    const { token, user: userData } = await response.json()
    localStorage.setItem('auth_token', token)
    setUser(userData)
  }

  const register = async (email: string, password: string, name: string) => {
    const response = await fetch('/api/auth/register', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ email, password, name })
    })

    if (!response.ok) {
      const error = await response.json()
      throw new Error(error.message || 'Registration failed')
    }

    const { token, user: userData } = await response.json()
    localStorage.setItem('auth_token', token)
    setUser(userData)
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
