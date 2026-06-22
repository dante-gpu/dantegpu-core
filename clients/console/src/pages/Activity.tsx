import { useLogSocket } from "@/hooks/useLogSocket";
import { LogStream } from "@/components/LogStream";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { cn } from "@/lib/cn";

// Live platform activity: a real WebSocket tail of the gateway's aggregated
// container logs (proxied from Loki). Surfaces a clear connection status so a
// down backend reads as "disconnected" rather than an empty void.
export default function Activity() {
  const { lines, status, clear } = useLogSocket(true);

  const tone = status === "open" ? "positive" : status === "connecting" ? "caution" : "critical";
  const label = status === "open" ? "Live" : status === "connecting" ? "Connecting…" : "Disconnected";

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-ink-50">Activity</h1>
          <p className="mt-1 text-sm text-ink-400">Live log tail across the platform services.</p>
        </div>
        <Badge tone={tone} dot>
          <span className={cn(status === "open" && "pulse-dot")}>{label}</span>
        </Badge>
      </div>

      <LogStream
        lines={lines}
        title="Platform logs"
        heightClass="h-[60vh]"
        action={
          <Button variant="ghost" size="sm" onClick={clear} disabled={lines.length === 0}>
            Clear
          </Button>
        }
      />

      {status === "closed" && lines.length === 0 && (
        <p className="text-center text-sm text-ink-400">
          Not connected to the log backend. Ensure the gateway and Loki are running and{" "}
          <code className="font-mono text-flux-300">VITE_API_BASE_URL</code> points at the gateway.
        </p>
      )}
    </div>
  );
}
