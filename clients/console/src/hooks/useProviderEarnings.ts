import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";
import type { ProviderEarnings } from "@/lib/types";

// Fetches a provider's earnings by id. Enabled only once an id is supplied
// (the user pastes their provider id from the daemon), and polls periodically so
// accruals show up while sessions run.
export function useProviderEarnings(providerId: string | undefined) {
  return useQuery<ProviderEarnings>({
    queryKey: ["provider-earnings", providerId],
    enabled: !!providerId,
    refetchInterval: 30_000,
    retry: 0,
    queryFn: () => api.provider.earnings(providerId!),
  });
}
