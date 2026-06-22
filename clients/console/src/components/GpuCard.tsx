import { MemoryStick, Zap, MapPin, Activity } from "lucide-react";
import { Card } from "./ui/Card";
import { Button } from "./ui/Button";
import { StatusBadge } from "./ui/Badge";
import { VendorMark } from "./VendorMark";
import { usdc, vram, watts } from "@/lib/format";
import type { GpuListing } from "@/lib/types";

// Marketplace tile for one GPU listing. Surfaces the specs a renter actually
// decides on (VRAM, power, location, uptime) and the hourly USDC rate, with a
// primary action that routes to checkout.
export function GpuCard({ gpu, onRent }: { gpu: GpuListing; onRent: (gpu: GpuListing) => void }) {
  const available = gpu.status === "available";
  return (
    <Card hover className="flex flex-col p-5">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="mb-1.5 flex items-center gap-2">
            <VendorMark vendor={gpu.vendor} />
            <StatusBadge status={gpu.status} />
          </div>
          <h3 className="truncate text-lg font-semibold text-ink-50">{gpu.model_name}</h3>
          {gpu.provider_name && <p className="truncate text-xs text-ink-400">by {gpu.provider_name}</p>}
        </div>
      </div>

      <dl className="mt-4 grid grid-cols-2 gap-3 text-sm">
        <Spec icon={<MemoryStick className="size-4" />} label="VRAM" value={vram(gpu.vram_mb)} />
        <Spec icon={<Zap className="size-4" />} label="Power" value={watts(gpu.power_consumption_w)} />
        <Spec
          icon={<MapPin className="size-4" />}
          label="Region"
          value={gpu.location ?? "Global"}
        />
        <Spec
          icon={<Activity className="size-4" />}
          label="Uptime"
          value={gpu.uptime_percent != null ? `${gpu.uptime_percent}%` : "-"}
        />
      </dl>

      <div className="mt-5 flex items-end justify-between border-t border-ink-700 pt-4">
        <div>
          {gpu.price_usdc_hour > 0 ? (
            <>
              <div className="nums text-xl font-semibold text-ember-300">{usdc(gpu.price_usdc_hour)}</div>
              <div className="text-xs text-ink-400">per hour</div>
            </>
          ) : (
            <>
              <div className="text-xl font-semibold text-ink-300">Rate n/a</div>
              <div className="text-xs text-ink-400">pricing unavailable</div>
            </>
          )}
        </div>
        <Button size="sm" disabled={!available || gpu.price_usdc_hour <= 0} onClick={() => onRent(gpu)}>
          {available ? "Rent now" : "Unavailable"}
        </Button>
      </div>
    </Card>
  );
}

function Spec({ icon, label, value }: { icon: React.ReactNode; label: string; value: string }) {
  return (
    <div className="flex items-center gap-2">
      <span className="text-ink-400">{icon}</span>
      <div className="min-w-0">
        <dt className="text-[11px] uppercase tracking-wide text-ink-500">{label}</dt>
        <dd className="truncate font-medium text-ink-100">{value}</dd>
      </div>
    </div>
  );
}
