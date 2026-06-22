import { useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useWallet } from "@solana/wallet-adapter-react";
import { ArrowUpFromLine, Wallet } from "lucide-react";
import { Modal } from "./ui/Modal";
import { Button } from "./ui/Button";
import { useToast } from "./ui/Toast";
import { api, ApiError } from "@/lib/api";
import { usdc } from "@/lib/format";

// Withdraws USDC from the platform balance back to a Solana address. Defaults
// the destination to the connected wallet so the common case is one tap.
export function WithdrawModal({
  open,
  onClose,
  walletId,
  available,
}: {
  open: boolean;
  onClose: () => void;
  walletId: string;
  available: number;
}) {
  const qc = useQueryClient();
  const toast = useToast();
  const { publicKey } = useWallet();
  const [amount, setAmount] = useState("");
  const [dest, setDest] = useState(publicKey?.toBase58() ?? "");
  const [busy, setBusy] = useState(false);

  const value = Number(amount);
  const overBalance = value > available;
  const invalid = !Number.isFinite(value) || value <= 0 || !dest.trim() || overBalance;

  async function submit() {
    setBusy(true);
    try {
      await api.wallet.withdraw(walletId, value, dest.trim());
      toast.success("Withdrawal submitted. USDC is on its way.");
      qc.invalidateQueries({ queryKey: ["balance"] });
      qc.invalidateQueries({ queryKey: ["transactions"] });
      onClose();
      setAmount("");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Withdrawal failed.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      title="Withdraw USDC"
      description="Send USDC from your platform balance to a Solana wallet."
      footer={
        <>
          <Button variant="ghost" onClick={onClose} disabled={busy}>
            Cancel
          </Button>
          <Button onClick={submit} loading={busy} disabled={invalid}>
            <ArrowUpFromLine className="size-4" /> Withdraw
          </Button>
        </>
      }
    >
      <div className="space-y-4">
        <div className="flex items-center justify-between rounded-lg border border-ink-600 bg-ink-850 px-3 py-2 text-sm">
          <span className="text-ink-300">Available</span>
          <span className="nums font-medium text-ink-50">{usdc(available)}</span>
        </div>

        <div>
          <label className="mb-1.5 block text-sm font-medium text-ink-100">Amount</label>
          <div className="flex items-center gap-2 rounded-lg border border-ink-500 bg-ink-850 px-3 focus-within:border-ember-500">
            <input
              value={amount}
              onChange={(e) => setAmount(e.target.value.replace(/[^0-9.]/g, ""))}
              inputMode="decimal"
              placeholder="0.00"
              className="nums h-11 w-full bg-transparent text-lg font-semibold text-ink-50 placeholder:text-ink-400 focus:outline-none"
            />
            <button
              onClick={() => setAmount(String(available))}
              className="rounded-md px-2 py-1 text-xs font-medium text-ember-300 hover:bg-ink-700"
            >
              MAX
            </button>
            <span className="text-sm text-ink-400">USDC</span>
          </div>
          {overBalance && <p className="mt-1 text-xs text-critical">Amount exceeds your available balance.</p>}
        </div>

        <div>
          <label className="mb-1.5 block text-sm font-medium text-ink-100">Destination</label>
          <div className="flex items-center gap-2 rounded-lg border border-ink-500 bg-ink-850 px-3 focus-within:border-ember-500">
            <Wallet className="size-4 text-ink-400" />
            <input
              value={dest}
              onChange={(e) => setDest(e.target.value)}
              placeholder="Solana address"
              className="nums h-11 w-full bg-transparent text-sm text-ink-50 placeholder:text-ink-400 focus:outline-none"
            />
            {publicKey && dest !== publicKey.toBase58() && (
              <button
                onClick={() => setDest(publicKey.toBase58())}
                className="shrink-0 rounded-md px-2 py-1 text-xs font-medium text-flux-300 hover:bg-ink-700"
              >
                Use wallet
              </button>
            )}
          </div>
        </div>
      </div>
    </Modal>
  );
}
