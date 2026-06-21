import { Fragment } from 'react'
import { Menu, Transition } from '@headlessui/react'
import { 
  BellIcon, 
  UserCircleIcon,
  CogIcon,
  ArrowRightOnRectangleIcon,
  WalletIcon
} from '@heroicons/react/24/outline'
import { useAuth } from '../contexts/AuthContext'
import { Link } from 'react-router-dom'

function Navbar() {
  const { user, logout } = useAuth()

  return (
    <nav className="bg-white shadow-sm border-b border-gray-200">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex justify-between h-16">
          <div className="flex items-center">
            <Link to="/" className="flex items-center">
              <div className="flex-shrink-0">
                <h1 className="text-2xl font-bold text-black font-aeonik">DanteGPU</h1>
              </div>
            </Link>
          </div>

          <div className="flex items-center space-x-4">
            {/* Balance Display */}
            <div className="flex items-center space-x-2 bg-cream-100 px-3 py-2 rounded-lg">
              <WalletIcon className="h-5 w-5 text-cream-600" />
              <span className="text-sm font-medium text-black">
                ${user?.balance?.toFixed(2) || '0.00'}
              </span>
            </div>

            {/* Notifications */}
            <button className="p-2 text-gray-400 hover:text-gray-500 relative">
              <BellIcon className="h-6 w-6" />
              <span className="absolute top-0 right-0 block h-2 w-2 rounded-full bg-red-400 ring-2 ring-white"></span>
            </button>

            {/* User Menu */}
            <Menu as="div" className="relative">
              <Menu.Button className="flex items-center space-x-3 text-sm rounded-full focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500">
                {user?.avatar ? (
                  <img
                    className="h-8 w-8 rounded-full"
                    src={user.avatar}
                    alt={user.name}
                  />
                ) : (
                  <UserCircleIcon className="h-8 w-8 text-gray-400" />
                )}
                <span className="hidden md:block text-gray-700 font-medium">
                  {user?.name}
                </span>
              </Menu.Button>

              <Transition
                as={Fragment}
                enter="transition ease-out duration-100"
                enterFrom="transform opacity-0 scale-95"
                enterTo="transform opacity-100 scale-100"
                leave="transition ease-in duration-75"
                leaveFrom="transform opacity-100 scale-100"
                leaveTo="transform opacity-0 scale-95"
              >
                <Menu.Items className="absolute right-0 mt-2 w-48 rounded-md shadow-lg bg-white ring-1 ring-black ring-opacity-5 focus:outline-none z-50">
                  <div className="py-1">
                    <Menu.Item>
                      {({ active }) => (
                        <Link
                          to="/profile"
                          className={`${
                            active ? 'bg-cream-100' : ''
                          } flex items-center px-4 py-2 text-sm text-black`}
                        >
                          <CogIcon className="mr-3 h-4 w-4" />
                          Profile Settings
                        </Link>
                      )}
                    </Menu.Item>
                    <Menu.Item>
                      {({ active }) => (
                        <button
                          onClick={logout}
                          className={`${
                            active ? 'bg-cream-100' : ''
                          } flex items-center w-full px-4 py-2 text-sm text-black`}
                        >
                          <ArrowRightOnRectangleIcon className="mr-3 h-4 w-4" />
                          Sign out
                        </button>
                      )}
                    </Menu.Item>
                  </div>
                </Menu.Items>
              </Transition>
            </Menu>
          </div>
        </div>
      </div>
    </nav>
  )
}

export default Navbar
