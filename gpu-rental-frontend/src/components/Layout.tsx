
import { Outlet, Navigate } from 'react-router-dom'
import Navbar from './Navbar'
import Sidebar from './Sidebar'
import { useAuth } from '../contexts/AuthContext'

function Layout() {
  const { user, loading } = useAuth()

  if (loading) {
    return (
      <div className="min-h-screen bg-cream-50 flex items-center justify-center">
        <div className="animate-spin rounded-full h-32 w-32 border-b-2 border-blue-600"></div>
      </div>
    )
  }

  if (!user) {
    return <Navigate to="/login" replace />
  }

  return (
    <div className="min-h-screen bg-cream-50">
      <Navbar />
      <div className="flex">
        <Sidebar />
        <main className="flex-1 p-6 bg-cream-50">
          <Outlet />
        </main>
      </div>
    </div>
  )
}

export default Layout
