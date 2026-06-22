import type { ReactNode } from "react";
import { cn } from "@/lib/cn";

export function Stat({
  label,
  value,
  sub,
  icon,
  tone = "default",
  className,
}: {
  label: string;
  value: ReactNode;
  sub?: ReactNode;
  icon?: ReactNode;
  tone?: "default" | "ember" | "flux" | "positive";
  className?: string;
}) {
  const accent = {
    default: "text-ink-50",
    ember: "text-ember-300",
    flux: "text-flux-300",
    positive: "text-positive",
  }[tone];

  return (
    <div className={cn("card p-5", className)}>
      <div className="flex items-center justify-between">
        <span className="text-xs font-medium uppercase tracking-wide text-ink-400">{label}</span>
        {icon && <span className="text-ink-300">{icon}</span>}
      </div>
      <div className={cn("nums mt-2 text-2xl font-semibold", accent)}>{value}</div>
      {sub && <div className="mt-1 text-xs text-ink-300">{sub}</div>}
    </div>
  );
}
