import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";
import type { GpuListing } from "@/lib/types";

// Marketplace listings. Refetches every 20s so availability/pricing stays fresh
// without hammering the gateway. Filtering is done client-side in the page.
export function useMarketplace() {
  return useQuery<GpuListing[]>({
    queryKey: ["marketplace"],
    queryFn: ({ signal }) => api.marketplace.list(signal),
    refetchInterval: 20_000,
  });
}
