import { useState } from "react";
import { Server, Cpu, Coins, ShieldCheck, Copy, Check, Terminal, Download } from "lucide-react";
import { Card, CardBody, CardHeader } from "@/components/ui/Card";
import { Badge } from "@/components/ui/Badge";
import { ProviderConsole } from "@/components/ProviderConsole";
import { cn } from "@/lib/cn";

const INSTALL_CMD = "curl -fsSL https://get.dantegpu.com/daemon.sh | sh";
const RUN_CMD = "dante-daemon --gateway https://api.dantegpu.com --wallet <YOUR_SOLANA_ADDRESS>";

const steps = [
  {
    icon: Download,
    title: "Install the provider daemon",
    body: "One command installs the daemon on Linux, macOS or Windows. It auto-detects NVIDIA, AMD, Apple and Intel GPUs. Verify your card is seen with dante-daemon --get-gpus-json before going live.",
    code: INSTALL_CMD,
  },
  {
    icon: Terminal,
    title: "Connect your wallet & start",
    body: "Point the daemon at the gateway and your Solana address. Earnings settle to this wallet in USDC.",
    code: RUN_CMD,
  },
  {
    icon: Cpu,
    title: "Get matched to renters",
    body: "Your GPU appears in the marketplace within seconds. The scheduler matches renters by GPU type, VRAM and power, and metering starts the moment a job lands.",
  },
];

export default function ProviderOnboarding() {
  return (
    <div className="space-y-8">
      <div className="flex flex-col gap-3">
        <Badge tone="ember" className="w-fit">
          <Server className="size-3" /> Earn with your hardware
        </Badge>
        <h1 className="text-3xl font-bold text-ink-50">Become a DanteGPU provider</h1>
        <p className="max-w-2xl text-ink-300">
          Put idle GPUs to work. Connect any card from any machine, set your price, and earn USDC for every second
          your hardware is rented, plus dGPU incentive rewards on top.
        </p>
      </div>

      {/* Provider earnings console (for existing providers) */}
      <ProviderConsole />

      {/* Value props */}
      <div className="grid gap-4 sm:grid-cols-3">
        <Perk icon={<Coins className="size-5 text-ember-400" />} title="Paid in USDC" body="Per-second metering, settled on Solana. No payout minimums." />
        <Perk icon={<Cpu className="size-5 text-flux-400" />} title="Any GPU" body="NVIDIA, AMD, Apple Silicon and Intel are all supported out of the box." />
        <Perk icon={<ShieldCheck className="size-5 text-positive" />} title="Escrowed jobs" body="Every rental is bonded in escrow before your GPU runs a single cycle." />
      </div>

      {/* Steps */}
      <div className="space-y-4">
        <h2 className="text-lg font-semibold text-ink-50">Start in three steps</h2>
        {steps.map((s, i) => (
          <Card key={i}>
            <CardBody className="flex flex-col gap-4 sm:flex-row sm:items-start">
              <div className="flex items-center gap-3">
                <span className="grid size-10 shrink-0 place-items-center rounded-xl border border-ink-600 bg-ink-850 text-ember-400">
                  <s.icon className="size-5" />
                </span>
                <span className="text-sm font-semibold text-ink-500 sm:hidden">Step {i + 1}</span>
              </div>
              <div className="min-w-0 flex-1">
                <h3 className="text-base font-semibold text-ink-50">
                  <span className="mr-2 hidden text-ink-500 sm:inline">{i + 1}.</span>
                  {s.title}
                </h3>
                <p className="mt-1 text-sm text-ink-300">{s.body}</p>
                {s.code && <CodeBlock code={s.code} />}
              </div>
            </CardBody>
          </Card>
        ))}
      </div>

      {/* Earnings note */}
      <Card>
        <CardHeader title="How earnings work" />
        <CardBody className="space-y-3 pt-1 text-sm text-ink-300">
          <p>
            You set an hourly rate; renters are billed per second against an escrowed deposit. When a session ends,
            the billing service settles the actual metered usage on-chain and the USDC lands in your wallet, minus a
            small protocol fee.
          </p>
          <p>
            On top of USDC earnings, active providers accrue <span className="text-positive">dGPU</span> incentive
            rewards that scale with uptime and utilization.
          </p>
        </CardBody>
      </Card>
    </div>
  );
}

function Perk({ icon, title, body }: { icon: React.ReactNode; title: string; body: string }) {
  return (
    <Card className="p-5">
      <span className="grid size-10 place-items-center rounded-xl border border-ink-600 bg-ink-850">{icon}</span>
      <h3 className="mt-3 font-semibold text-ink-50">{title}</h3>
      <p className="mt-1 text-sm text-ink-400">{body}</p>
    </Card>
  );
}

function CodeBlock({ code }: { code: string }) {
  const [copied, setCopied] = useState(false);
  function copy() {
    navigator.clipboard.writeText(code).then(() => {
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1500);
    });
  }
  return (
    <div className="mt-3 flex items-center gap-2 rounded-lg border border-ink-600 bg-ink-950/70 px-3 py-2.5">
      <Terminal className="size-4 shrink-0 text-ink-500" />
      <code className="flex-1 overflow-x-auto whitespace-nowrap font-mono text-xs text-flux-300">{code}</code>
      <button
        onClick={copy}
        className={cn("shrink-0 rounded-md p-1.5 transition-colors hover:bg-ink-700", copied ? "text-positive" : "text-ink-400")}
        aria-label="Copy command"
      >
        {copied ? <Check className="size-4" /> : <Copy className="size-4" />}
      </button>
    </div>
  );
}
