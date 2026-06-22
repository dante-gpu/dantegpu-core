// Formatting helpers. USDC carries 6 decimals on-chain but we surface it to
// renters as a human dollar-like figure. Keep all money rendering funneled
// through here so precision and symbols stay consistent.

const usdcFmt = new Intl.NumberFormat("en-US", {
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
});

const usdcFmtPrecise = new Intl.NumberFormat("en-US", {
  minimumFractionDigits: 4,
  maximumFractionDigits: 6,
});

/** Render a USDC amount, e.g. 12.5 -> "12.50 USDC". */
export function usdc(amount: number | undefined | null, opts?: { precise?: boolean; symbol?: boolean }): string {
  const v = Number.isFinite(amount as number) ? (amount as number) : 0;
  const body = opts?.precise ? usdcFmtPrecise.format(v) : usdcFmt.format(v);
  return opts?.symbol === false ? body : `${body} USDC`;
}

/** Compact USDC for tight spaces: 1250 -> "1.25K USDC". */
export function usdcCompact(amount: number | undefined | null): string {
  const v = Number.isFinite(amount as number) ? (amount as number) : 0;
  if (Math.abs(v) >= 1000) {
    return `${(v / 1000).toFixed(2)}K USDC`;
  }
  return usdc(v);
}

/** Megabytes -> "24 GB" / "512 MB". */
export function vram(mb: number | undefined | null): string {
  const v = mb ?? 0;
  if (v >= 1024) {
    const gb = v / 1024;
    return `${Number.isInteger(gb) ? gb : gb.toFixed(1)} GB`;
  }
  return `${v} MB`;
}

/** Watts -> "450 W". */
export function watts(w: number | undefined | null): string {
  return `${w ?? 0} W`;
}

/** Seconds -> "1h 04m 12s" style duration. */
export function duration(totalSeconds: number | undefined | null): string {
  const s = Math.max(0, Math.floor(totalSeconds ?? 0));
  const h = Math.floor(s / 3600);
  const m = Math.floor((s % 3600) / 60);
  const sec = s % 60;
  if (h > 0) return `${h}h ${pad(m)}m ${pad(sec)}s`;
  if (m > 0) return `${m}m ${pad(sec)}s`;
  return `${sec}s`;
}

function pad(n: number): string {
  return n.toString().padStart(2, "0");
}

/** ISO timestamp -> "Jun 22, 14:05" local. */
export function timestamp(iso: string | undefined | null): string {
  if (!iso) return "-";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "-";
  return d.toLocaleString("en-US", {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

/** "5 minutes ago" relative time. */
export function relativeTime(iso: string | undefined | null): string {
  if (!iso) return "-";
  const d = new Date(iso).getTime();
  if (Number.isNaN(d)) return "-";
  const diff = Date.now() - d;
  const min = Math.floor(diff / 60000);
  if (min < 1) return "just now";
  if (min < 60) return `${min}m ago`;
  const hr = Math.floor(min / 60);
  if (hr < 24) return `${hr}h ago`;
  const day = Math.floor(hr / 24);
  return `${day}d ago`;
}

/** Truncate a Solana address / long id to "AbCd…WxYz". */
export function shortAddress(addr: string | undefined | null, lead = 4, tail = 4): string {
  if (!addr) return "-";
  if (addr.length <= lead + tail + 1) return addr;
  return `${addr.slice(0, lead)}…${addr.slice(-tail)}`;
}

/** Title-case a vendor / status string. */
export function titleCase(s: string | undefined | null): string {
  if (!s) return "";
  return s.charAt(0).toUpperCase() + s.slice(1);
}

// Probes an object for the first numeric value among candidate keys. Used to
// read figures out of loosely-typed backend payloads (earnings, pricing).
export function pickNumber(obj: unknown, keys: string[]): number | null {
  if (!obj || typeof obj !== "object") return null;
  const o = obj as Record<string, unknown>;
  for (const k of keys) {
    const v = o[k];
    if (typeof v === "number" && Number.isFinite(v)) return v;
    if (typeof v === "string") {
      const n = Number.parseFloat(v);
      if (Number.isFinite(n)) return n;
    }
  }
  return null;
}

// The pricing engine returns `estimated_cost` as either a number, a numeric
// string, or a nested object. Normalize it to a single USDC number, probing the
// likely total fields. Returns null when nothing numeric is found.
export function parseEstimatedCost(raw: unknown): number | null {
  if (typeof raw === "number" && Number.isFinite(raw)) return raw;
  if (typeof raw === "string") {
    const n = Number.parseFloat(raw);
    return Number.isFinite(n) ? n : null;
  }
  if (raw && typeof raw === "object") {
    const obj = raw as Record<string, unknown>;
    const keys = ["total_cost", "total_usdc", "total", "cost", "estimated_cost", "amount", "usdc"];
    for (const k of keys) {
      const v = obj[k];
      if (typeof v === "number" && Number.isFinite(v)) return v;
      if (typeof v === "string") {
        const n = Number.parseFloat(v);
        if (Number.isFinite(n)) return n;
      }
    }
  }
  return null;
}
