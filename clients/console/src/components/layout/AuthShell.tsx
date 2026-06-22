import type { ReactNode } from "react";
import { Logo } from "@/components/Logo";
import { Cpu, ShieldCheck, Gauge } from "lucide-react";

// Split-screen auth frame: brand/marketing panel on the left, the form on the
// right. The left panel collapses on small screens.
export function AuthShell({ children }: { children: ReactNode }) {
  return (
    <div className="grid min-h-screen lg:grid-cols-2">
      <div className="relative hidden flex-col justify-between overflow-hidden border-r border-ink-700 p-12 lg:flex">
        <div
          className="absolute inset-0 opacity-60"
          style={{
            background:
              "radial-gradient(600px 400px at 30% 20%, rgb(255 87 34 / 0.18), transparent 60%), radial-gradient(500px 400px at 80% 80%, rgb(34 211 238 / 0.10), transparent 55%)",
          }}
        />
        <Logo className="relative" />
        <div className="relative space-y-6">
          <h1 className="font-display text-4xl font-bold leading-tight text-ink-50">
            Rent any GPU.
            <br />
            Pay by the second in <span className="text-ember-gradient">USDC</span>.
          </h1>
          <p className="max-w-md text-ink-300">
            DanteGPU connects you to NVIDIA, AMD, Apple and Intel hardware from providers worldwide.
            Metered usage, transparent pricing, and settlement secured on Solana.
          </p>
          <ul className="space-y-3 text-sm text-ink-200">
            <Feature icon={<Cpu className="size-4 text-ember-400" />}>Any vendor, any GPU class</Feature>
            <Feature icon={<Gauge className="size-4 text-flux-400" />}>Per-second metering, no idle waste</Feature>
            <Feature icon={<ShieldCheck className="size-4 text-positive" />}>Escrowed, on-chain settlement</Feature>
          </ul>
        </div>
        <p className="relative text-xs text-ink-500">© {new Date().getFullYear()} DanteGPU</p>
      </div>

      <div className="flex items-center justify-center p-6">
        <div className="w-full max-w-sm">
          <div className="mb-8 lg:hidden">
            <Logo />
          </div>
          {children}
        </div>
      </div>
    </div>
  );
}

function Feature({ icon, children }: { icon: ReactNode; children: ReactNode }) {
  return (
    <li className="flex items-center gap-3">
      <span className="grid size-8 place-items-center rounded-lg border border-ink-600 bg-ink-850">{icon}</span>
      {children}
    </li>
  );
}
