import { NavLink } from "react-router-dom";
import { LayoutDashboard, Cpu, HardDrive, Wallet } from "lucide-react";
import { cn } from "@/lib/cn";

const items = [
  { to: "/", label: "Home", icon: LayoutDashboard, end: true },
  { to: "/marketplace", label: "GPUs", icon: Cpu, end: false },
  { to: "/rentals", label: "Rentals", icon: HardDrive, end: false },
  { to: "/wallet", label: "Wallet", icon: Wallet, end: false },
];

// Bottom tab bar shown only below the lg breakpoint, mirroring the sidebar's
// primary destinations.
export function MobileNav() {
  return (
    <nav className="fixed inset-x-0 bottom-0 z-40 flex border-t border-ink-700 bg-ink-900/95 backdrop-blur lg:hidden">
      {items.map(({ to, label, icon: Icon, end }) => (
        <NavLink
          key={to}
          to={to}
          end={end}
          className={({ isActive }) =>
            cn(
              "flex flex-1 flex-col items-center gap-1 py-2.5 text-xs font-medium transition-colors",
              isActive ? "text-ember-300" : "text-ink-400",
            )
          }
        >
          <Icon className="size-5" />
          {label}
        </NavLink>
      ))}
    </nav>
  );
}
