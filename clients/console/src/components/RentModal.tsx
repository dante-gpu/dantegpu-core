import { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import { Clock, Wallet, AlertTriangle } from "lucide-react";
import { Modal } from "./ui/Modal";
import { Button } from "./ui/Button";
import { VendorMark } from "./VendorMark";
import { useToast } from "./ui/Toast";
import { useBalance } from "@/hooks/useBalance";
import { api, ApiError } from "@/lib/api";
import { rentalsStore } from "@/lib/rentalsStore";
import { usdc, vram } from "@/lib/format";
import { cn } from "@/lib/cn";
import type { GpuListing } from "@/lib/types";

const DURATIONS = [
  { label: "1 hour", hours: 1 },
  { label: "4 hours", hours: 4 },
  { label: "12 hours", hours: 12 },
  { label: "24 hours", hours: 24 },
];

// Checkout flow for renting a GPU. Picks a billing horizon, shows the projected
// USDC cost and whether the platform balance covers it, then submits the job to
// the gateway and routes to the live session.
export function RentModal({ gpu, onClose }: { gpu: GpuListing | null; onClose: () => void }) {
  const navigate = useNavigate();
  const qc = useQueryClient();
  const toast = useToast();
  const { data: balance } = useBalance();
  const [hours, setHours] = useState(4);
  const [name, setName] = useState("");
  const [busy, setBusy] = useState(false);

  const estimate = useMemo(() => (gpu ? gpu.price_usdc_hour * hours : 0), [gpu, hours]);
  const available = balance?.usdc_available ?? 0;
  const insufficient = estimate > available;

  if (!gpu) return null;

  async function startRental() {
    if (!gpu) return;
    setBusy(true);
    try {
      const res = await api.jobs.submit({
        type: "gpu-rental",
        name: name.trim() || `${gpu.model_name} rental`,
        gpu_type: gpu.model_name,
        gpu_count: 1,
        min_vram_mb: gpu.vram_mb,
        params: {
          provider_id: gpu.provider_id,
          listing_id: gpu.id,
          duration_hours: hours,
          rate_usdc_hour: gpu.price_usdc_hour,
        },
      });
      rentalsStore.add({
        jobId: res.job_id,
        name: name.trim() || `${gpu.model_name} rental`,
        gpuModel: gpu.model_name,
        vendor: gpu.vendor,
        rateUsdcHour: gpu.price_usdc_hour,
        vramMb: gpu.vram_mb,
        providerId: gpu.provider_id,
        startedAt: new Date().toISOString(),
      });
      toast.success("Rental started. Provisioning your GPU…");
      qc.invalidateQueries({ queryKey: ["balance"] });
      onClose();
      navigate(`/rentals/${res.job_id}`);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Could not start the rental.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal
      open={!!gpu}
      onClose={onClose}
      title="Rent GPU"
      description="Confirm the horizon and start a metered session."
      footer={
        <>
          <Button variant="ghost" onClick={onClose} disabled={busy}>
            Cancel
          </Button>
          <Button onClick={startRental} loading={busy} disabled={insufficient}>
            Start rental
          </Button>
        </>
      }
    >
      <div className="space-y-5">
        <div className="flex items-center justify-between rounded-xl border border-ink-600 bg-ink-850 p-4">
          <div>
            <div className="flex items-center gap-2">
              <VendorMark vendor={gpu.vendor} />
              <span className="font-semibold text-ink-50">{gpu.model_name}</span>
            </div>
            <p className="mt-1 text-xs text-ink-400">
              {vram(gpu.vram_mb)} VRAM · {usdc(gpu.price_usdc_hour)}/hr
            </p>
          </div>
        </div>

        <div>
          <label className="mb-1.5 block text-sm font-medium text-ink-100">Rental name (optional)</label>
          <input
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="llama-70b finetune"
            className="h-11 w-full rounded-lg border border-ink-500 bg-ink-850 px-3 text-sm text-ink-50 placeholder:text-ink-400 focus:border-ember-500 focus:outline-none focus:ring-2 focus:ring-ember-500/25"
          />
        </div>

        <div>
          <label className="mb-2 flex items-center gap-1.5 text-sm font-medium text-ink-100">
            <Clock className="size-4 text-ink-400" /> Billing horizon
          </label>
          <div className="grid grid-cols-4 gap-2">
            {DURATIONS.map((d) => (
              <button
                key={d.hours}
                onClick={() => setHours(d.hours)}
                className={cn(
                  "rounded-lg border px-2 py-2.5 text-sm font-medium transition-colors",
                  hours === d.hours
                    ? "border-ember-500 bg-ember-500/10 text-ember-200"
                    : "border-ink-600 text-ink-300 hover:border-ink-500",
                )}
              >
                {d.label}
              </button>
            ))}
          </div>
          <p className="mt-2 text-xs text-ink-400">
            You are billed per second; the horizon caps the prepaid escrow. Stop anytime to settle for actual usage.
          </p>
        </div>

        <div className="space-y-2 rounded-xl border border-ink-600 bg-ink-850 p-4">
          <Row label="Rate" value={`${usdc(gpu.price_usdc_hour)} / hr`} />
          <Row label={`Max cost (${hours}h)`} value={usdc(estimate)} strong />
          <div className="h-px bg-ink-700" />
          <Row
            label="Your balance"
            value={usdc(available)}
            icon={<Wallet className="size-3.5" />}
            tone={insufficient ? "critical" : "default"}
          />
        </div>

        {insufficient && (
          <div className="flex items-start gap-2 rounded-lg border border-critical/30 bg-critical/10 px-3 py-2.5 text-sm text-critical">
            <AlertTriangle className="mt-0.5 size-4 shrink-0" />
            <span>
              Balance is below the prepaid escrow.{" "}
              <button onClick={() => navigate("/wallet")} className="font-medium underline">
                Deposit USDC
              </button>{" "}
              to continue.
            </span>
          </div>
        )}
      </div>
    </Modal>
  );
}

function Row({
  label,
  value,
  strong,
  icon,
  tone = "default",
}: {
  label: string;
  value: string;
  strong?: boolean;
  icon?: React.ReactNode;
  tone?: "default" | "critical";
}) {
  return (
    <div className="flex items-center justify-between text-sm">
      <span className="flex items-center gap-1.5 text-ink-300">
        {icon}
        {label}
      </span>
      <span
        className={cn(
          "nums",
          strong ? "text-base font-semibold text-ink-50" : "text-ink-100",
          tone === "critical" && "text-critical",
        )}
      >
        {value}
      </span>
    </div>
  );
}
