import React, { useState } from 'react'
import {
  UserCircleIcon,
  CreditCardIcon,
  ShieldCheckIcon,
  BellIcon,

  WalletIcon
} from '@heroicons/react/24/outline'
import { useAuth } from '../contexts/AuthContext'
import toast from 'react-hot-toast'

function Profile() {
  const { user } = useAuth()
  const [activeTab, setActiveTab] = useState('profile')
  const [loading, setLoading] = useState(false)

  const [profileData, setProfileData] = useState({
    name: user?.name || '',
    email: user?.email || '',
    phone: '',
    company: '',
    bio: ''
  })

  const [securityData, setSecurityData] = useState({
    currentPassword: '',
    newPassword: '',
    confirmPassword: ''
  })

  const [notificationSettings, setNotificationSettings] = useState({
    emailNotifications: true,
    smsNotifications: false,
    rentalAlerts: true,
    billingAlerts: true,
    maintenanceAlerts: true
  })

  const handleProfileUpdate = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)

    try {
      // API call to update profile
      await new Promise(resolve => setTimeout(resolve, 1000)) // Mock delay
      toast.success('Profile updated successfully!')
    } catch (error) {
      toast.error('Failed to update profile')
    } finally {
      setLoading(false)
    }
  }

  const handlePasswordChange = async (e: React.FormEvent) => {
    e.preventDefault()
    
    if (securityData.newPassword !== securityData.confirmPassword) {
      toast.error('New passwords do not match')
      return
    }

    if (securityData.newPassword.length < 8) {
      toast.error('Password must be at least 8 characters')
      return
    }

    setLoading(true)

    try {
      // API call to change password
      await new Promise(resolve => setTimeout(resolve, 1000)) // Mock delay
      toast.success('Password changed successfully!')
      setSecurityData({ currentPassword: '', newPassword: '', confirmPassword: '' })
    } catch (error) {
      toast.error('Failed to change password')
    } finally {
      setLoading(false)
    }
  }

  const handleAddFunds = () => {
    // Open payment modal or redirect to payment page
    toast.success('Redirecting to payment...')
  }

  const tabs = [
    { id: 'profile', name: 'Profile', icon: UserCircleIcon },
    { id: 'security', name: 'Security', icon: ShieldCheckIcon },
    { id: 'billing', name: 'Billing', icon: CreditCardIcon },
    { id: 'notifications', name: 'Notifications', icon: BellIcon }
  ]

  return (
    <div className="min-h-screen bg-cream-50 p-6">
      <div className="max-w-7xl mx-auto space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Profile Settings</h1>
        <p className="text-gray-600">Manage your account settings and preferences</p>
      </div>

      <div className="flex flex-col lg:flex-row gap-6">
        {/* Sidebar */}
        <div className="lg:w-64">
          <nav className="space-y-1">
            {tabs.map((tab) => (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={`w-full flex items-center px-3 py-2 text-sm font-medium rounded-md ${
                  activeTab === tab.id
                    ? 'bg-blue-50 text-blue-700 border-r-2 border-blue-700'
                    : 'text-gray-600 hover:bg-gray-50 hover:text-gray-900'
                }`}
              >
                <tab.icon className="mr-3 h-5 w-5" />
                {tab.name}
              </button>
            ))}
          </nav>
        </div>

        {/* Content */}
        <div className="flex-1">
          {activeTab === 'profile' && (
            <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
              <h2 className="text-lg font-semibold text-gray-900 mb-6">Profile Information</h2>
              
              <form onSubmit={handleProfileUpdate} className="space-y-6">
                <div className="flex items-center space-x-6">
                  <div className="flex-shrink-0">
                    {user?.avatar ? (
                      <img
                        className="h-20 w-20 rounded-full"
                        src={user.avatar}
                        alt={user.name}
                      />
                    ) : (
                      <UserCircleIcon className="h-20 w-20 text-gray-400" />
                    )}
                  </div>
                  <div>
                    <button
                      type="button"
                      className="px-4 py-2 text-sm font-medium text-blue-600 bg-blue-50 rounded-md hover:bg-blue-100"
                    >
                      Change Avatar
                    </button>
                    <p className="text-xs text-gray-500 mt-1">JPG, GIF or PNG. 1MB max.</p>
                  </div>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-2">
                      Full Name
                    </label>
                    <input
                      type="text"
                      className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                      value={profileData.name}
                      onChange={(e) => setProfileData({ ...profileData, name: e.target.value })}
                    />
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-2">
                      Email Address
                    </label>
                    <input
                      type="email"
                      className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                      value={profileData.email}
                      onChange={(e) => setProfileData({ ...profileData, email: e.target.value })}
                    />
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-2">
                      Phone Number
                    </label>
                    <input
                      type="tel"
                      className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                      value={profileData.phone}
                      onChange={(e) => setProfileData({ ...profileData, phone: e.target.value })}
                    />
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-2">
                      Company
                    </label>
                    <input
                      type="text"
                      className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                      value={profileData.company}
                      onChange={(e) => setProfileData({ ...profileData, company: e.target.value })}
                    />
                  </div>
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Bio
                  </label>
                  <textarea
                    rows={4}
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                    value={profileData.bio}
                    onChange={(e) => setProfileData({ ...profileData, bio: e.target.value })}
                    placeholder="Tell us about yourself..."
                  />
                </div>

                <div className="flex justify-end">
                  <button
                    type="submit"
                    disabled={loading}
                    className="px-6 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50"
                  >
                    {loading ? 'Saving...' : 'Save Changes'}
                  </button>
                </div>
              </form>
            </div>
          )}

          {activeTab === 'security' && (
            <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
              <h2 className="text-lg font-semibold text-gray-900 mb-6">Security Settings</h2>
              
              <form onSubmit={handlePasswordChange} className="space-y-6">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Current Password
                  </label>
                  <input
                    type="password"
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                    value={securityData.currentPassword}
                    onChange={(e) => setSecurityData({ ...securityData, currentPassword: e.target.value })}
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    New Password
                  </label>
                  <input
                    type="password"
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                    value={securityData.newPassword}
                    onChange={(e) => setSecurityData({ ...securityData, newPassword: e.target.value })}
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Confirm New Password
                  </label>
                  <input
                    type="password"
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                    value={securityData.confirmPassword}
                    onChange={(e) => setSecurityData({ ...securityData, confirmPassword: e.target.value })}
                  />
                </div>

                <div className="flex justify-end">
                  <button
                    type="submit"
                    disabled={loading}
                    className="px-6 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50"
                  >
                    {loading ? 'Changing...' : 'Change Password'}
                  </button>
                </div>
              </form>
            </div>
          )}

          {activeTab === 'billing' && (
            <div className="space-y-6">
              {/* Account Balance */}
              <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
                <h2 className="text-lg font-semibold text-gray-900 mb-4">Account Balance</h2>
                <div className="flex items-center justify-between">
                  <div className="flex items-center space-x-3">
                    <WalletIcon className="h-8 w-8 text-green-600" />
                    <div>
                      <p className="text-2xl font-bold text-gray-900">${user?.balance?.toFixed(2) || '0.00'}</p>
                      <p className="text-sm text-gray-500">Available balance</p>
                    </div>
                  </div>
                  <button
                    onClick={handleAddFunds}
                    className="px-6 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700"
                  >
                    Add Funds
                  </button>
                </div>
              </div>

              {/* Payment Methods */}
              <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
                <h2 className="text-lg font-semibold text-gray-900 mb-4">Payment Methods</h2>
                <div className="space-y-4">
                  <div className="flex items-center justify-between p-4 border border-gray-200 rounded-lg">
                    <div className="flex items-center space-x-3">
                      <CreditCardIcon className="h-6 w-6 text-gray-400" />
                      <div>
                        <p className="font-medium">•••• •••• •••• 4242</p>
                        <p className="text-sm text-gray-500">Expires 12/25</p>
                      </div>
                    </div>
                    <span className="px-2 py-1 text-xs font-medium text-green-600 bg-green-100 rounded-full">
                      Default
                    </span>
                  </div>
                  <button className="w-full p-4 border-2 border-dashed border-gray-300 rounded-lg text-gray-500 hover:border-gray-400 hover:text-gray-600">
                    + Add Payment Method
                  </button>
                </div>
              </div>
            </div>
          )}

          {activeTab === 'notifications' && (
            <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
              <h2 className="text-lg font-semibold text-gray-900 mb-6">Notification Preferences</h2>
              
              <div className="space-y-6">
                {Object.entries(notificationSettings).map(([key, value]) => (
                  <div key={key} className="flex items-center justify-between">
                    <div>
                      <p className="font-medium text-gray-900">
                        {key.replace(/([A-Z])/g, ' $1').replace(/^./, str => str.toUpperCase())}
                      </p>
                      <p className="text-sm text-gray-500">
                        {key === 'emailNotifications' && 'Receive notifications via email'}
                        {key === 'smsNotifications' && 'Receive notifications via SMS'}
                        {key === 'rentalAlerts' && 'Get alerts about your GPU rentals'}
                        {key === 'billingAlerts' && 'Get alerts about billing and payments'}
                        {key === 'maintenanceAlerts' && 'Get alerts about system maintenance'}
                      </p>
                    </div>
                    <label className="relative inline-flex items-center cursor-pointer">
                      <input
                        type="checkbox"
                        className="sr-only peer"
                        checked={value}
                        onChange={(e) => setNotificationSettings({
                          ...notificationSettings,
                          [key]: e.target.checked
                        })}
                      />
                      <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
                    </label>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>
      </div>
    </div>
  )
}

export default Profile
