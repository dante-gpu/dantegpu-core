import { Link } from "react-router-dom";
import { Coins, Cpu, HardDrive, Gift, ArrowRight, Plus, Server } from "lucide-react";
import { useAuth } from "@/providers/AuthProvider";
import { useBalance } from "@/hooks/useBalance";
import { useRentals } from "@/hooks/useRentals";
import { useMarketplace } from "@/hooks/useMarketplace";
import { Stat } from "@/components/ui/Stat";
import { Card, CardHeader, CardBody } from "@/components/ui/Card";
import { Button } from "@/components/ui/Button";
import { GpuCard } from "@/components/GpuCard";
import { RentalRow } from "@/components/RentalRow";
import { EmptyState } from "@/components/ui/EmptyState";
import { usdc } from "@/lib/format";
import { useState } from "react";
import { RentModal } from "@/components/RentModal";
import type { GpuListing } from "@/lib/types";

export default function Dashboard() {
  const { user } = useAuth();
  const { data: balance } = useBalance();
  const rentals = useRentals();
  const { data: gpus } = useMarketplace();
  const [renting, setRenting] = useState<GpuListing | null>(null);

  const available = (gpus ?? []).filter((g) => g.status === "available");
  const featured = [...available].sort((a, b) => b.vram_mb - a.vram_mb).slice(0, 3);

  return (
    <div className="space-y-8">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold text-ink-50">
            Welcome back, <span className="text-ember-gradient">{user?.username}</span>
          </h1>
          <p className="mt-1 text-sm text-ink-400">Here is what is happening across your account.</p>
        </div>
        <Link to="/marketplace">
          <Button>
            <Plus className="size-4" /> Rent a GPU
          </Button>
        </Link>
      </div>

      {/* Stats */}
      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <Stat
          label="Available balance"
          value={usdc(balance?.usdc_available, { symbol: false })}
          sub="USDC"
          tone="ember"
          icon={<Coins className="size-4" />}
        />
        <Stat
          label="Reserved in escrow"
          value={usdc(balance?.usdc_reserved, { symbol: false })}
          sub="USDC"
          icon={<HardDrive className="size-4" />}
        />
        <Stat
          label="Active rentals"
          value={rentals.length}
          sub={rentals.length === 1 ? "session" : "sessions"}
          tone="flux"
          icon={<Cpu className="size-4" />}
        />
        <Stat
          label="dGPU rewards"
          value={(balance?.dgpu_rewards ?? 0).toLocaleString()}
          sub="incentive token"
          tone="positive"
          icon={<Gift className="size-4" />}
        />
      </div>

      {/* Active rentals */}
      <Card>
        <CardHeader
          title="Active rentals"
          description="Live sessions you have started from this browser"
          action={
            <Link to="/rentals">
              <Button variant="ghost" size="sm">
                View all <ArrowRight className="size-4" />
              </Button>
            </Link>
          }
        />
        <CardBody className="pt-2">
          {rentals.length === 0 ? (
            <EmptyState
              icon={<HardDrive className="size-7" />}
              title="No active rentals"
              description="Rent a GPU from the marketplace to spin up your first session."
              action={
                <Link to="/marketplace">
                  <Button size="sm">Browse GPUs</Button>
                </Link>
              }
            />
          ) : (
            <div className="divide-y divide-ink-700">
              {rentals.slice(0, 4).map((r) => (
                <RentalRow key={r.jobId} rental={r} />
              ))}
            </div>
          )}
        </CardBody>
      </Card>

      {/* Featured GPUs */}
      <div>
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-lg font-semibold text-ink-50">Top GPUs available now</h2>
          <Link to="/marketplace" className="text-sm font-medium text-ember-300 hover:text-ember-200">
            See marketplace
          </Link>
        </div>
        {featured.length === 0 ? (
          <Card className="p-6">
            <div className="flex items-center gap-3 text-sm text-ink-400">
              <Server className="size-5" />
              No GPUs are online right now. Check back shortly or become a provider.
            </div>
          </Card>
        ) : (
          <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
            {featured.map((gpu) => (
              <GpuCard key={gpu.id} gpu={gpu} onRent={setRenting} />
            ))}
          </div>
        )}
      </div>

      <RentModal gpu={renting} onClose={() => setRenting(null)} />
    </div>
  );
}
