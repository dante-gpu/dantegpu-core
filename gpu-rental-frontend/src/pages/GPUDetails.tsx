import { useState } from 'react'
import { useParams, Link } from 'react-router-dom'
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

function GPUDetails() {
  const { id } = useParams()
  console.log('GPU ID:', id) // Use the id parameter
  const [rentalDuration, setRentalDuration] = useState(1)
  const [loading, setLoading] = useState(false)

  // Mock GPU data - will be replaced with API call
  const gpu = {
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
      tensorCores: 512,
      baseClock: '2230 MHz',
      boostClock: '2520 MHz',
      memoryInterface: '384-bit',
      powerConsumption: '450W'
    },
    features: [
      'CUDA Compute Capability 8.9',
      'RT Cores 3rd Gen',
      'Tensor Cores 4th Gen',
      'NVENC/NVDEC Support',
      'PCIe 4.0 Support',
      'DLSS 3.0 Support'
    ],
    benchmarks: {
      gaming: 98,
      compute: 95,
      aiTraining: 92,
      rendering: 96
    },
    description: 'The NVIDIA GeForce RTX 4090 is the ultimate GeForce GPU. It brings an enormous leap in performance, efficiency, and AI-powered graphics. Experience ultra-high performance gaming, incredibly detailed virtual worlds with ray tracing, unprecedented productivity, and new ways to create.'
  }

  const handleRent = async () => {
    setLoading(true)
    
    try {
      // API call to rent GPU
      await new Promise(resolve => setTimeout(resolve, 2000)) // Mock delay
      toast.success(`GPU rented successfully for ${rentalDuration} hour${rentalDuration > 1 ? 's' : ''}!`)
    } catch (error) {
      toast.error('Failed to rent GPU')
    } finally {
      setLoading(false)
    }
  }

  const totalCost = gpu.pricePerHour * rentalDuration

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
              <h1 className="text-2xl font-bold text-gray-900">{gpu.name}</h1>
              <p className="text-gray-600">{gpu.provider} • {gpu.location}</p>
            </div>
          </div>
          <div className="text-right">
            <div className="text-3xl font-bold text-green-600">${gpu.pricePerHour}</div>
            <div className="text-sm text-gray-500">per hour</div>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Main Content */}
        <div className="lg:col-span-2 space-y-6">
          {/* Description */}
          <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
            <h2 className="text-lg font-semibold text-gray-900 mb-4">Description</h2>
            <p className="text-gray-700 leading-relaxed">{gpu.description}</p>
          </div>

          {/* Specifications */}
          <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
            <h2 className="text-lg font-semibold text-gray-900 mb-4">Specifications</h2>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div className="space-y-3">
                <div className="flex justify-between">
                  <span className="text-gray-500">Architecture:</span>
                  <span className="font-medium">{gpu.specs.architecture}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-500">CUDA Cores:</span>
                  <span className="font-medium">{gpu.cores.toLocaleString()}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-500">Memory:</span>
                  <span className="font-medium">{gpu.memory}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-500">Memory Bandwidth:</span>
                  <span className="font-medium">{gpu.specs.memoryBandwidth}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-500">Memory Interface:</span>
                  <span className="font-medium">{gpu.specs.memoryInterface}</span>
                </div>
              </div>
              <div className="space-y-3">
                <div className="flex justify-between">
                  <span className="text-gray-500">Base Clock:</span>
                  <span className="font-medium">{gpu.specs.baseClock}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-500">Boost Clock:</span>
                  <span className="font-medium">{gpu.specs.boostClock}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-500">Tensor Cores:</span>
                  <span className="font-medium">{gpu.specs.tensorCores}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-500">Power Consumption:</span>
                  <span className="font-medium">{gpu.specs.powerConsumption}</span>
                </div>
              </div>
            </div>
          </div>

          {/* Features */}
          <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
            <h2 className="text-lg font-semibold text-gray-900 mb-4">Features</h2>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
              {gpu.features.map((feature, index) => (
                <div key={index} className="flex items-center space-x-2">
                  <div className="w-2 h-2 bg-blue-600 rounded-full"></div>
                  <span className="text-gray-700">{feature}</span>
                </div>
              ))}
            </div>
          </div>

          {/* Benchmarks */}
          <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
            <h2 className="text-lg font-semibold text-gray-900 mb-4">Performance Benchmarks</h2>
            <div className="space-y-4">
              {Object.entries(gpu.benchmarks).map(([category, score]) => (
                <div key={category}>
                  <div className="flex justify-between mb-1">
                    <span className="text-sm font-medium text-gray-700 capitalize">
                      {category.replace(/([A-Z])/g, ' $1').trim()}
                    </span>
                    <span className="text-sm text-gray-500">{score}%</span>
                  </div>
                  <div className="w-full bg-gray-200 rounded-full h-2">
                    <div
                      className="bg-blue-600 h-2 rounded-full"
                      style={{ width: `${score}%` }}
                    ></div>
                  </div>
                </div>
              ))}
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

              <div className="border-t pt-4">
                <div className="flex justify-between text-sm mb-2">
                  <span>Rate per hour:</span>
                  <span>${gpu.pricePerHour}</span>
                </div>
                <div className="flex justify-between text-sm mb-2">
                  <span>Duration:</span>
                  <span>{rentalDuration} hour{rentalDuration > 1 ? 's' : ''}</span>
                </div>
                <div className="flex justify-between font-semibold text-lg border-t pt-2">
                  <span>Total Cost:</span>
                  <span className="text-green-600">${totalCost.toFixed(2)}</span>
                </div>
              </div>

              <button
                onClick={handleRent}
                disabled={loading || gpu.availability !== 'available'}
                className={`w-full py-3 px-4 rounded-lg font-medium transition-colors ${
                  gpu.availability === 'available' && !loading
                    ? 'bg-blue-600 text-white hover:bg-blue-700'
                    : 'bg-gray-100 text-gray-400 cursor-not-allowed'
                }`}
              >
                {loading ? (
                  <div className="flex items-center justify-center">
                    <div className="animate-spin rounded-full h-5 w-5 border-b-2 border-white mr-2"></div>
                    Processing...
                  </div>
                ) : gpu.availability === 'available' ? (
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
