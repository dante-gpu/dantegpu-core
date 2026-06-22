import { useEffect, useRef, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import { ArrowLeft, Square, Server, MemoryStick, Hash, Clock } from "lucide-react";
import { useJob, isTerminal } from "@/hooks/useJob";
import { rentalsStore } from "@/lib/rentalsStore";
import { api, ApiError } from "@/lib/api";
import { CostMeter } from "@/components/CostMeter";
import { LogStream, type LogLine } from "@/components/LogStream";
import { Card, CardBody, CardHeader } from "@/components/ui/Card";
import { Button } from "@/components/ui/Button";
import { StatusBadge } from "@/components/ui/Badge";
import { VendorMark } from "@/components/VendorMark";
import { Modal } from "@/components/ui/Modal";
import { useToast } from "@/components/ui/Toast";
import { FullPageSpinner } from "@/components/ui/Spinner";
import { vram, shortAddress, timestamp } from "@/lib/format";
import type { JobState } from "@/lib/types";

// Human-readable narration for each scheduler state, used to synthesize the log
// stream from real state transitions observed while polling.
const stateNarration: Record<JobState, { text: string; level: LogLine["level"] }> = {
  pending: { text: "Job queued, waiting for the scheduler to pick it up.", level: "system" },
  searching: { text: "Searching for a provider that matches your GPU and VRAM requirements…", level: "info" },
  assigning: { text: "Provider found. Reserving the slot and validating billing.", level: "info" },
  dispatched: { text: "Task dispatched to the provider daemon. Container is starting.", level: "info" },
  running: { text: "GPU is live. Metering has started and you are being billed per second.", level: "system" },
  completed: { text: "Session completed. Final settlement submitted on-chain.", level: "system" },
  failed: { text: "Session failed. Escrow will be released back to your balance.", level: "error" },
  cancelled: { text: "Session stopped by you. Settling for actual usage.", level: "warn" },
};

export default function RentalSession() {
  const { jobId } = useParams<{ jobId: string }>();
  const navigate = useNavigate();
  const qc = useQueryClient();
  const toast = useToast();
  const rental = jobId ? rentalsStore.get(jobId) : undefined;
  const { data: job, isLoading } = useJob(jobId);
  const [logs, setLogs] = useState<LogLine[]>([]);
  const [confirmStop, setConfirmStop] = useState(false);
  const [stopping, setStopping] = useState(false);
  const lastState = useRef<JobState | null>(null);

  // Append a log line whenever the scheduler state actually changes.
  useEffect(() => {
    const state = job?.state;
    if (!state || state === lastState.current) return;
    lastState.current = state;
    const n = stateNarration[state];
    setLogs((prev) => [
      ...prev,
      { ts: new Date().toLocaleTimeString("en-US", { hour12: false }), level: n.level, text: n.text },
    ]);
    if (job?.provider_id && state === "assigning") {
      setLogs((prev) => [
        ...prev,
        {
          ts: new Date().toLocaleTimeString("en-US", { hour12: false }),
          level: "info",
          text: `Assigned provider ${job.provider_id}.`,
        },
      ]);
    }
    if (isTerminal(state)) qc.invalidateQueries({ queryKey: ["balance"] });
  }, [job?.state, job?.provider_id, qc]);

  if (isLoading && !job) return <FullPageSpinner label="Loading session…" />;

  const state = job?.state ?? "pending";
  const running = !isTerminal(state);
  const rate = rental?.rateUsdcHour ?? 0;
  const startedAt = rental?.startedAt ?? job?.received_at ?? new Date().toISOString();

  async function stopRental() {
    if (!jobId) return;
    setStopping(true);
    try {
      await api.jobs.cancel(jobId);
      toast.success("Stopping the session. Settling final usage…");
      setConfirmStop(false);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Could not stop the session.");
    } finally {
      setStopping(false);
    }
  }

  function archive() {
    if (jobId) rentalsStore.remove(jobId);
    navigate("/rentals");
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <Link to="/rentals" className="inline-flex items-center gap-2 text-sm text-ink-300 hover:text-ink-50">
          <ArrowLeft className="size-4" /> Back to rentals
        </Link>
        <StatusBadge status={state} />
      </div>

      <div className="flex flex-col gap-2">
        <div className="flex items-center gap-3">
          {rental && <VendorMark vendor={rental.vendor} />}
          <h1 className="text-2xl font-bold text-ink-50">{rental?.name ?? "Rental session"}</h1>
        </div>
        <p className="text-sm text-ink-400">{rental?.gpuModel ?? job?.gpu_model ?? "GPU session"}</p>
      </div>

      <div className="grid gap-6 lg:grid-cols-5">
        <div className="space-y-6 lg:col-span-3">
          <CostMeter startedAt={startedAt} rateUsdcHour={rate} backendCostUsdc={0} running={running} />
          <LogStream lines={logs} />
        </div>

        <div className="space-y-6 lg:col-span-2">
          <Card>
            <CardHeader title="Session details" />
            <CardBody className="space-y-3 pt-1">
              <Detail icon={<Hash className="size-4" />} label="Job ID" value={shortAddress(jobId, 6, 6)} mono />
              <Detail
                icon={<Server className="size-4" />}
                label="Provider"
                value={job?.provider_id ? shortAddress(job.provider_id, 6, 6) : "Pending"}
                mono
              />
              <Detail
                icon={<MemoryStick className="size-4" />}
                label="VRAM"
                value={rental ? vram(rental.vramMb) : "-"}
              />
              <Detail icon={<Clock className="size-4" />} label="Started" value={timestamp(startedAt)} />
              {job?.attempts != null && (
                <Detail icon={<Hash className="size-4" />} label="Schedule attempts" value={String(job.attempts)} />
              )}
              {job?.last_error && (
                <div className="rounded-lg border border-critical/30 bg-critical/10 px-3 py-2 text-xs text-critical">
                  {job.last_error}
                </div>
              )}
            </CardBody>
          </Card>

          {running ? (
            <Button variant="danger" size="lg" className="w-full" onClick={() => setConfirmStop(true)}>
              <Square className="size-4" /> Stop rental & settle
            </Button>
          ) : (
            <Button variant="secondary" size="lg" className="w-full" onClick={archive}>
              Archive session
            </Button>
          )}
        </div>
      </div>

      <Modal
        open={confirmStop}
        onClose={() => setConfirmStop(false)}
        title="Stop this rental?"
        description="The GPU is released immediately and your escrow settles for the metered usage so far."
        footer={
          <>
            <Button variant="ghost" onClick={() => setConfirmStop(false)} disabled={stopping}>
              Keep running
            </Button>
            <Button variant="danger" onClick={stopRental} loading={stopping}>
              Stop & settle
            </Button>
          </>
        }
      >
        <p className="text-sm text-ink-300">
          Settlement is final and recorded on-chain. Any unused prepaid escrow returns to your available balance.
        </p>
      </Modal>
    </div>
  );
}

function Detail({
  icon,
  label,
  value,
  mono,
}: {
  icon: React.ReactNode;
  label: string;
  value: string;
  mono?: boolean;
}) {
  return (
    <div className="flex items-center justify-between gap-3">
      <span className="flex items-center gap-2 text-sm text-ink-300">
        <span className="text-ink-500">{icon}</span>
        {label}
      </span>
      <span className={mono ? "nums text-sm text-ink-100" : "text-sm text-ink-100"}>{value}</span>
    </div>
  );
}
