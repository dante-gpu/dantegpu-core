import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { parseEstimatedCost } from "@/lib/format";
import type { GpuListing } from "@/lib/types";

// Asks the billing pricing engine for an authoritative cost estimate for a GPU
// over a horizon. Returns the normalized USDC figure (or null if the engine is
// unreachable / returns an unexpected shape, in which case the caller falls
// back to a simple rate*hours projection).
export function useEstimate(gpu: GpuListing | null, hours: number) {
  return useQuery<number | null>({
    queryKey: ["estimate", gpu?.id, hours],
    enabled: !!gpu && hours > 0,
    staleTime: 60_000,
    retry: 0,
    queryFn: async () => {
      if (!gpu) return null;
      const res = await api.pricing.estimate({
        gpu_model: gpu.model_name,
        vram_required_mb: gpu.vram_mb,
        estimated_hours: hours,
        estimated_power_w: gpu.power_consumption_w,
      });
      return parseEstimatedCost(res.estimated_cost);
    },
  });
}
