import { useState } from "react";
import { Link } from "react-router-dom";
import { ChevronDown, LogOut, Coins } from "lucide-react";
import { WalletButton } from "@/components/WalletButton";
import { ThemeToggle } from "@/components/ThemeToggle";
import { useAuth } from "@/providers/AuthProvider";
import { useBalance } from "@/hooks/useBalance";
import { usdc } from "@/lib/format";
import { cn } from "@/lib/cn";

export function Topbar() {
  const { user, logout } = useAuth();
  const { data: balance } = useBalance();
  const [menuOpen, setMenuOpen] = useState(false);

  return (
    <header className="sticky top-0 z-30 flex h-16 items-center justify-between gap-4 border-b border-ink-700 bg-ink-950/70 px-5 backdrop-blur">
      <div className="flex items-center gap-3">
        <Link
          to="/wallet"
          className="hidden items-center gap-2 rounded-full border border-ink-600 bg-ink-850 px-3 py-1.5 text-sm transition-colors hover:border-ember-500/50 sm:flex"
        >
          <Coins className="size-4 text-ember-400" />
          <span className="nums font-medium text-ink-50">{usdc(balance?.usdc_available)}</span>
          <span className="text-xs text-ink-400">available</span>
        </Link>
      </div>

      <div className="flex items-center gap-3">
        <ThemeToggle className="hidden sm:grid" />
        <WalletButton size="sm" />

        <div className="relative">
          <button
            onClick={() => setMenuOpen((v) => !v)}
            className="flex items-center gap-2 rounded-lg px-2 py-1.5 transition-colors hover:bg-ink-700"
          >
            <span className="grid size-8 place-items-center rounded-full bg-gradient-to-br from-ember-400 to-ember-600 text-sm font-semibold text-white">
              {(user?.username ?? "?").charAt(0).toUpperCase()}
            </span>
            <span className="hidden text-sm font-medium text-ink-100 sm:block">{user?.username}</span>
            <ChevronDown className={cn("size-4 text-ink-400 transition-transform", menuOpen && "rotate-180")} />
          </button>

          {menuOpen && (
            <>
              <div className="fixed inset-0 z-10" onClick={() => setMenuOpen(false)} />
              <div className="card absolute right-0 z-20 mt-2 w-52 p-1.5">
                <div className="border-b border-ink-700 px-3 py-2">
                  <p className="truncate text-sm font-medium text-ink-50">{user?.username}</p>
                  {user?.email && <p className="truncate text-xs text-ink-400">{user.email}</p>}
                </div>
                <Link
                  to="/settings"
                  onClick={() => setMenuOpen(false)}
                  className="block rounded-md px-3 py-2 text-sm text-ink-200 transition-colors hover:bg-ink-700 hover:text-white"
                >
                  Settings
                </Link>
                <button
                  onClick={() => {
                    setMenuOpen(false);
                    logout();
                  }}
                  className="flex w-full items-center gap-2 rounded-md px-3 py-2 text-sm text-critical transition-colors hover:bg-critical/10"
                >
                  <LogOut className="size-4" /> Sign out
                </button>
              </div>
            </>
          )}
        </div>
      </div>
    </header>
  );
}
