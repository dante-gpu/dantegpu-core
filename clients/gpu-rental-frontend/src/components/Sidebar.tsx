
import { NavLink } from 'react-router-dom'
import {
  HomeIcon,
  CpuChipIcon,
  ClockIcon,
  UserIcon,
  ChartBarIcon,
  CreditCardIcon
} from '@heroicons/react/24/outline'

const navigation = [
  { name: 'Dashboard', href: '/', icon: HomeIcon },
  { name: 'GPU Marketplace', href: '/marketplace', icon: CpuChipIcon },
  { name: 'My Rentals', href: '/my-rentals', icon: ClockIcon },
  { name: 'Analytics', href: '/analytics', icon: ChartBarIcon },
  { name: 'Billing', href: '/billing', icon: CreditCardIcon },
  { name: 'Profile', href: '/profile', icon: UserIcon },
]

function Sidebar() {
  return (
    <div className="w-64 bg-white shadow-sm border-r border-gray-200 min-h-screen">
      <div className="p-4">
        <nav className="space-y-2">
          {navigation.map((item) => (
            <NavLink
              key={item.name}
              to={item.href}
              className={({ isActive }) =>
                `group flex items-center px-3 py-2 text-sm font-medium rounded-md transition-colors ${
                  isActive
                    ? 'bg-blue-50 text-blue-700 border-r-2 border-blue-700'
                    : 'text-gray-600 hover:bg-gray-50 hover:text-gray-900'
                }`
              }
            >
              <item.icon
                className="mr-3 h-5 w-5 flex-shrink-0"
                aria-hidden="true"
              />
              {item.name}
            </NavLink>
          ))}
        </nav>
      </div>
    </div>
  )
}

export default Sidebar
