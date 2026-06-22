import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { useAuth } from "@/providers/AuthProvider";
import type { WalletBalance } from "@/lib/types";

// Platform-side USDC balance (deposited, reserved, dGPU rewards) for the signed
// in user. Distinct from the on-chain wallet balance in useOnchainUsdc.
export function useBalance() {
  const { user } = useAuth();
  return useQuery<WalletBalance>({
    queryKey: ["balance", user?.id],
    queryFn: () => api.wallet.balance(user!.id),
    enabled: !!user?.id,
    refetchInterval: 30_000,
  });
}
