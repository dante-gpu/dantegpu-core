import { useWallet } from "@solana/wallet-adapter-react";
import { useWalletModal } from "@solana/wallet-adapter-react-ui";
import { Wallet, LogOut } from "lucide-react";
import { Button } from "./ui/Button";
import { shortAddress } from "@/lib/format";

// A compact wallet connect control that uses the adapter's modal for selection
// but renders in the dante design language instead of the default blue pill.
export function WalletButton({ size = "md" }: { size?: "sm" | "md" }) {
  const { publicKey, disconnect, connecting } = useWallet();
  const { setVisible } = useWalletModal();

  if (publicKey) {
    return (
      <Button variant="secondary" size={size} onClick={() => disconnect()} title={publicKey.toBase58()}>
        <span className="size-2 rounded-full bg-positive" />
        <span className="nums">{shortAddress(publicKey.toBase58())}</span>
        <LogOut className="size-3.5 text-ink-300" />
      </Button>
    );
  }

  return (
    <Button variant="primary" size={size} loading={connecting} onClick={() => setVisible(true)}>
      <Wallet className="size-4" />
      Connect Wallet
    </Button>
  );
}
