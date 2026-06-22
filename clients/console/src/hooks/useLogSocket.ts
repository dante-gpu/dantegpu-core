import { useEffect, useRef, useState } from "react";
import { gatewayWsUrl } from "@/lib/api";
import type { LogLine } from "@/components/LogStream";

export type SocketStatus = "connecting" | "open" | "closed";

// Loki tail frame: { streams: [ { stream: {labels}, values: [ [nsTs, line], ... ] } ] }
interface LokiTailFrame {
  streams?: Array<{ stream?: Record<string, string>; values?: [string, string][] }>;
}

function levelFor(line: string): LogLine["level"] {
  const l = line.toLowerCase();
  if (l.includes("error") || l.includes("panic") || l.includes("fatal")) return "error";
  if (l.includes("warn")) return "warn";
  return "info";
}

function tsFrom(ns: string): string {
  const ms = Number(ns) / 1e6;
  const d = Number.isFinite(ms) ? new Date(ms) : new Date();
  return d.toLocaleTimeString("en-US", { hour12: false });
}

// Connects to the gateway's WebSocket log tail (which proxies Loki), parses the
// frames into LogLine[], keeps a bounded rolling buffer, and transparently
// reconnects with backoff. Returns the lines plus a live connection status so
// the UI can show a clear disconnected state instead of a silent empty list.
export function useLogSocket(enabled: boolean, maxLines = 300) {
  const [lines, setLines] = useState<LogLine[]>([]);
  const [status, setStatus] = useState<SocketStatus>("closed");
  const wsRef = useRef<WebSocket | null>(null);
  const retryRef = useRef<number>(0);
  const timerRef = useRef<number | null>(null);
  const closedByUs = useRef(false);

  useEffect(() => {
    if (!enabled) return;
    closedByUs.current = false;

    function append(batch: LogLine[]) {
      if (batch.length === 0) return;
      setLines((prev) => [...prev, ...batch].slice(-maxLines));
    }

    function connect() {
      setStatus("connecting");
      let ws: WebSocket;
      try {
        ws = new WebSocket(gatewayWsUrl("/logs/stream"));
      } catch {
        scheduleRetry();
        return;
      }
      wsRef.current = ws;

      ws.onopen = () => {
        retryRef.current = 0;
        setStatus("open");
      };

      ws.onmessage = (ev) => {
        const text = typeof ev.data === "string" ? ev.data : "";
        if (!text) return;
        try {
          const frame = JSON.parse(text) as LokiTailFrame;
          const batch: LogLine[] = [];
          for (const s of frame.streams ?? []) {
            const container = s.stream?.container || s.stream?.service_name || s.stream?.job;
            for (const [ns, raw] of s.values ?? []) {
              const text = container ? `[${container}] ${raw}` : raw;
              batch.push({ ts: tsFrom(ns), level: levelFor(raw), text });
            }
          }
          append(batch);
        } catch {
          // Non-JSON frame (e.g. an error notice from the gateway): show as-is.
          append([{ ts: new Date().toLocaleTimeString("en-US", { hour12: false }), level: "system", text }]);
        }
      };

      ws.onerror = () => ws.close();
      ws.onclose = () => {
        setStatus("closed");
        if (!closedByUs.current) scheduleRetry();
      };
    }

    function scheduleRetry() {
      const delay = Math.min(1000 * 2 ** retryRef.current, 15_000);
      retryRef.current += 1;
      timerRef.current = window.setTimeout(connect, delay);
    }

    connect();

    return () => {
      closedByUs.current = true;
      if (timerRef.current) window.clearTimeout(timerRef.current);
      wsRef.current?.close();
    };
  }, [enabled, maxLines]);

  return { lines, status, clear: () => setLines([]) };
}
