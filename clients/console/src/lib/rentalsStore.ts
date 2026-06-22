// Local record of rentals this browser has started. The gateway exposes job
// status by id but not a "list my jobs" endpoint, so we persist the rentals we
// launch here and poll each one. This is the source of truth for the My Rentals
// list and the dashboard's active-rental count.

import type { GpuVendor } from "./types";

export interface LocalRental {
  jobId: string;
  name: string;
  gpuModel: string;
  vendor: GpuVendor;
  rateUsdcHour: number;
  vramMb: number;
  providerId?: string;
  startedAt: string;
}

const KEY = "dante.rentals";
const listeners = new Set<() => void>();
let cache: LocalRental[] | null = null;

function read(): LocalRental[] {
  if (cache) return cache;
  try {
    const raw = localStorage.getItem(KEY);
    cache = raw ? (JSON.parse(raw) as LocalRental[]) : [];
  } catch {
    cache = [];
  }
  return cache;
}

function write(next: LocalRental[]) {
  cache = next;
  localStorage.setItem(KEY, JSON.stringify(next));
  listeners.forEach((l) => l());
}

export const rentalsStore = {
  getSnapshot(): LocalRental[] {
    return read();
  },
  subscribe(listener: () => void): () => void {
    listeners.add(listener);
    return () => listeners.delete(listener);
  },
  add(rental: LocalRental) {
    const existing = read().filter((r) => r.jobId !== rental.jobId);
    write([rental, ...existing]);
  },
  remove(jobId: string) {
    write(read().filter((r) => r.jobId !== jobId));
  },
  get(jobId: string): LocalRental | undefined {
    return read().find((r) => r.jobId === jobId);
  },
};
