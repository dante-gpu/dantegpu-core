import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import {
  CpuChipIcon,
  CurrencyDollarIcon,
  MagnifyingGlassIcon
} from '@heroicons/react/24/outline'

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

function GPUMarketplace() {
  const [searchTerm, setSearchTerm] = useState('')
  const [filterType, setFilterType] = useState('all')
  const [sortBy, setSortBy] = useState('price')
  const [gpus, setGpus] = useState<GPU[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const apiBaseUrl = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080'

  useEffect(() => {
    fetchGPUs()
  }, [])

  const fetchGPUs = async () => {
    try {
      setLoading(true)
      const response = await fetch(`${apiBaseUrl}/api/v1/gpus`)

      if (!response.ok) {
        throw new Error('Failed to fetch GPUs')
      }

      const data = await response.json()
      if (data.success) {
        setGpus(data.gpus || [])
      } else {
        throw new Error(data.error || 'Failed to load GPUs')
      }
    } catch (err) {
      console.error('Failed to fetch GPUs:', err)
      setError(err instanceof Error ? err.message : 'Failed to load GPUs')
    } finally {
      setLoading(false)
    }
  }

  const getAvailabilityColor = (status: string) => {
    switch (status) {
      case 'available':
        return 'text-green-600 bg-green-100'
      case 'rented':
        return 'text-yellow-600 bg-yellow-100'
      case 'maintenance':
        return 'text-red-600 bg-red-100'
      default:
        return 'text-gray-600 bg-gray-100'
    }
  }

  const filteredGPUs = gpus.filter(gpu => {
    const matchesSearch = gpu.model.toLowerCase().includes(searchTerm.toLowerCase())
    const matchesFilter = filterType === 'all' ||
      (filterType === 'available' && gpu.status === 'available') ||
      (filterType === 'rtx' && gpu.model.includes('RTX')) ||
      (filterType === 'professional' && (gpu.model.includes('A100') || gpu.model.includes('H100') || gpu.model.includes('A40')))
    return matchesSearch && matchesFilter
  })

  const sortedGPUs = [...filteredGPUs].sort((a, b) => {
    switch (sortBy) {
      case 'price':
        return a.price_per_hour - b.price_per_hour
      case 'price-desc':
        return b.price_per_hour - a.price_per_hour
      case 'memory':
        return parseInt(b.vram) - parseInt(a.vram)
      case 'performance':
        return b.cuda_cores - a.cuda_cores
      default:
        return 0
    }
  })

  return (
    <div className="min-h-screen bg-cream-50 p-6">
      <div className="max-w-7xl mx-auto space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">GPU Marketplace</h1>
          <p className="text-gray-600">Find and rent the perfect GPU for your workload</p>
        </div>
      </div>

      {/* Search and Filters */}
      <div className="bg-white rounded-lg p-6 shadow-sm border border-gray-200">
        <div className="flex flex-col md:flex-row gap-4">
          <div className="flex-1">
            <div className="relative">
              <MagnifyingGlassIcon className="absolute left-3 top-1/2 transform -translate-y-1/2 h-5 w-5 text-gray-400" />
              <input
                type="text"
                placeholder="Search GPUs..."
                className="w-full pl-10 pr-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
              />
            </div>
          </div>

          <div className="flex gap-4">
            <select
              className="px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              value={filterType}
              onChange={(e) => setFilterType(e.target.value)}
            >
              <option value="all">All GPUs</option>
              <option value="available">Available Only</option>
              <option value="rtx">RTX Series</option>
              <option value="professional">Professional</option>
            </select>

            <select
              className="px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              value={sortBy}
              onChange={(e) => setSortBy(e.target.value)}
            >
              <option value="price">Price: Low to High</option>
              <option value="price-desc">Price: High to Low</option>
              <option value="performance">Performance</option>
              <option value="memory">Memory</option>
            </select>
          </div>
        </div>
      </div>

      {/* GPU Grid */}
      {loading ? (
        <div className="text-center py-12">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-black mx-auto"></div>
          <p className="mt-4 text-gray-600">Loading GPUs...</p>
        </div>
      ) : error ? (
        <div className="text-center py-12 text-red-600">
          <p>{error}</p>
          <button onClick={fetchGPUs} className="mt-4 px-4 py-2 bg-black text-white rounded-lg hover:bg-gray-800">
            Retry
          </button>
        </div>
      ) : sortedGPUs.length === 0 ? (
        <div className="text-center py-12 text-gray-600">
          <p>No GPUs found matching your criteria</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {sortedGPUs.map((gpu) => (
            <div key={gpu.id} className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden hover:shadow-md transition-shadow">
              <div className="p-6">
                <div className="flex items-center justify-between mb-4">
                  <div className="flex items-center space-x-2">
                    <CpuChipIcon className="h-6 w-6 text-blue-600" />
                    <h3 className="text-lg font-semibold text-gray-900">{gpu.model}</h3>
                  </div>
                  <span className={`px-2 py-1 text-xs font-medium rounded-full ${getAvailabilityColor(gpu.status)}`}>
                    {gpu.status === 'available' ? 'Available' : gpu.status}
                  </span>
              </div>

              <div className="space-y-3 mb-4">
                <div className="flex justify-between text-sm">
                  <span className="text-gray-500">Memory:</span>
                  <span className="font-medium">{gpu.vram}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-gray-500">CUDA Cores:</span>
                  <span className="font-medium">{gpu.cuda_cores.toLocaleString()}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-gray-500">Provider:</span>
                  <span className="font-medium">{gpu.provider_name}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-gray-500">Location:</span>
                  <span className="font-medium">{gpu.location}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-gray-500">Utilization:</span>
                  <span className="font-medium">{gpu.utilization}%</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-gray-500">Temperature:</span>
                  <span className="font-medium">{gpu.temperature}°C</span>
                </div>
              </div>

              <div className="border-t pt-4">
                <div className="flex items-center justify-between mb-3">
                  <div className="flex items-center space-x-1">
                    <CurrencyDollarIcon className="h-4 w-4 text-green-600" />
                    <span className="text-2xl font-bold text-gray-900">{gpu.price_per_hour}</span>
                    <span className="text-gray-500 text-sm">dGPU/hour</span>
                  </div>
                </div>

                <Link
                  to={`/gpu/${gpu.id}`}
                  className={`block w-full py-2 px-4 rounded-lg text-center font-medium transition-colors ${
                    gpu.status === 'available'
                      ? 'bg-blue-600 text-white hover:bg-blue-700'
                      : 'bg-gray-100 text-gray-400 cursor-not-allowed pointer-events-none'
                  }`}
                >
                  {gpu.status === 'available' ? 'Rent Now' : 'Unavailable'}
                </Link>
              </div>
            </div>
          </div>
        ))}
      </div>
      )}
    </div>
  )
}

export default GPUMarketplace
