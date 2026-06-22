import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";
import type { Job, JobState } from "@/lib/types";

const TERMINAL: JobState[] = ["completed", "failed", "cancelled"];

// Polls a single job's scheduler state. Polling stops once the job reaches a
// terminal state so we don't keep hitting the gateway for finished work.
export function useJob(jobId: string | undefined) {
  return useQuery<Job>({
    queryKey: ["job", jobId],
    enabled: !!jobId,
    queryFn: ({ signal }) => api.jobs.status(jobId!, signal),
    refetchInterval: (query) => {
      const state = query.state.data?.state;
      if (state && TERMINAL.includes(state)) return false;
      return 4000;
    },
  });
}

export function isTerminal(state: JobState | undefined): boolean {
  return !!state && TERMINAL.includes(state);
}
