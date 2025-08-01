
import { Link } from 'react-router-dom'
import {
  CpuChipIcon,
  ClockIcon,
  CurrencyDollarIcon,
  ChartBarIcon,
  PlayIcon,
  PauseIcon
} from '@heroicons/react/24/outline'
import { useAuth } from '../contexts/AuthContext'

function Dashboard() {
  const { user } = useAuth()

  // Mock data - will be replaced with real API calls
  const stats = {
    activeRentals: 3,
    totalSpent: 245.67,
    hoursUsed: 127.5,
    savings: 89.23
  }

  const recentRentals = [
    {
      id: '1',
      gpu: 'NVIDIA RTX 4090',
      status: 'running',
      startTime: '2024-01-15T10:30:00Z',
      cost: 2.50,
      usage: '4.2 hours'
    },
    {
      id: '2',
      gpu: 'NVIDIA A100',
      status: 'stopped',
      startTime: '2024-01-14T15:20:00Z',
      cost: 8.75,
      usage: '2.5 hours'
    },
    {
      id: '3',
      gpu: 'NVIDIA RTX 3080',
      status: 'running',
      startTime: '2024-01-14T09:15:00Z',
      cost: 1.80,
      usage: '6.8 hours'
    }
  ]

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
        <div className="space-y-4">
          {recentRentals.map((rental) => (
            <div key={rental.id} className="flex items-center justify-between p-4 border border-cream-200 rounded-lg">
              <div className="flex items-center space-x-4">
                <div className="flex-shrink-0">
                  {rental.status === 'running' ? (
                    <PlayIcon className="h-6 w-6 text-black" />
                  ) : (
                    <PauseIcon className="h-6 w-6 text-cream-600" />
                  )}
                </div>
                <div>
                  <p className="font-medium text-black">{rental.gpu}</p>
                  <p className="text-sm text-cream-600">
                    {rental.usage} • ${rental.cost}/hour
                  </p>
                </div>
              </div>
              <div className="text-right">
                <p className={`text-sm font-medium ${
                  rental.status === 'running' ? 'text-black' : 'text-cream-600'
                }`}>
                  {rental.status === 'running' ? 'Running' : 'Stopped'}
                </p>
                <p className="text-sm text-cream-600">
                  {new Date(rental.startTime).toLocaleDateString()}
                </p>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
    </div>
  )
}

export default Dashboard
