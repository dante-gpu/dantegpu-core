import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useWallet } from "@solana/wallet-adapter-react";
import { Coins, Wallet as WalletIcon, ArrowDownToLine, ExternalLink, Gift, Lock } from "lucide-react";
import { useAuth } from "@/providers/AuthProvider";
import { useBalance } from "@/hooks/useBalance";
import { useOnchainUsdc } from "@/hooks/useOnchainUsdc";
import { useDeposit } from "@/hooks/useDeposit";
import { useToast } from "@/components/ui/Toast";
import { WalletButton } from "@/components/WalletButton";
import { Card, CardBody, CardHeader } from "@/components/ui/Card";
import { Button } from "@/components/ui/Button";
import { Badge } from "@/components/ui/Badge";
import { EmptyState } from "@/components/ui/EmptyState";
import { api } from "@/lib/api";
import { explorerUrl } from "@/lib/solana";
import { usdc, timestamp, shortAddress, titleCase } from "@/lib/format";
import { cn } from "@/lib/cn";
import type { Transaction } from "@/lib/types";

const QUICK = [10, 50, 100, 500];

export default function Wallet() {
  const { user } = useAuth();
  const qc = useQueryClient();
  const toast = useToast();
  const { publicKey } = useWallet();
  const { data: balance } = useBalance();
  const { data: onchain } = useOnchainUsdc();
  const { deposit, busy } = useDeposit();
  const [amount, setAmount] = useState("50");

  const walletId = balance?.wallet_id ?? user?.id ?? "";
  const { data: txns } = useQuery<Transaction[]>({
    queryKey: ["transactions", walletId],
    queryFn: () => api.wallet.transactions(walletId),
    enabled: !!walletId,
  });

  async function onDeposit() {
    const value = Number(amount);
    if (!Number.isFinite(value) || value <= 0) {
      toast.error("Enter a valid amount.");
      return;
    }
    try {
      const { signature } = await deposit(value);
      toast.success("USDC transfer confirmed. Crediting your balance…");
      // Notify the billing service so it credits the platform balance.
      try {
        await api.wallet.deposit(walletId, value, signature);
      } catch {
        // The on-chain transfer succeeded; backend reconciliation can catch up.
      }
      qc.invalidateQueries({ queryKey: ["balance"] });
      qc.invalidateQueries({ queryKey: ["transactions"] });
      qc.invalidateQueries({ queryKey: ["onchain-usdc"] });
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Deposit failed.");
    }
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-ink-50">Wallet</h1>
        <p className="mt-1 text-sm text-ink-400">Top up USDC to rent GPUs and track your spending.</p>
      </div>

      {/* Balance cards */}
      <div className="grid gap-4 lg:grid-cols-3">
        <Card className="relative overflow-hidden p-6 lg:col-span-2">
          <div
            className="absolute right-0 top-0 size-48 opacity-40"
            style={{ background: "radial-gradient(circle at 70% 30%, rgb(255 87 34 / 0.25), transparent 60%)" }}
          />
          <div className="relative">
            <span className="text-xs font-medium uppercase tracking-wide text-ink-400">Platform balance</span>
            <div className="mt-2 flex items-end gap-3">
              <span className="nums text-4xl font-bold text-ink-50">{usdc(balance?.usdc_available)}</span>
            </div>
            <div className="mt-4 flex flex-wrap gap-2">
              <Badge tone="ember">
                <Lock className="size-3" /> {usdc(balance?.usdc_reserved)} reserved
              </Badge>
              <Badge tone="positive">
                <Gift className="size-3" /> {(balance?.dgpu_rewards ?? 0).toLocaleString()} dGPU
              </Badge>
            </div>
          </div>
        </Card>

        <Card className="p-6">
          <span className="text-xs font-medium uppercase tracking-wide text-ink-400">On-chain wallet</span>
          {publicKey ? (
            <>
              <div className="nums mt-2 text-3xl font-bold text-flux-300">{usdc(onchain)}</div>
              <p className="mt-1 text-xs text-ink-400">{shortAddress(publicKey.toBase58(), 6, 6)}</p>
            </>
          ) : (
            <div className="mt-3 space-y-3">
              <p className="text-sm text-ink-400">Connect a wallet to deposit USDC.</p>
              <WalletButton size="sm" />
            </div>
          )}
        </Card>
      </div>

      {/* Deposit */}
      <Card>
        <CardHeader title="Deposit USDC" description="Transfer USDC from your wallet into your platform balance." />
        <CardBody className="space-y-4 pt-1">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-end">
            <div className="flex-1">
              <label className="mb-1.5 block text-sm font-medium text-ink-100">Amount</label>
              <div className="flex items-center gap-2 rounded-lg border border-ink-500 bg-ink-850 px-3 focus-within:border-ember-500">
                <Coins className="size-4 text-ember-400" />
                <input
                  value={amount}
                  onChange={(e) => setAmount(e.target.value.replace(/[^0-9.]/g, ""))}
                  inputMode="decimal"
                  className="nums h-11 w-full bg-transparent text-lg font-semibold text-ink-50 focus:outline-none"
                />
                <span className="text-sm text-ink-400">USDC</span>
              </div>
            </div>
            <Button size="lg" loading={busy} disabled={!publicKey} onClick={onDeposit}>
              <ArrowDownToLine className="size-4" /> Deposit
            </Button>
          </div>
          <div className="flex flex-wrap gap-2">
            {QUICK.map((q) => (
              <button
                key={q}
                onClick={() => setAmount(String(q))}
                className={cn(
                  "rounded-full border px-3 py-1 text-sm transition-colors",
                  Number(amount) === q
                    ? "border-ember-500 bg-ember-500/10 text-ember-200"
                    : "border-ink-600 text-ink-300 hover:border-ink-500",
                )}
              >
                {q} USDC
              </button>
            ))}
          </div>
          {!publicKey && (
            <p className="flex items-center gap-2 text-sm text-ink-400">
              <WalletIcon className="size-4" /> Connect a wallet above to enable deposits.
            </p>
          )}
        </CardBody>
      </Card>

      {/* Transactions */}
      <Card>
        <CardHeader title="Transaction history" />
        <CardBody className="pt-1">
          {!txns || txns.length === 0 ? (
            <EmptyState
              icon={<Coins className="size-7" />}
              title="No transactions yet"
              description="Your deposits, charges and rewards will appear here."
            />
          ) : (
            <div className="divide-y divide-ink-700">
              {txns.map((t) => (
                <div key={t.id} className="flex items-center justify-between gap-3 py-3">
                  <div className="min-w-0">
                    <p className="text-sm font-medium text-ink-50">{titleCase(t.type)}</p>
                    <p className="text-xs text-ink-400">{timestamp(t.created_at)}</p>
                  </div>
                  <div className="flex items-center gap-3">
                    <span
                      className={cn(
                        "nums text-sm font-medium",
                        t.type === "deposit" || t.type === "reward" || t.type === "refund"
                          ? "text-positive"
                          : "text-ink-100",
                      )}
                    >
                      {t.type === "charge" ? "-" : "+"}
                      {usdc(t.amount_usdc, { symbol: false })}
                    </span>
                    <Badge tone={t.status === "confirmed" ? "positive" : t.status === "failed" ? "critical" : "caution"}>
                      {t.status}
                    </Badge>
                    {t.signature && (
                      <a
                        href={explorerUrl("tx", t.signature)}
                        target="_blank"
                        rel="noreferrer"
                        className="text-ink-400 hover:text-ink-100"
                      >
                        <ExternalLink className="size-4" />
                      </a>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardBody>
      </Card>
    </div>
  );
}
