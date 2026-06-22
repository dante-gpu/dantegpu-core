import { Navigate, Route, Routes, useLocation } from "react-router-dom";
import { useAuth } from "@/providers/AuthProvider";
import { AppShell } from "@/components/layout/AppShell";
import { FullPageSpinner } from "@/components/ui/Spinner";
import Login from "@/pages/Login";
import Register from "@/pages/Register";
import Dashboard from "@/pages/Dashboard";
import Marketplace from "@/pages/Marketplace";
import RentalSession from "@/pages/RentalSession";
import MyRentals from "@/pages/MyRentals";
import Wallet from "@/pages/Wallet";
import Activity from "@/pages/Activity";
import ProviderOnboarding from "@/pages/ProviderOnboarding";
import Settings from "@/pages/Settings";
import NotFound from "@/pages/NotFound";

// Top-level routing with auth gating. While the session hydrates we hold on a
// spinner; unauthenticated users only reach the auth screens, authenticated
// users get the full app shell.
export default function App() {
  const { user, loading } = useAuth();
  const location = useLocation();

  if (loading) return <FullPageSpinner label="Starting DanteGPU…" />;

  if (!user) {
    return (
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route path="/register" element={<Register />} />
        <Route path="*" element={<Navigate to="/login" replace state={{ from: location }} />} />
      </Routes>
    );
  }

  return (
    <Routes>
      <Route element={<AppShell />}>
        <Route path="/" element={<Dashboard />} />
        <Route path="/marketplace" element={<Marketplace />} />
        <Route path="/rentals" element={<MyRentals />} />
        <Route path="/rentals/:jobId" element={<RentalSession />} />
        <Route path="/wallet" element={<Wallet />} />
        <Route path="/activity" element={<Activity />} />
        <Route path="/provider" element={<ProviderOnboarding />} />
        <Route path="/settings" element={<Settings />} />
        <Route path="*" element={<NotFound />} />
      </Route>
      {/* Authed users hitting auth routes bounce to the dashboard. */}
      <Route path="/login" element={<Navigate to="/" replace />} />
      <Route path="/register" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
