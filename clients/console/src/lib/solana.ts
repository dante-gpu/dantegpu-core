// Solana / USDC helpers shared by the wallet provider and the deposit flow.
// The console reads on-chain USDC balances directly from the RPC and builds the
// SPL transfer that tops up a renter's platform balance.

import { clusterApiUrl, type Cluster } from "@solana/web3.js";

export const USDC_MINT = (import.meta.env.VITE_USDC_MINT as string | undefined) ||
  "Gh9ZwEmdLJ8DscKNTkTqPbNwLNNBjuSzaG9Vp2KGtKJr"; // devnet USDC default

export const USDC_DECIMALS = Number(import.meta.env.VITE_USDC_DECIMALS ?? 6);

export const PLATFORM_DEPOSIT_ADDRESS =
  (import.meta.env.VITE_PLATFORM_DEPOSIT_ADDRESS as string | undefined) || "";

/** Resolve the RPC endpoint from env, preferring an explicit URL over a cluster name. */
export function rpcEndpoint(): string {
  const explicit = import.meta.env.VITE_SOLANA_RPC_URL as string | undefined;
  if (explicit) return explicit;
  const cluster = (import.meta.env.VITE_SOLANA_CLUSTER as Cluster | undefined) || "devnet";
  return clusterApiUrl(cluster);
}

export function solanaCluster(): Cluster {
  return (import.meta.env.VITE_SOLANA_CLUSTER as Cluster | undefined) || "devnet";
}

/** Convert a human USDC amount to base units (atoms) for an SPL transfer. */
export function toUsdcAtoms(amount: number): bigint {
  return BigInt(Math.round(amount * 10 ** USDC_DECIMALS));
}

/** Convert SPL base units back to a human USDC number. */
export function fromUsdcAtoms(atoms: bigint | number): number {
  return Number(atoms) / 10 ** USDC_DECIMALS;
}

/** Explorer link for a signature / address on the active cluster. */
export function explorerUrl(kind: "tx" | "address", value: string): string {
  const cluster = solanaCluster();
  const suffix = cluster === "mainnet-beta" ? "" : `?cluster=${cluster}`;
  return `https://explorer.solana.com/${kind}/${value}${suffix}`;
}
