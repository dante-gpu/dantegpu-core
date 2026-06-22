import { useEffect, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { Banknote, Coins, Cpu, HandCoins, Hash, X } from "lucide-react";
import { Card, CardBody, CardHeader } from "./ui/Card";
import { Button } from "./ui/Button";
import { Stat } from "./ui/Stat";
import { useToast } from "./ui/Toast";
import { useProviderEarnings } from "@/hooks/useProviderEarnings";
import { api, ApiError } from "@/lib/api";
import { usdc, pickNumber, shortAddress } from "@/lib/format";

const KEY = "dante.providerId";

// The earnings side of the provider experience. A provider pastes the provider
// id their daemon registered with; the console then shows live accrued/pending
// USDC and lets them trigger an on-chain payout. The id is remembered locally so
// returning providers land straight on their dashboard.
export function ProviderConsole() {
  const qc = useQueryClient();
  const toast = useToast();
  const [providerId, setProviderId] = useState<string>(() => localStorage.getItem(KEY) ?? "");
  const [draft, setDraft] = useState("");
  const [payingOut, setPayingOut] = useState(false);
  const { data, isLoading, isError } = useProviderEarnings(providerId || undefined);

  useEffect(() => {
    if (providerId) localStorage.setItem(KEY, providerId);
  }, [providerId]);

  function connect() {
    if (draft.trim()) setProviderId(draft.trim());
  }

  function disconnect() {
    localStorage.removeItem(KEY);
    setProviderId("");
    setDraft("");
  }

  async function payout() {
    setPayingOut(true);
    try {
      await api.provider.payout(providerId);
      toast.success("Payout requested. USDC will settle to your wallet.");
      qc.invalidateQueries({ queryKey: ["provider-earnings"] });
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Payout request failed.");
    } finally {
      setPayingOut(false);
    }
  }

  if (!providerId) {
    return (
      <Card>
        <CardHeader title="Provider console" description="Already running a daemon? Check your earnings." />
        <CardBody className="pt-1">
          <div className="flex flex-col gap-3 sm:flex-row">
            <div className="flex flex-1 items-center gap-2 rounded-lg border border-ink-500 bg-ink-850 px-3 focus-within:border-ember-500">
              <Hash className="size-4 text-ink-400" />
              <input
                value={draft}
                onChange={(e) => setDraft(e.target.value)}
                onKeyDown={(e) => e.key === "Enter" && connect()}
                placeholder="Your provider ID (from the daemon)"
                className="nums h-11 w-full bg-transparent text-sm text-ink-50 placeholder:text-ink-400 focus:outline-none"
              />
            </div>
            <Button onClick={connect} disabled={!draft.trim()}>
              View earnings
            </Button>
          </div>
        </CardBody>
      </Card>
    );
  }

  const totalEarned = pickNumber(data, ["total_earned_usdc", "total_earnings", "lifetime_usdc"]) ?? 0;
  const pending = pickNumber(data, ["pending_usdc", "pending_payout", "available_payout"]) ?? 0;
  const paidOut = pickNumber(data, ["paid_out_usdc", "total_paid", "withdrawn_usdc"]) ?? 0;
  const sessions = pickNumber(data, ["total_sessions", "rentals", "active_rentals"]) ?? 0;

  return (
    <Card>
      <CardHeader
        title="Provider console"
        description={`Earnings for ${shortAddress(providerId, 6, 6)}`}
        action={
          <Button variant="ghost" size="sm" onClick={disconnect}>
            <X className="size-4" /> Switch
          </Button>
        }
      />
      <CardBody className="space-y-4 pt-1">
        {isError ? (
          <p className="text-sm text-ink-400">
            Could not load earnings for this provider id. Check the id or that the billing service is reachable.
          </p>
        ) : (
          <>
            <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
              <Stat label="Total earned" value={usdc(totalEarned, { symbol: false })} sub="USDC" tone="ember" icon={<Coins className="size-4" />} />
              <Stat label="Pending payout" value={usdc(pending, { symbol: false })} sub="USDC" tone="positive" icon={<HandCoins className="size-4" />} />
              <Stat label="Paid out" value={usdc(paidOut, { symbol: false })} sub="USDC" icon={<Banknote className="size-4" />} />
              <Stat label="Sessions" value={sessions} tone="flux" icon={<Cpu className="size-4" />} />
            </div>
            <div className="flex items-center justify-between gap-3 rounded-xl border border-ink-600 bg-ink-850 p-4">
              <div>
                <p className="text-sm font-medium text-ink-50">Request payout</p>
                <p className="text-xs text-ink-400">Settle your pending USDC on-chain to your wallet.</p>
              </div>
              <Button onClick={payout} loading={payingOut || isLoading} disabled={pending <= 0}>
                <HandCoins className="size-4" /> Pay out {usdc(pending)}
              </Button>
            </div>
          </>
        )}
      </CardBody>
    </Card>
  );
}
