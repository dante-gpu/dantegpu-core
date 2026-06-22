import { useEffect, useMemo, useRef, useState } from "react";
import { Area, AreaChart, ResponsiveContainer, Tooltip, YAxis } from "recharts";
import { Card } from "./ui/Card";
import { usdc, duration } from "@/lib/format";

interface CostMeterProps {
  // Authoritative values from the billing backend.
  startedAt: string;
  rateUsdcHour: number;
  // The last backend-reported cost; the meter interpolates forward from here so
  // the number ticks smoothly between polls.
  backendCostUsdc: number;
  running: boolean;
}

// A live, ticking cost display. Between backend polls it extrapolates cost from
// the rate so renters watch the meter move in real time, then snaps to the
// authoritative figure whenever a fresh backendCostUsdc arrives.
export function CostMeter({ startedAt, rateUsdcHour, backendCostUsdc, running }: CostMeterProps) {
  const startMs = useMemo(() => new Date(startedAt).getTime(), [startedAt]);
  const [now, setNow] = useState(() => Date.now());
  const history = useRef<{ t: number; cost: number }[]>([]);
  const ratePerMs = rateUsdcHour / 3_600_000;

  useEffect(() => {
    if (!running) return;
    const id = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(id);
  }, [running]);

  const elapsedSec = Math.max(0, Math.floor((now - startMs) / 1000));
  // Interpolated cost = backend baseline, advanced by the rate. We keep the max
  // of the two so the displayed cost never visually goes backwards on a poll.
  const projected = backendCostUsdc + Math.max(0, (now - startMs) * ratePerMs - backendCostUsdc);
  const liveCost = Math.max(backendCostUsdc, projected, elapsedSec * (rateUsdcHour / 3600));

  // Sample into a rolling 60-point history for the sparkline.
  if (history.current.length === 0 || now - history.current[history.current.length - 1].t > 1500) {
    history.current = [...history.current, { t: now, cost: liveCost }].slice(-60);
  }

  return (
    <Card className="p-6">
      <div className="flex items-start justify-between">
        <div>
          <p className="text-xs font-medium uppercase tracking-wide text-ink-400">Accrued cost</p>
          <p className="nums mt-1 text-4xl font-bold text-ember-gradient">{usdc(liveCost, { precise: true })}</p>
          <p className="mt-1 text-sm text-ink-300">
            {duration(elapsedSec)} elapsed · {usdc(rateUsdcHour)}/hr
          </p>
        </div>
        {running && (
          <span className="pulse-dot mt-1.5 inline-block size-2.5 rounded-full bg-positive text-positive" />
        )}
      </div>

      <div className="mt-4 h-24">
        <ResponsiveContainer width="100%" height="100%">
          <AreaChart data={history.current} margin={{ top: 4, right: 0, bottom: 0, left: 0 }}>
            <defs>
              <linearGradient id="cost-fill" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor="#ff5722" stopOpacity={0.5} />
                <stop offset="100%" stopColor="#ff5722" stopOpacity={0} />
              </linearGradient>
            </defs>
            <YAxis hide domain={["dataMin", "dataMax"]} />
            <Tooltip
              contentStyle={{
                background: "#16161b",
                border: "1px solid #26262f",
                borderRadius: 8,
                fontSize: 12,
              }}
              labelFormatter={() => ""}
              formatter={(v: number) => [usdc(v, { precise: true }), "cost"]}
            />
            <Area
              type="monotone"
              dataKey="cost"
              stroke="#ff7438"
              strokeWidth={2}
              fill="url(#cost-fill)"
              isAnimationActive={false}
            />
          </AreaChart>
        </ResponsiveContainer>
      </div>
    </Card>
  );
}
