
import { Routes, Route } from 'react-router-dom'
import Layout from './components/Layout'
import Dashboard from './pages/Dashboard.tsx'
import GPUMarketplace from './pages/GPUMarketplace.tsx'
import MyRentals from './pages/MyRentals.tsx'
import Profile from './pages/Profile.tsx'
import Login from './pages/Login.tsx'
import Register from './pages/Register.tsx'
import GPUDetails from './pages/GPUDetails.tsx'
import { AuthProvider } from './contexts/AuthContext'

function App() {
  return (
    <div className="min-h-screen bg-cream-50">
      <AuthProvider>
        <Routes>
          <Route path="/login" element={<Login />} />
          <Route path="/register" element={<Register />} />
          <Route path="/" element={<Layout />}>
            <Route index element={<Dashboard />} />
            <Route path="marketplace" element={<GPUMarketplace />} />
            <Route path="gpu/:id" element={<GPUDetails />} />
            <Route path="my-rentals" element={<MyRentals />} />
            <Route path="profile" element={<Profile />} />
          </Route>
        </Routes>
      </AuthProvider>
    </div>
  )
}

export default App
