import { useQuery } from "@tanstack/react-query";
import { useConnection, useWallet } from "@solana/wallet-adapter-react";
import { PublicKey } from "@solana/web3.js";
import { getAssociatedTokenAddress, getAccount, TokenAccountNotFoundError } from "@solana/spl-token";
import { USDC_MINT, fromUsdcAtoms } from "@/lib/solana";

// Reads the connected wallet's on-chain USDC balance via its associated token
// account. Returns 0 (not an error) when the ATA does not exist yet, since a
// fresh wallet simply holds no USDC.
export function useOnchainUsdc() {
  const { connection } = useConnection();
  const { publicKey } = useWallet();

  return useQuery<number>({
    queryKey: ["onchain-usdc", publicKey?.toBase58()],
    enabled: !!publicKey,
    refetchInterval: 30_000,
    queryFn: async () => {
      if (!publicKey) return 0;
      const mint = new PublicKey(USDC_MINT);
      const ata = await getAssociatedTokenAddress(mint, publicKey);
      try {
        const account = await getAccount(connection, ata);
        return fromUsdcAtoms(account.amount);
      } catch (err) {
        if (err instanceof TokenAccountNotFoundError) return 0;
        throw err;
      }
    },
  });
}
