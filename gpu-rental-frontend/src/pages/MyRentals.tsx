import { useState } from 'react'
import {
  PlayIcon,
  PauseIcon,
  StopIcon,
  ClockIcon,
  CpuChipIcon,
  ComputerDesktopIcon
} from '@heroicons/react/24/outline'

interface Rental {
  id: string
  gpu: {
    name: string
    memory: string
    cores: number
  }
  status: 'running' | 'stopped' | 'paused'
  startTime: string
  endTime?: string
  duration: number // in hours
  costPerHour: number
  totalCost: number
  provider: string
  location: string
  instanceId: string
}

function MyRentals() {
  const [activeTab, setActiveTab] = useState<'active' | 'history'>('active')

  // Mock rental data
  const rentals: Rental[] = [
    {
      id: '1',
      gpu: {
        name: 'NVIDIA RTX 4090',
        memory: '24GB GDDR6X',
        cores: 16384
      },
      status: 'running',
      startTime: '2024-01-15T10:30:00Z',
      duration: 4.2,
      costPerHour: 2.50,
      totalCost: 10.50,
      provider: 'CloudGPU Pro',
      location: 'US-East',
      instanceId: 'gpu-4090-001'
    },
    {
      id: '2',
      gpu: {
        name: 'NVIDIA A100',
        memory: '80GB HBM2e',
        cores: 6912
      },
      status: 'stopped',
      startTime: '2024-01-14T15:20:00Z',
      endTime: '2024-01-14T17:50:00Z',
      duration: 2.5,
      costPerHour: 8.75,
      totalCost: 21.88,
      provider: 'Enterprise Cloud',
      location: 'EU-West',
      instanceId: 'gpu-a100-007'
    },
    {
      id: '3',
      gpu: {
        name: 'NVIDIA RTX 3080',
        memory: '10GB GDDR6X',
        cores: 8704
      },
      status: 'paused',
      startTime: '2024-01-14T09:15:00Z',
      duration: 6.8,
      costPerHour: 1.80,
      totalCost: 12.24,
      provider: 'GPU Farm',
      location: 'US-West',
      instanceId: 'gpu-3080-042'
    }
  ]

  const activeRentals = rentals.filter(r => r.status !== 'stopped')
  const rentalHistory = rentals.filter(r => r.status === 'stopped')

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'running':
        return <PlayIcon className="h-5 w-5 text-green-600" />
      case 'paused':
        return <PauseIcon className="h-5 w-5 text-yellow-600" />
      case 'stopped':
        return <StopIcon className="h-5 w-5 text-gray-600" />
      default:
        return <ClockIcon className="h-5 w-5 text-gray-600" />
    }
  }

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'running':
        return 'text-green-600 bg-green-100'
      case 'paused':
        return 'text-yellow-600 bg-yellow-100'
      case 'stopped':
        return 'text-gray-600 bg-gray-100'
      default:
        return 'text-gray-600 bg-gray-100'
    }
  }

  const handleAction = (rentalId: string, action: 'start' | 'pause' | 'stop') => {
    // Handle rental actions
    console.log(`${action} rental ${rentalId}`)
  }

  const openTerminal = (instanceId: string) => {
    // Open terminal connection
    console.log(`Opening terminal for ${instanceId}`)
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
      <div className="space-y-4">
        {(activeTab === 'active' ? activeRentals : rentalHistory).map((rental) => (
          <div key={rental.id} className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center space-x-3">
                <CpuChipIcon className="h-8 w-8 text-blue-600" />
                <div>
                  <h3 className="text-lg font-semibold text-gray-900">{rental.gpu.name}</h3>
                  <p className="text-sm text-gray-500">Instance: {rental.instanceId}</p>
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
                    <span className="text-gray-500">Memory:</span>
                    <span>{rental.gpu.memory}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-500">CUDA Cores:</span>
                    <span>{rental.gpu.cores.toLocaleString()}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-500">Provider:</span>
                    <span>{rental.provider}</span>
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
                    <span>{new Date(rental.startTime).toLocaleString()}</span>
                  </div>
                  {rental.endTime && (
                    <div className="flex justify-between">
                      <span className="text-gray-500">Ended:</span>
                      <span>{new Date(rental.endTime).toLocaleString()}</span>
                    </div>
                  )}
                  <div className="flex justify-between">
                    <span className="text-gray-500">Duration:</span>
                    <span>{rental.duration.toFixed(1)} hours</span>
                  </div>
                </div>
              </div>

              <div className="space-y-3">
                <h4 className="font-medium text-gray-900">Cost</h4>
                <div className="space-y-2 text-sm">
                  <div className="flex justify-between">
                    <span className="text-gray-500">Rate:</span>
                    <span>${rental.costPerHour}/hour</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-500">Total Cost:</span>
                    <span className="font-semibold text-green-600">${rental.totalCost.toFixed(2)}</span>
                  </div>
                  {rental.status === 'running' && (
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
                  {rental.status === 'running' && (
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
                
                {rental.status === 'running' && (
                  <button
                    onClick={() => openTerminal(rental.instanceId)}
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

      {(activeTab === 'active' ? activeRentals : rentalHistory).length === 0 && (
        <div className="text-center py-12">
          <CpuChipIcon className="mx-auto h-12 w-12 text-gray-400" />
          <h3 className="mt-2 text-sm font-medium text-gray-900">
            {activeTab === 'active' ? 'No active rentals' : 'No rental history'}
          </h3>
          <p className="mt-1 text-sm text-gray-500">
            {activeTab === 'active' 
              ? 'Start by renting a GPU from the marketplace.'
              : 'Your completed rentals will appear here.'
            }
          </p>
        </div>
      )}
      </div>
    </div>
  )
}

export default MyRentals
