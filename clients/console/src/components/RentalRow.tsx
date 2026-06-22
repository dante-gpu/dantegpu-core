import { Link } from "react-router-dom";
import { ChevronRight } from "lucide-react";
import { StatusBadge } from "./ui/Badge";
import { VendorMark } from "./VendorMark";
import { useJob } from "@/hooks/useJob";
import { usdc, relativeTime } from "@/lib/format";
import type { LocalRental } from "@/lib/rentalsStore";

// One row in the rentals list. Polls the job's live scheduler state and shows a
// rough accrued cost derived from the rate and elapsed time.
export function RentalRow({ rental }: { rental: LocalRental }) {
  const { data: job } = useJob(rental.jobId);
  const elapsedHr = (Date.now() - new Date(rental.startedAt).getTime()) / 3_600_000;
  const accrued = Math.max(0, elapsedHr * rental.rateUsdcHour);
  const state = job?.state ?? "pending";

  return (
    <Link
      to={`/rentals/${rental.jobId}`}
      className="flex items-center gap-4 py-3.5 transition-colors hover:bg-ink-800/40"
    >
      <div className="flex min-w-0 flex-1 items-center gap-3">
        <VendorMark vendor={rental.vendor} />
        <div className="min-w-0">
          <p className="truncate text-sm font-medium text-ink-50">{rental.name}</p>
          <p className="truncate text-xs text-ink-400">
            {rental.gpuModel} · started {relativeTime(rental.startedAt)}
          </p>
        </div>
      </div>
      <div className="hidden text-right sm:block">
        <p className="nums text-sm font-medium text-ember-300">{usdc(accrued)}</p>
        <p className="text-xs text-ink-400">accrued</p>
      </div>
      <StatusBadge status={state} />
      <ChevronRight className="size-4 text-ink-500" />
    </Link>
  );
}
