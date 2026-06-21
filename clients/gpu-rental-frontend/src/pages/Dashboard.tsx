import { Link } from 'react-router-dom'
import { useState, useEffect } from 'react'
import {
  CpuChipIcon,
  ClockIcon,
  CurrencyDollarIcon,
  ChartBarIcon,
  PlayIcon,
  PauseIcon
} from '@heroicons/react/24/outline'
import { useAuth } from '../contexts/AuthContext'

interface Stats {
  activeRentals: number
  totalSpent: number
  hoursUsed: number
  savings: number
}

interface Rental {
  id: string
  gpu_id: string
  gpu_model: string
  status: string
  started_at: string
  hourly_rate: number
  duration_hours: number
  total_cost: number
}

function Dashboard() {
  const { user } = useAuth()
  const [stats, setStats] = useState<Stats>({
    activeRentals: 0,
    totalSpent: 0,
    hoursUsed: 0,
    savings: 0
  })
  const [recentRentals, setRecentRentals] = useState<Rental[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const apiBaseUrl = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080'

  useEffect(() => {
    fetchDashboardData()
  }, [])

  const fetchDashboardData = async () => {
    try {
      setLoading(true)
      const token = localStorage.getItem('auth_token')

      // Fetch stats
      const statsResponse = await fetch(`${apiBaseUrl}/api/v1/stats/user`, {
        headers: {
          'Authorization': `Bearer ${token}`
        }
      })

      if (statsResponse.ok) {
        const statsData = await statsResponse.json()
        if (statsData.success) {
          setStats(statsData.stats)
        }
      }

      // Fetch recent rentals
      const rentalsResponse = await fetch(`${apiBaseUrl}/api/v1/rentals?limit=3`, {
        headers: {
          'Authorization': `Bearer ${token}`
        }
      })

      if (rentalsResponse.ok) {
        const rentalsData = await rentalsResponse.json()
        if (rentalsData.success) {
          setRecentRentals(rentalsData.rentals || [])
        }
      }
    } catch (err) {
      console.error('Failed to fetch dashboard data:', err)
      setError('Failed to load dashboard data')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen bg-cream-50 p-6">
      <div className="max-w-7xl mx-auto space-y-6">
        {/* Welcome Section */}
        <div className="bg-white border border-cream-200 rounded-lg p-6 shadow-sm">
        <h1 className="text-2xl font-bold mb-2">
          Welcome back, {user?.name}!
        </h1>
        <p className="text-cream-600">
          Manage your GPU rentals and monitor your computing resources
        </p>
      </div>

      {/* Stats Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <div className="bg-white rounded-lg p-6 shadow-sm border border-cream-200">
          <div className="flex items-center">
            <div className="flex-shrink-0">
              <CpuChipIcon className="h-8 w-8 text-black" />
            </div>
            <div className="ml-4">
              <p className="text-sm font-medium text-cream-600">Active Rentals</p>
              <p className="text-2xl font-semibold text-black">{stats.activeRentals}</p>
            </div>
          </div>
        </div>

        <div className="bg-white rounded-lg p-6 shadow-sm border border-cream-200">
          <div className="flex items-center">
            <div className="flex-shrink-0">
              <CurrencyDollarIcon className="h-8 w-8 text-black" />
            </div>
            <div className="ml-4">
              <p className="text-sm font-medium text-cream-600">Total Spent</p>
              <p className="text-2xl font-semibold text-black">${stats.totalSpent}</p>
            </div>
          </div>
        </div>

        <div className="bg-white rounded-lg p-6 shadow-sm border border-cream-200">
          <div className="flex items-center">
            <div className="flex-shrink-0">
              <ClockIcon className="h-8 w-8 text-black" />
            </div>
            <div className="ml-4">
              <p className="text-sm font-medium text-cream-600">Hours Used</p>
              <p className="text-2xl font-semibold text-black">{stats.hoursUsed}h</p>
            </div>
          </div>
        </div>

        <div className="bg-white rounded-lg p-6 shadow-sm border border-cream-200">
          <div className="flex items-center">
            <div className="flex-shrink-0">
              <ChartBarIcon className="h-8 w-8 text-black" />
            </div>
            <div className="ml-4">
              <p className="text-sm font-medium text-cream-600">Savings</p>
              <p className="text-2xl font-semibold text-black">${stats.savings}</p>
            </div>
          </div>
        </div>
      </div>

      {/* Quick Actions */}
      <div className="bg-white rounded-lg p-6 shadow-sm border border-cream-200">
        <h2 className="text-lg font-semibold text-black mb-4">Quick Actions</h2>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <Link
            to="/marketplace"
            className="flex items-center p-4 border border-cream-200 rounded-lg hover:bg-cream-50 transition-colors"
          >
            <CpuChipIcon className="h-8 w-8 text-black mr-3" />
            <div>
              <p className="font-medium text-black">Browse GPUs</p>
              <p className="text-sm text-cream-600">Find the perfect GPU for your needs</p>
            </div>
          </Link>

          <Link
            to="/my-rentals"
            className="flex items-center p-4 border border-cream-200 rounded-lg hover:bg-cream-50 transition-colors"
          >
            <ClockIcon className="h-8 w-8 text-black mr-3" />
            <div>
              <p className="font-medium text-black">My Rentals</p>
              <p className="text-sm text-cream-600">Manage your active rentals</p>
            </div>
          </Link>

          <Link
            to="/profile"
            className="flex items-center p-4 border border-cream-200 rounded-lg hover:bg-cream-50 transition-colors"
          >
            <CurrencyDollarIcon className="h-8 w-8 text-black mr-3" />
            <div>
              <p className="font-medium text-black">Add Funds</p>
              <p className="text-sm text-cream-600">Top up your account balance</p>
            </div>
          </Link>
        </div>
      </div>

      {/* Recent Rentals */}
      <div className="bg-white rounded-lg p-6 shadow-sm border border-cream-200">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold text-black">Recent Rentals</h2>
          <Link
            to="/my-rentals"
            className="text-sm text-black hover:text-cream-700"
          >
            View all
          </Link>
        </div>
        {loading ? (
          <div className="text-center py-8">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-black mx-auto"></div>
            <p className="mt-2 text-cream-600">Loading rentals...</p>
          </div>
        ) : error ? (
          <div className="text-center py-8 text-red-600">
            {error}
          </div>
        ) : recentRentals.length === 0 ? (
          <div className="text-center py-8">
            <p className="text-cream-600">No rentals yet</p>
            <Link to="/marketplace" className="text-black hover:underline mt-2 inline-block">
              Browse GPUs to get started
            </Link>
          </div>
        ) : (
          <div className="space-y-4">
            {recentRentals.map((rental) => (
              <div key={rental.id} className="flex items-center justify-between p-4 border border-cream-200 rounded-lg">
                <div className="flex items-center space-x-4">
                  <div className="flex-shrink-0">
                    {rental.status === 'active' ? (
                      <PlayIcon className="h-6 w-6 text-black" />
                    ) : (
                      <PauseIcon className="h-6 w-6 text-cream-600" />
                    )}
                  </div>
                  <div>
                    <p className="font-medium text-black">{rental.gpu_model}</p>
                    <p className="text-sm text-cream-600">
                      {rental.duration_hours.toFixed(1)}h • {rental.hourly_rate} dGPU/hour
                    </p>
                  </div>
                </div>
                <div className="text-right">
                  <p className={`text-sm font-medium ${
                    rental.status === 'active' ? 'text-black' : 'text-cream-600'
                  }`}>
                    {rental.status === 'active' ? 'Running' : 'Stopped'}
                  </p>
                  <p className="text-sm text-cream-600">
                    {new Date(rental.started_at).toLocaleDateString()}
                  </p>
                  <p className="text-sm font-medium text-black">
                    {rental.total_cost.toFixed(2)} dGPU
                  </p>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
    </div>
  )
}

export default Dashboard
