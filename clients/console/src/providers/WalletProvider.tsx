import { useMemo, type ReactNode } from "react";
import { ConnectionProvider, WalletProvider as AdapterWalletProvider } from "@solana/wallet-adapter-react";
import { WalletModalProvider } from "@solana/wallet-adapter-react-ui";
import { PhantomWalletAdapter, SolflareWalletAdapter } from "@solana/wallet-adapter-wallets";
import { rpcEndpoint } from "@/lib/solana";

import "@solana/wallet-adapter-react-ui/styles.css";

// Wraps the app in the Solana wallet stack so any screen can read the connected
// wallet, sign USDC transfers and open the wallet-select modal. We keep the
// adapter list short (Phantom + Solflare) since those cover the vast majority of
// renters; more can be added without touching call sites.
export function WalletProvider({ children }: { children: ReactNode }) {
  const endpoint = useMemo(() => rpcEndpoint(), []);
  const wallets = useMemo(() => [new PhantomWalletAdapter(), new SolflareWalletAdapter()], []);

  return (
    <ConnectionProvider endpoint={endpoint}>
      <AdapterWalletProvider wallets={wallets} autoConnect>
        <WalletModalProvider>{children}</WalletModalProvider>
      </AdapterWalletProvider>
    </ConnectionProvider>
  );
}
