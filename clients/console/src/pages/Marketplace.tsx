import { useMemo, useState } from "react";
import { Search, SlidersHorizontal, Cpu } from "lucide-react";
import { useMarketplace } from "@/hooks/useMarketplace";
import { GpuCard } from "@/components/GpuCard";
import { RentModal } from "@/components/RentModal";
import { SkeletonCard } from "@/components/ui/Skeleton";
import { EmptyState } from "@/components/ui/EmptyState";
import { Button } from "@/components/ui/Button";
import { cn } from "@/lib/cn";
import type { GpuListing, GpuVendor } from "@/lib/types";

type SortKey = "price-asc" | "price-desc" | "vram-desc";

const VENDORS: (GpuVendor | "all")[] = ["all", "nvidia", "amd", "apple", "intel"];

export default function Marketplace() {
  const { data, isLoading, isError, refetch } = useMarketplace();
  const [query, setQuery] = useState("");
  const [vendor, setVendor] = useState<GpuVendor | "all">("all");
  const [sort, setSort] = useState<SortKey>("price-asc");
  const [availableOnly, setAvailableOnly] = useState(true);
  const [renting, setRenting] = useState<GpuListing | null>(null);

  const listings = useMemo(() => {
    let rows = data ?? [];
    if (vendor !== "all") rows = rows.filter((g) => g.vendor === vendor);
    if (availableOnly) rows = rows.filter((g) => g.status === "available");
    if (query.trim()) {
      const q = query.toLowerCase();
      rows = rows.filter(
        (g) => (g.model_name ?? "").toLowerCase().includes(q) || (g.provider_name ?? "").toLowerCase().includes(q),
      );
    }
    return [...rows].sort((a, b) => {
      if (sort === "price-asc") return a.price_usdc_hour - b.price_usdc_hour;
      if (sort === "price-desc") return b.price_usdc_hour - a.price_usdc_hour;
      return b.vram_mb - a.vram_mb;
    });
  }, [data, vendor, availableOnly, query, sort]);

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-1">
        <h1 className="text-2xl font-bold text-ink-50">Marketplace</h1>
        <p className="text-sm text-ink-400">
          {data ? `${data.length} GPUs from providers worldwide` : "Browse available GPUs"}
        </p>
      </div>

      {/* Filters */}
      <div className="flex flex-col gap-3 lg:flex-row lg:items-center">
        <div className="flex flex-1 items-center gap-2 rounded-lg border border-ink-500 bg-ink-850 px-3">
          <Search className="size-4 text-ink-400" />
          <input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search model or provider…"
            className="h-10 w-full bg-transparent text-sm text-ink-50 placeholder:text-ink-400 focus:outline-none"
          />
        </div>

        <div className="flex flex-wrap items-center gap-2">
          {VENDORS.map((v) => (
            <button
              key={v}
              onClick={() => setVendor(v)}
              className={cn(
                "rounded-full border px-3 py-1.5 text-sm font-medium capitalize transition-colors",
                vendor === v
                  ? "border-ember-500 bg-ember-500/10 text-ember-200"
                  : "border-ink-600 text-ink-300 hover:border-ink-500",
              )}
            >
              {v}
            </button>
          ))}
        </div>

        <div className="flex items-center gap-2">
          <SlidersHorizontal className="size-4 text-ink-400" />
          <select
            value={sort}
            onChange={(e) => setSort(e.target.value as SortKey)}
            className="h-10 rounded-lg border border-ink-500 bg-ink-850 px-2 text-sm text-ink-100 focus:border-ember-500 focus:outline-none"
          >
            <option value="price-asc">Price: Low to High</option>
            <option value="price-desc">Price: High to Low</option>
            <option value="vram-desc">VRAM: High to Low</option>
          </select>
          <label className="flex cursor-pointer items-center gap-2 text-sm text-ink-300">
            <input
              type="checkbox"
              checked={availableOnly}
              onChange={(e) => setAvailableOnly(e.target.checked)}
              className="size-4 accent-ember-500"
            />
            Available
          </label>
        </div>
      </div>

      {/* Grid */}
      {isLoading ? (
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <SkeletonCard key={i} />
          ))}
        </div>
      ) : isError ? (
        <EmptyState
          icon={<Cpu className="size-8" />}
          title="Could not load the marketplace"
          description="The gateway did not respond. Check that the services are running."
          action={<Button onClick={() => refetch()}>Retry</Button>}
        />
      ) : listings.length === 0 ? (
        <EmptyState
          icon={<Cpu className="size-8" />}
          title="No GPUs match your filters"
          description="Try widening the vendor filter or turning off the availability toggle."
        />
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
          {listings.map((gpu) => (
            <GpuCard key={gpu.id} gpu={gpu} onRent={setRenting} />
          ))}
        </div>
      )}

      <RentModal gpu={renting} onClose={() => setRenting(null)} />
    </div>
  );
}
