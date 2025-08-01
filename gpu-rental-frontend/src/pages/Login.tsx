import React, { useState } from 'react'
import { Link, Navigate } from 'react-router-dom'
import { useAuth } from '../contexts/AuthContext'
import toast from 'react-hot-toast'

function Login() {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const { user, login } = useAuth()

  if (user) {
    return <Navigate to="/" replace />
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError('')

    try {
      await login(email, password)
      toast.success('Welcome back!')
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Login failed'
      setError(errorMessage)
      toast.error(errorMessage)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-cream-50 py-12 px-4 sm:px-6 lg:px-8">
      <div className="max-w-md w-full space-y-8">
        <div>
          <h2 className="mt-6 text-center text-3xl font-extrabold text-gray-900">
            Sign in to DanteGPU
          </h2>
          <p className="mt-2 text-center text-sm text-gray-600">
            Access powerful GPU computing resources
          </p>
        </div>
        <div className="bg-white rounded-lg shadow-sm border border-cream-200 p-8">
          <form className="space-y-6" onSubmit={handleSubmit}>
            <div>
              <label htmlFor="email" className="block text-sm font-medium text-gray-700 mb-2">
                Email address
              </label>
              <input
                id="email"
                name="email"
                type="email"
                autoComplete="email"
                required
                className="appearance-none relative block w-full px-3 py-2 border border-cream-200 placeholder-cream-600 text-black rounded-md focus:outline-none focus:ring-2 focus:ring-cream-400 focus:border-cream-400 sm:text-sm"
                placeholder="Enter your email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
              />
            </div>
            <div>
              <label htmlFor="password" className="block text-sm font-medium text-gray-700 mb-2">
                Password
              </label>
              <input
                id="password"
                name="password"
                type="password"
                autoComplete="current-password"
                required
                className="appearance-none relative block w-full px-3 py-2 border border-cream-200 placeholder-cream-600 text-black rounded-md focus:outline-none focus:ring-2 focus:ring-cream-400 focus:border-cream-400 sm:text-sm"
                placeholder="Enter your password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
              />
            </div>

            {error && (
              <div className="text-red-600 text-sm text-center bg-red-50 border border-red-200 rounded-md p-3">{error}</div>
            )}

            <div>
              <button
                type="submit"
                disabled={loading}
                className="group relative w-full flex justify-center py-3 px-4 border border-transparent text-sm font-medium rounded-md text-white bg-cream-800 hover:bg-cream-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-cream-400 disabled:opacity-50 transition-colors"
              >
                {loading ? 'Signing in...' : 'Sign in'}
              </button>
            </div>

            <div className="text-center">
              <Link to="/register" className="text-black hover:text-cream-700 font-medium">
                Don't have an account? Sign up
              </Link>
            </div>
          </form>
        </div>
      </div>
    </div>
  )
}

export default Login
