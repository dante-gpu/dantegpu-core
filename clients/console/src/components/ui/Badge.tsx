import type { ReactNode } from "react";
import { cn } from "@/lib/cn";

type Tone = "neutral" | "ember" | "flux" | "positive" | "caution" | "critical";

const tones: Record<Tone, string> = {
  neutral: "bg-ink-700 text-ink-100 border-ink-500",
  ember: "bg-ember-500/12 text-ember-300 border-ember-500/30",
  flux: "bg-flux-500/12 text-flux-300 border-flux-500/30",
  positive: "bg-positive/12 text-positive border-positive/30",
  caution: "bg-caution/12 text-caution border-caution/30",
  critical: "bg-critical/12 text-critical border-critical/30",
};

export function Badge({
  children,
  tone = "neutral",
  className,
  dot,
}: {
  children: ReactNode;
  tone?: Tone;
  className?: string;
  dot?: boolean;
}) {
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-xs font-medium",
        tones[tone],
        className,
      )}
    >
      {dot && <span className="size-1.5 rounded-full bg-current" />}
      {children}
    </span>
  );
}

// Maps a GPU / job status string to a toned badge.
export function StatusBadge({ status }: { status: string }) {
  const s = status.toLowerCase();
  const map: Record<string, { tone: Tone; label: string }> = {
    available: { tone: "positive", label: "Available" },
    running: { tone: "flux", label: "Running" },
    rented: { tone: "ember", label: "Rented" },
    dispatched: { tone: "flux", label: "Dispatched" },
    pending: { tone: "caution", label: "Pending" },
    searching: { tone: "caution", label: "Searching" },
    assigning: { tone: "caution", label: "Assigning" },
    completed: { tone: "positive", label: "Completed" },
    offline: { tone: "neutral", label: "Offline" },
    failed: { tone: "critical", label: "Failed" },
    cancelled: { tone: "neutral", label: "Cancelled" },
  };
  const entry = map[s] ?? { tone: "neutral" as Tone, label: status };
  return (
    <Badge tone={entry.tone} dot>
      {entry.label}
    </Badge>
  );
}
