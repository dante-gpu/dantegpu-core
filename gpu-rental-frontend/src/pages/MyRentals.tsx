import { useState, useEffect } from 'react'
import {
  PlayIcon,
  PauseIcon,
  StopIcon,
  ClockIcon,
  CpuChipIcon,
  ComputerDesktopIcon
} from '@heroicons/react/24/outline'
import { useAuth } from '../contexts/AuthContext'

interface Rental {
  id: string
  gpu_id: string
  gpu_model: string
  gpu_vram: string
  gpu_cuda_cores: number
  provider_name: string
  location: string
  status: string
  started_at: string
  estimated_end: string
  hourly_rate: number
  escrow_amount: number
  duration_hours: number
  current_cost: number
}

function MyRentals() {
  const { user } = useAuth()
  const [activeTab, setActiveTab] = useState<'active' | 'history'>('active')
  const [rentals, setRentals] = useState<Rental[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const apiBaseUrl = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080'

  useEffect(() => {
    if (user) {
      fetchRentals()
    }
  }, [user])

  const fetchRentals = async () => {
    try {
      setLoading(true)
      const token = localStorage.getItem('auth_token')
      const response = await fetch(`${apiBaseUrl}/api/v1/rentals`, {
        headers: {
          'Authorization': `Bearer ${token}`,
          'X-User-ID': user?.id || ''
        }
      })

      if (!response.ok) {
        throw new Error('Failed to fetch rentals')
      }

      const data = await response.json()
      if (data.success) {
        setRentals(data.rentals || [])
      } else {
        throw new Error(data.error || 'Failed to load rentals')
      }
    } catch (err) {
      console.error('Failed to fetch rentals:', err)
      setError(err instanceof Error ? err.message : 'Failed to load rentals')
    } finally {
      setLoading(false)
    }
  }

  const activeRentals = rentals.filter(r => r.status === 'active')
  const rentalHistory = rentals.filter(r => r.status !== 'active')

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'active':
        return <PlayIcon className="h-5 w-5 text-green-600" />
      case 'paused':
        return <PauseIcon className="h-5 w-5 text-yellow-600" />
      case 'completed':
        return <StopIcon className="h-5 w-5 text-gray-600" />
      default:
        return <ClockIcon className="h-5 w-5 text-gray-600" />
    }
  }

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'active':
        return 'text-green-600 bg-green-100'
      case 'paused':
        return 'text-yellow-600 bg-yellow-100'
      case 'completed':
        return 'text-gray-600 bg-gray-100'
      default:
        return 'text-gray-600 bg-gray-100'
    }
  }

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString()
  }

  const formatDuration = (hours: number) => {
    if (hours < 1) {
      return `${Math.round(hours * 60)} minutes`
    }
    return `${hours.toFixed(1)} hours`
  }

  const handleAction = (rentalId: string, action: 'start' | 'pause' | 'stop') => {
    console.log(`${action} rental ${rentalId}`)
  }

  const openTerminal = (instanceId: string) => {
    console.log(`Opening terminal for ${instanceId}`)
  }

  if (loading) {
    return (
      <div className="min-h-screen bg-cream-50 flex items-center justify-center">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-black mx-auto"></div>
          <p className="mt-4 text-gray-600">Loading rentals...</p>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-cream-50 p-6">
      <div className="max-w-7xl mx-auto space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-gray-900">My Rentals</h1>
        <p className="text-gray-600">Manage your GPU rentals and monitor usage</p>
      </div>

      {/* Tabs */}
      <div className="border-b border-gray-200">
        <nav className="-mb-px flex space-x-8">
          <button
            onClick={() => setActiveTab('active')}
            className={`py-2 px-1 border-b-2 font-medium text-sm ${
              activeTab === 'active'
                ? 'border-blue-500 text-blue-600'
                : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
            }`}
          >
            Active Rentals ({activeRentals.length})
          </button>
          <button
            onClick={() => setActiveTab('history')}
            className={`py-2 px-1 border-b-2 font-medium text-sm ${
              activeTab === 'history'
                ? 'border-blue-500 text-blue-600'
                : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
            }`}
          >
            History ({rentalHistory.length})
          </button>
        </nav>
      </div>

      {/* Rental Cards */}
      {error ? (
        <div className="text-center py-12 text-red-600">
          <p>{error}</p>
          <button onClick={fetchRentals} className="mt-4 px-4 py-2 bg-black text-white rounded-lg hover:bg-gray-800">
            Retry
          </button>
        </div>
      ) : rentals.length === 0 ? (
        <div className="text-center py-12 text-gray-600">
          <p>No rentals found</p>
        </div>
      ) : (
      <div className="space-y-4">
        {(activeTab === 'active' ? activeRentals : rentalHistory).map((rental) => (
          <div key={rental.id} className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center space-x-3">
                <CpuChipIcon className="h-8 w-8 text-blue-600" />
                <div>
                  <h3 className="text-lg font-semibold text-gray-900">{rental.gpu_model}</h3>
                  <p className="text-sm text-gray-500">GPU ID: {rental.gpu_id}</p>
                </div>
              </div>
              <div className="flex items-center space-x-2">
                {getStatusIcon(rental.status)}
                <span className={`px-2 py-1 text-xs font-medium rounded-full ${getStatusColor(rental.status)}`}>
                  {rental.status.charAt(0).toUpperCase() + rental.status.slice(1)}
                </span>
              </div>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-6">
              <div className="space-y-3">
                <h4 className="font-medium text-gray-900">GPU Specifications</h4>
                <div className="space-y-2 text-sm">
                  <div className="flex justify-between">
                    <span className="text-gray-500">VRAM:</span>
                    <span>{rental.gpu_vram}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-500">CUDA Cores:</span>
                    <span>{rental.gpu_cuda_cores.toLocaleString()}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-500">Provider:</span>
                    <span>{rental.provider_name}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-500">Location:</span>
                    <span>{rental.location}</span>
                  </div>
                </div>
              </div>

              <div className="space-y-3">
                <h4 className="font-medium text-gray-900">Usage & Timing</h4>
                <div className="space-y-2 text-sm">
                  <div className="flex justify-between">
                    <span className="text-gray-500">Started:</span>
                    <span>{formatDate(rental.started_at)}</span>
                  </div>
                  {rental.estimated_end && (
                    <div className="flex justify-between">
                      <span className="text-gray-500">Estimated End:</span>
                      <span>{formatDate(rental.estimated_end)}</span>
                    </div>
                  )}
                  <div className="flex justify-between">
                    <span className="text-gray-500">Duration:</span>
                    <span>{formatDuration(rental.duration_hours)}</span>
                  </div>
                </div>
              </div>

              <div className="space-y-3">
                <h4 className="font-medium text-gray-900">Cost</h4>
                <div className="space-y-2 text-sm">
                  <div className="flex justify-between">
                    <span className="text-gray-500">Rate:</span>
                    <span>{rental.hourly_rate} dGPU/hour</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-500">Escrow:</span>
                    <span>{rental.escrow_amount} dGPU</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-500">Current Cost:</span>
                    <span className="font-semibold text-green-600">{rental.current_cost.toFixed(2)} dGPU</span>
                  </div>
                  {rental.status === 'active' && (
                    <div className="text-xs text-gray-500">
                      * Cost updates in real-time
                    </div>
                  )}
                </div>
              </div>
            </div>

            {/* Actions */}
            {activeTab === 'active' && (
              <div className="flex items-center justify-between pt-4 border-t border-gray-200">
                <div className="flex space-x-3">
                  {rental.status === 'active' && (
                    <>
                      <button
                        onClick={() => handleAction(rental.id, 'pause')}
                        className="flex items-center px-3 py-2 text-sm font-medium text-yellow-700 bg-yellow-100 rounded-md hover:bg-yellow-200"
                      >
                        <PauseIcon className="h-4 w-4 mr-1" />
                        Pause
                      </button>
                      <button
                        onClick={() => handleAction(rental.id, 'stop')}
                        className="flex items-center px-3 py-2 text-sm font-medium text-red-700 bg-red-100 rounded-md hover:bg-red-200"
                      >
                        <StopIcon className="h-4 w-4 mr-1" />
                        Stop
                      </button>
                    </>
                  )}
                  {rental.status === 'paused' && (
                    <>
                      <button
                        onClick={() => handleAction(rental.id, 'start')}
                        className="flex items-center px-3 py-2 text-sm font-medium text-green-700 bg-green-100 rounded-md hover:bg-green-200"
                      >
                        <PlayIcon className="h-4 w-4 mr-1" />
                        Resume
                      </button>
                      <button
                        onClick={() => handleAction(rental.id, 'stop')}
                        className="flex items-center px-3 py-2 text-sm font-medium text-red-700 bg-red-100 rounded-md hover:bg-red-200"
                      >
                        <StopIcon className="h-4 w-4 mr-1" />
                        Stop
                      </button>
                    </>
                  )}
                </div>

                {rental.status === 'active' && (
                  <button
                    onClick={() => openTerminal(rental.gpu_id)}
                    className="flex items-center px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700"
                  >
                    <ComputerDesktopIcon className="h-4 w-4 mr-2" />
                    Open Terminal
                  </button>
                )}
              </div>
            )}
          </div>
        ))}
      </div>
      )}
      </div>
    </div>
  )
}

export default MyRentals
