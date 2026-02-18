import { useState, useEffect } from 'react'
import { useParams, Link, useNavigate } from 'react-router-dom'
import {
  CpuChipIcon,
  ClockIcon,
  CurrencyDollarIcon,
  ChartBarIcon,
  ArrowLeftIcon,
  PlayIcon,
  InformationCircleIcon
} from '@heroicons/react/24/outline'
import toast from 'react-hot-toast'
import { useAuth } from '../contexts/AuthContext'

interface GPU {
  id: string
  model: string
  vram: string
  cuda_cores: number
  price_per_hour: number
  status: string
  provider_name: string
  location: string
  utilization: number
  temperature: number
}

function GPUDetails() {
  const { id } = useParams()
  const navigate = useNavigate()
  const { user } = useAuth()
  const [gpu, setGpu] = useState<GPU | null>(null)
  const [rentalDuration, setRentalDuration] = useState(1)
  const [escrowAmount, setEscrowAmount] = useState(10)
  const [loading, setLoading] = useState(true)
  const [renting, setRenting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const apiBaseUrl = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080'

  useEffect(() => {
    if (id) {
      fetchGPU()
    }
  }, [id])

  const fetchGPU = async () => {
    try {
      setLoading(true)
      const response = await fetch(`${apiBaseUrl}/api/v1/gpus/${id}`)

      if (!response.ok) {
        throw new Error('GPU not found')
      }

      const data = await response.json()
      if (data.success) {
        setGpu(data.gpu)
      } else {
        throw new Error(data.error || 'Failed to load GPU')
      }
    } catch (err) {
      console.error('Failed to fetch GPU:', err)
      setError(err instanceof Error ? err.message : 'Failed to load GPU')
    } finally {
      setLoading(false)
    }
  }

  const handleRent = async () => {
    if (!user) {
      toast.error('Please login to rent a GPU')
      navigate('/login')
      return
    }

    if (!gpu) return

    setRenting(true)

    try {
      const token = localStorage.getItem('auth_token')
      const response = await fetch(`${apiBaseUrl}/api/v1/rentals`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
          'X-User-ID': user.id
        },
        body: JSON.stringify({
          gpu_id: gpu.id,
          escrow_amount: escrowAmount,
          duration_hours: rentalDuration
        })
      })

      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.error || 'Failed to rent GPU')
      }

      const data = await response.json()
      if (data.success) {
        toast.success(`GPU rented successfully for ${rentalDuration} hour${rentalDuration > 1 ? 's' : ''}!`)
        navigate('/my-rentals')
      } else {
        throw new Error(data.error || 'Failed to rent GPU')
      }
    } catch (error) {
      console.error('Rental error:', error)
      toast.error(error instanceof Error ? error.message : 'Failed to rent GPU')
    } finally {
      setRenting(false)
    }
  }

  if (loading) {
    return (
      <div className="min-h-screen bg-cream-50 flex items-center justify-center">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-black mx-auto"></div>
          <p className="mt-4 text-gray-600">Loading GPU details...</p>
        </div>
      </div>
    )
  }

  if (error || !gpu) {
    return (
      <div className="min-h-screen bg-cream-50 flex items-center justify-center">
        <div className="text-center">
          <p className="text-red-600 mb-4">{error || 'GPU not found'}</p>
          <Link to="/marketplace" className="text-black hover:underline">
            Back to Marketplace
          </Link>
        </div>
      </div>
    )
  }

  const totalCost = gpu.price_per_hour * rentalDuration

  return (
    <div className="min-h-screen bg-cream-50 p-6">
      <div className="max-w-7xl mx-auto space-y-6">
      {/* Back Button */}
      <Link
        to="/marketplace"
        className="inline-flex items-center text-sm text-gray-500 hover:text-gray-700"
      >
        <ArrowLeftIcon className="h-4 w-4 mr-1" />
        Back to Marketplace
      </Link>

      {/* Header */}
      <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-4">
            <CpuChipIcon className="h-12 w-12 text-blue-600" />
            <div>
              <h1 className="text-2xl font-bold text-gray-900">{gpu.model}</h1>
              <p className="text-gray-600">{gpu.provider_name} • {gpu.location}</p>
            </div>
          </div>
          <div className="text-right">
            <div className="text-3xl font-bold text-green-600">{gpu.price_per_hour}</div>
            <div className="text-sm text-gray-500">dGPU/hour</div>
            <div className={`mt-2 px-3 py-1 rounded-full text-sm font-medium inline-block ${
              gpu.status === 'available' ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'
            }`}>
              {gpu.status === 'available' ? 'Available' : 'Unavailable'}
            </div>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Main Content */}
        <div className="lg:col-span-2 space-y-6">
          {/* Specifications */}
          <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
            <h2 className="text-lg font-semibold text-gray-900 mb-4">Specifications</h2>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div className="space-y-3">
                <div className="flex justify-between">
                  <span className="text-gray-500">Model:</span>
                  <span className="font-medium">{gpu.model}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-500">CUDA Cores:</span>
                  <span className="font-medium">{gpu.cuda_cores.toLocaleString()}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-500">VRAM:</span>
                  <span className="font-medium">{gpu.vram}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-500">Provider:</span>
                  <span className="font-medium">{gpu.provider_name}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-500">Location:</span>
                  <span className="font-medium">{gpu.location}</span>
                </div>
              </div>
              <div className="space-y-3">
                <div className="flex justify-between">
                  <span className="text-gray-500">Status:</span>
                  <span className={`font-medium ${gpu.status === 'available' ? 'text-green-600' : 'text-red-600'}`}>
                    {gpu.status}
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-500">Utilization:</span>
                  <span className="font-medium">{gpu.utilization}%</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-500">Temperature:</span>
                  <span className="font-medium">{gpu.temperature}°C</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-500">Price:</span>
                  <span className="font-medium text-green-600">{gpu.price_per_hour} dGPU/hour</span>
                </div>
              </div>
            </div>
          </div>

          {/* Real-time Metrics */}
          <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
            <h2 className="text-lg font-semibold text-gray-900 mb-4">Real-time Metrics</h2>
            <div className="space-y-4">
              <div>
                <div className="flex justify-between mb-1">
                  <span className="text-sm font-medium text-gray-700">GPU Utilization</span>
                  <span className="text-sm text-gray-500">{gpu.utilization}%</span>
                </div>
                <div className="w-full bg-gray-200 rounded-full h-2">
                  <div
                    className="bg-blue-600 h-2 rounded-full"
                    style={{ width: `${gpu.utilization}%` }}
                  ></div>
                </div>
              </div>
              <div>
                <div className="flex justify-between mb-1">
                  <span className="text-sm font-medium text-gray-700">Temperature</span>
                  <span className="text-sm text-gray-500">{gpu.temperature}°C</span>
                </div>
                <div className="w-full bg-gray-200 rounded-full h-2">
                  <div
                    className={`h-2 rounded-full ${
                      gpu.temperature < 60 ? 'bg-green-600' :
                      gpu.temperature < 80 ? 'bg-yellow-600' : 'bg-red-600'
                    }`}
                    style={{ width: `${(gpu.temperature / 100) * 100}%` }}
                  ></div>
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* Sidebar */}
        <div className="space-y-6">
          {/* Rental Form */}
          <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6 sticky top-6">
            <h2 className="text-lg font-semibold text-gray-900 mb-4">Rent This GPU</h2>
            
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Rental Duration
                </label>
                <select
                  value={rentalDuration}
                  onChange={(e) => setRentalDuration(Number(e.target.value))}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                >
                  <option value={1}>1 hour</option>
                  <option value={2}>2 hours</option>
                  <option value={4}>4 hours</option>
                  <option value={8}>8 hours</option>
                  <option value={12}>12 hours</option>
                  <option value={24}>24 hours</option>
                </select>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Escrow Amount (dGPU)
                </label>
                <input
                  type="number"
                  value={escrowAmount}
                  onChange={(e) => setEscrowAmount(Number(e.target.value))}
                  min={totalCost}
                  step={1}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                />
                <p className="text-xs text-gray-500 mt-1">
                  Minimum: {totalCost.toFixed(2)} dGPU (estimated cost)
                </p>
              </div>

              <div className="border-t pt-4">
                <div className="flex justify-between text-sm mb-2">
                  <span>Rate per hour:</span>
                  <span>{gpu.price_per_hour} dGPU</span>
                </div>
                <div className="flex justify-between text-sm mb-2">
                  <span>Duration:</span>
                  <span>{rentalDuration} hour{rentalDuration > 1 ? 's' : ''}</span>
                </div>
                <div className="flex justify-between text-sm mb-2">
                  <span>Estimated Cost:</span>
                  <span>{totalCost.toFixed(2)} dGPU</span>
                </div>
                <div className="flex justify-between font-semibold text-lg border-t pt-2">
                  <span>Escrow Amount:</span>
                  <span className="text-green-600">{escrowAmount} dGPU</span>
                </div>
              </div>

              <button
                onClick={handleRent}
                disabled={renting || gpu.status !== 'available'}
                className={`w-full py-3 px-4 rounded-lg font-medium transition-colors ${
                  gpu.status === 'available' && !renting
                    ? 'bg-blue-600 text-white hover:bg-blue-700'
                    : 'bg-gray-100 text-gray-400 cursor-not-allowed'
                }`}
              >
                {renting ? (
                  <div className="flex items-center justify-center">
                    <div className="animate-spin rounded-full h-5 w-5 border-b-2 border-white mr-2"></div>
                    Processing...
                  </div>
                ) : gpu.status === 'available' ? (
                  <>
                    <PlayIcon className="inline h-5 w-5 mr-2" />
                    Rent Now
                  </>
                ) : (
                  'Unavailable'
                )}
              </button>

              <div className="flex items-start space-x-2 text-xs text-gray-500">
                <InformationCircleIcon className="h-4 w-4 mt-0.5 flex-shrink-0" />
                <p>
                  You will be charged per hour of usage. You can stop the rental at any time.
                  Minimum billing is 1 hour.
                </p>
              </div>
            </div>
          </div>

          {/* Quick Stats */}
          <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
            <h3 className="text-lg font-semibold text-gray-900 mb-4">Quick Stats</h3>
            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <div className="flex items-center space-x-2">
                  <ChartBarIcon className="h-5 w-5 text-blue-600" />
                  <span className="text-sm text-gray-600">Performance</span>
                </div>
                <span className="font-semibold text-blue-600">{gpu.performance}%</span>
              </div>
              <div className="flex items-center justify-between">
                <div className="flex items-center space-x-2">
                  <ClockIcon className="h-5 w-5 text-green-600" />
                  <span className="text-sm text-gray-600">Availability</span>
                </div>
                <span className="font-semibold text-green-600">Available</span>
              </div>
              <div className="flex items-center justify-between">
                <div className="flex items-center space-x-2">
                  <CurrencyDollarIcon className="h-5 w-5 text-purple-600" />
                  <span className="text-sm text-gray-600">Cost Effective</span>
                </div>
                <span className="font-semibold text-purple-600">High</span>
              </div>
            </div>
          </div>
        </div>
      </div>
      </div>
    </div>
  )
}

export default GPUDetails
