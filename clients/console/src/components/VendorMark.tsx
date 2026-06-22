import { cn } from "@/lib/cn";
import type { GpuVendor } from "@/lib/types";

// A small vendor chip with a brand-ish tint. We avoid shipping vendor logos
// (trademark) and instead use a tinted monogram that stays on-brand.
const styles: Record<GpuVendor, { label: string; cls: string }> = {
  nvidia: { label: "NVIDIA", cls: "bg-[#76b900]/15 text-[#9bd32a] border-[#76b900]/30" },
  amd: { label: "AMD", cls: "bg-[#ED1C24]/12 text-[#ff6b70] border-[#ED1C24]/30" },
  apple: { label: "Apple", cls: "bg-ink-100/10 text-ink-100 border-ink-300/30" },
  intel: { label: "Intel", cls: "bg-[#0071c5]/15 text-[#4ca6e8] border-[#0071c5]/30" },
  unknown: { label: "GPU", cls: "bg-ink-700 text-ink-200 border-ink-500" },
};

export function VendorMark({ vendor }: { vendor: GpuVendor }) {
  const s = styles[vendor] ?? styles.unknown;
  return (
    <span className={cn("inline-flex items-center rounded-md border px-2 py-0.5 text-[11px] font-semibold", s.cls)}>
      {s.label}
    </span>
  );
}
