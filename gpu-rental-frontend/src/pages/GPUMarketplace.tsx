import { useState } from 'react'
import { Link } from 'react-router-dom'
import {
  CpuChipIcon,
  CurrencyDollarIcon,
  MagnifyingGlassIcon
} from '@heroicons/react/24/outline'

interface GPU {
  id: string
  name: string
  memory: string
  cores: number
  pricePerHour: number
  availability: 'available' | 'busy' | 'maintenance'
  provider: string
  location: string
  performance: number
  specs: {
    architecture: string
    memoryBandwidth: string
    tensorCores?: number
  }
}

function GPUMarketplace() {
  const [searchTerm, setSearchTerm] = useState('')
  const [filterType, setFilterType] = useState('all')
  const [sortBy, setSortBy] = useState('price')

  // Mock GPU data - will be replaced with API calls (for now - virjil)
  const gpus: GPU[] = [
    {
      id: '1',
      name: 'NVIDIA RTX 4090',
      memory: '24GB GDDR6X',
      cores: 16384,
      pricePerHour: 2.50,
      availability: 'available',
      provider: 'CloudGPU Pro',
      location: 'US-East',
      performance: 95,
      specs: {
        architecture: 'Ada Lovelace',
        memoryBandwidth: '1008 GB/s',
        tensorCores: 512
      }
    },
    {
      id: '2',
      name: 'NVIDIA A100',
      memory: '80GB HBM2e',
      cores: 6912,
      pricePerHour: 8.75,
      availability: 'available',
      provider: 'Enterprise Cloud',
      location: 'EU-West',
      performance: 100,
      specs: {
        architecture: 'Ampere',
        memoryBandwidth: '2039 GB/s',
        tensorCores: 432
      }
    },
    {
      id: '3',
      name: 'NVIDIA RTX 3080',
      memory: '10GB GDDR6X',
      cores: 8704,
      pricePerHour: 1.80,
      availability: 'busy',
      provider: 'GPU Farm',
      location: 'US-West',
      performance: 85,
      specs: {
        architecture: 'Ampere',
        memoryBandwidth: '760 GB/s',
        tensorCores: 272
      }
    },
    {
      id: '4',
      name: 'NVIDIA H100',
      memory: '80GB HBM3',
      cores: 14592,
      pricePerHour: 12.00,
      availability: 'available',
      provider: 'AI Compute',
      location: 'US-Central',
      performance: 100,
      specs: {
        architecture: 'Hopper',
        memoryBandwidth: '3350 GB/s',
        tensorCores: 456
      }
    }
  ]

  const getAvailabilityColor = (availability: string) => {
    switch (availability) {
      case 'available':
        return 'text-green-600 bg-green-100'
      case 'busy':
        return 'text-yellow-600 bg-yellow-100'
      case 'maintenance':
        return 'text-red-600 bg-red-100'
      default:
        return 'text-gray-600 bg-gray-100'
    }
  }

  const getAvailabilityText = (availability: string) => {
    switch (availability) {
      case 'available':
        return 'Available'
      case 'busy':
        return 'Busy'
      case 'maintenance':
        return 'Maintenance'
      default:
        return 'Unknown'
    }
  }

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
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {gpus.map((gpu) => (
          <div key={gpu.id} className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden hover:shadow-md transition-shadow">
            <div className="p-6">
              <div className="flex items-center justify-between mb-4">
                <div className="flex items-center space-x-2">
                  <CpuChipIcon className="h-6 w-6 text-blue-600" />
                  <h3 className="text-lg font-semibold text-gray-900">{gpu.name}</h3>
                </div>
                <span className={`px-2 py-1 text-xs font-medium rounded-full ${getAvailabilityColor(gpu.availability)}`}>
                  {getAvailabilityText(gpu.availability)}
                </span>
              </div>

              <div className="space-y-3 mb-4">
                <div className="flex justify-between text-sm">
                  <span className="text-gray-500">Memory:</span>
                  <span className="font-medium">{gpu.memory}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-gray-500">CUDA Cores:</span>
                  <span className="font-medium">{gpu.cores.toLocaleString()}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-gray-500">Architecture:</span>
                  <span className="font-medium">{gpu.specs.architecture}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-gray-500">Provider:</span>
                  <span className="font-medium">{gpu.provider}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-gray-500">Location:</span>
                  <span className="font-medium">{gpu.location}</span>
                </div>
              </div>

              <div className="border-t pt-4">
                <div className="flex items-center justify-between mb-3">
                  <div className="flex items-center space-x-1">
                    <CurrencyDollarIcon className="h-4 w-4 text-green-600" />
                    <span className="text-2xl font-bold text-gray-900">${gpu.pricePerHour}</span>
                    <span className="text-gray-500">/hour</span>
                  </div>
                  <div className="text-right">
                    <div className="text-sm text-gray-500">Performance</div>
                    <div className="text-lg font-semibold text-blue-600">{gpu.performance}%</div>
                  </div>
                </div>

                <Link
                  to={`/gpu/${gpu.id}`}
                  className={`w-full py-2 px-4 rounded-lg text-center font-medium transition-colors ${
                    gpu.availability === 'available'
                      ? 'bg-blue-600 text-white hover:bg-blue-700'
                      : 'bg-gray-100 text-gray-400 cursor-not-allowed'
                  }`}
                >
                  {gpu.availability === 'available' ? 'Rent Now' : 'Unavailable'}
                </Link>
              </div>
            </div>
          </div>
        ))}
      </div>

      {/* Load More */}
      <div className="text-center">
        <button className="px-6 py-2 border border-gray-300 rounded-lg text-gray-700 hover:bg-gray-50 transition-colors">
          Load More GPUs
        </button>
      </div>
      </div>
    </div>
  )
}

export default GPUMarketplace
