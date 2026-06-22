import { NavLink } from "react-router-dom";
import { LayoutDashboard, Cpu, Server, Wallet, Settings, HardDrive, Activity } from "lucide-react";
import { Logo } from "@/components/Logo";
import { cn } from "@/lib/cn";

const nav = [
  { to: "/", label: "Dashboard", icon: LayoutDashboard, end: true },
  { to: "/marketplace", label: "Marketplace", icon: Cpu, end: false },
  { to: "/rentals", label: "My Rentals", icon: HardDrive, end: false },
  { to: "/wallet", label: "Wallet", icon: Wallet, end: false },
  { to: "/activity", label: "Activity", icon: Activity, end: false },
  { to: "/provider", label: "Become a Provider", icon: Server, end: false },
  { to: "/settings", label: "Settings", icon: Settings, end: false },
];

export function Sidebar() {
  return (
    <aside className="hidden w-64 shrink-0 flex-col border-r border-ink-700 bg-ink-900/60 lg:flex">
      <div className="flex h-16 items-center px-5">
        <Logo />
      </div>
      <nav className="flex-1 space-y-1 px-3 py-4">
        {nav.map(({ to, label, icon: Icon, end }) => (
          <NavLink
            key={to}
            to={to}
            end={end}
            className={({ isActive }) =>
              cn(
                "flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors",
                isActive
                  ? "bg-ember-500/10 text-ember-200 shadow-[inset_2px_0_0_0_var(--color-ember-500)]"
                  : "text-ink-300 hover:bg-ink-700/60 hover:text-ink-50",
              )
            }
          >
            <Icon className="size-[18px]" />
            {label}
          </NavLink>
        ))}
      </nav>
      <div className="border-t border-ink-700 px-5 py-4">
        <p className="text-xs text-ink-400">
          Settled on <span className="text-flux-300">Solana</span> · billed in{" "}
          <span className="text-ember-300">USDC</span>
        </p>
      </div>
    </aside>
  );
}
