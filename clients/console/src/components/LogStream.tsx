import { useEffect, useRef } from "react";
import { Terminal } from "lucide-react";
import { cn } from "@/lib/cn";

export interface LogLine {
  ts: string;
  level: "info" | "warn" | "error" | "system";
  text: string;
}

const levelColor: Record<LogLine["level"], string> = {
  info: "text-ink-200",
  warn: "text-caution",
  error: "text-critical",
  system: "text-flux-300",
};

// A terminal-style, auto-scrolling log viewer for a rental session. Lines are
// fed from the parent (which polls the job/session); this component only owns
// rendering and stick-to-bottom behavior.
export function LogStream({ lines, title = "Session logs" }: { lines: LogLine[]; title?: string }) {
  const endRef = useRef<HTMLDivElement>(null);
  const atBottom = useRef(true);

  useEffect(() => {
    if (atBottom.current) endRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [lines]);

  return (
    <div className="card overflow-hidden">
      <div className="flex items-center gap-2 border-b border-ink-700 px-4 py-2.5">
        <Terminal className="size-4 text-ink-400" />
        <span className="text-sm font-medium text-ink-100">{title}</span>
        <span className="ml-auto flex gap-1.5">
          <span className="size-2.5 rounded-full bg-critical/60" />
          <span className="size-2.5 rounded-full bg-caution/60" />
          <span className="size-2.5 rounded-full bg-positive/60" />
        </span>
      </div>
      <div
        className="h-72 overflow-y-auto bg-ink-950/60 p-4 font-mono text-xs leading-relaxed"
        onScroll={(e) => {
          const el = e.currentTarget;
          atBottom.current = el.scrollHeight - el.scrollTop - el.clientHeight < 40;
        }}
      >
        {lines.length === 0 ? (
          <p className="text-ink-500">Waiting for log output…</p>
        ) : (
          lines.map((l, i) => (
            <div key={i} className="flex gap-3">
              <span className="shrink-0 text-ink-600">{l.ts}</span>
              <span className={cn("whitespace-pre-wrap break-all", levelColor[l.level])}>{l.text}</span>
            </div>
          ))
        )}
        <div ref={endRef} />
      </div>
    </div>
  );
}
