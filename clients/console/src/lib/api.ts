// Typed client for the dante api-gateway. Every backend call funnels through
// `request()` so auth, JSON handling and error surfacing stay uniform. In dev
// the base is empty and vite proxies /api -> gateway; in prod VITE_API_BASE_URL
// points at the gateway origin.

import type {
  AuthResponse,
  EstimateResponse,
  GpuListing,
  Job,
  PricingRate,
  ProviderEarnings,
  SubmitJobPayload,
  Transaction,
  User,
  WalletBalance,
} from "./types";

const API_BASE = (import.meta.env.VITE_API_BASE_URL as string | undefined)?.replace(/\/$/, "") || "";
const TOKEN_KEY = "dante.token";

// Derives the gateway WebSocket URL for the live log tail. In prod API_BASE is
// the gateway origin; in dev set VITE_API_BASE_URL so this resolves to the
// gateway rather than the vite dev origin.
export function gatewayWsUrl(path: string): string {
  const base = API_BASE || window.location.origin;
  return `${base.replace(/^http/, "ws")}${path}`;
}

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY);
}

export function setToken(token: string | null): void {
  if (token) localStorage.setItem(TOKEN_KEY, token);
  else localStorage.removeItem(TOKEN_KEY);
}

export class ApiError extends Error {
  status: number;
  body: unknown;
  constructor(status: number, message: string, body?: unknown) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.body = body;
  }
}

interface RequestOptions {
  method?: "GET" | "POST" | "PUT" | "DELETE";
  body?: unknown;
  // When true, a 401 will NOT auto-clear the token (used by the login call).
  noAuthRedirect?: boolean;
  signal?: AbortSignal;
}

async function request<T>(path: string, opts: RequestOptions = {}): Promise<T> {
  const headers: Record<string, string> = { Accept: "application/json" };
  const token = getToken();
  if (token) headers.Authorization = `Bearer ${token}`;

  let body: BodyInit | undefined;
  if (opts.body !== undefined) {
    headers["Content-Type"] = "application/json";
    body = JSON.stringify(opts.body);
  }

  const res = await fetch(`${API_BASE}/api/v1${path}`, {
    method: opts.method ?? "GET",
    headers,
    body,
    signal: opts.signal,
  });

  // 204 / empty bodies are valid for some mutations.
  const text = await res.text();
  const parsed = text ? safeJson(text) : null;

  if (!res.ok) {
    if (res.status === 401 && !opts.noAuthRedirect) {
      setToken(null);
    }
    throw new ApiError(res.status, extractMessage(parsed, text, res.status), parsed);
  }

  return parsed as T;
}

function safeJson(text: string): unknown {
  try {
    return JSON.parse(text);
  } catch {
    return text;
  }
}

// Pulls the best human message out of an error body, preferring an `error` or
// `message` field, then the raw text, then a generic fallback.
function extractMessage(parsed: unknown, text: string, status: number): string {
  if (parsed && typeof parsed === "object") {
    const obj = parsed as Record<string, unknown>;
    if (typeof obj.error === "string") return obj.error;
    if (typeof obj.message === "string") return obj.message;
  }
  if (text) return text;
  return `Request failed (${status})`;
}

function num(v: unknown): number {
  const n = typeof v === "number" ? v : typeof v === "string" ? Number.parseFloat(v) : NaN;
  return Number.isFinite(n) ? n : 0;
}

// The billing service returns a BalanceResponse ({ wallet_id, available_balance,
// locked_balance, ... }) or a Wallet ({ id, balance, locked_balance, ... }).
// Map either onto the console's WalletBalance shape, tolerating the usdc_* names
// too in case the backend evolves.
function normalizeBalance(raw: unknown): WalletBalance {
  const o = (raw && typeof raw === "object" ? raw : {}) as Record<string, unknown>;
  return {
    usdc_available: num(o.usdc_available ?? o.available_balance ?? o.balance),
    usdc_reserved: num(o.usdc_reserved ?? o.locked_balance ?? o.pending_balance),
    dgpu_rewards: o.dgpu_rewards != null ? num(o.dgpu_rewards) : undefined,
    wallet_id: String(o.wallet_id ?? o.id ?? ""),
    address: typeof o.solana_address === "string" ? o.solana_address : undefined,
  };
}

// Coerces a loosely-typed backend listing into a fully-populated GpuListing so
// every downstream consumer (sorts, search, cards) is safe even when the backend
// omits or nulls a field. Missing numeric fields become 0 (never NaN/undefined),
// and a missing model name becomes a clear placeholder instead of crashing.
function normalizeListing(g: Partial<GpuListing>): GpuListing {
  const vendor = (g.vendor ?? "unknown") as GpuListing["vendor"];
  const status = (g.status ?? "available") as GpuListing["status"];
  return {
    id: String(g.id ?? g.provider_id ?? crypto.randomUUID()),
    provider_id: String(g.provider_id ?? ""),
    provider_name: g.provider_name,
    model_name: g.model_name && String(g.model_name).trim() ? String(g.model_name) : "Unknown GPU",
    vendor: ["nvidia", "amd", "apple", "intel", "unknown"].includes(vendor) ? vendor : "unknown",
    vram_mb: num(g.vram_mb),
    power_consumption_w: g.power_consumption_w != null ? num(g.power_consumption_w) : undefined,
    price_usdc_hour: num(g.price_usdc_hour),
    price_usdc_second: g.price_usdc_second != null ? num(g.price_usdc_second) : undefined,
    location: g.location,
    status: ["available", "rented", "offline"].includes(status) ? status : "offline",
    cuda_cores: g.cuda_cores,
    driver_version: g.driver_version,
    uptime_percent: g.uptime_percent != null ? num(g.uptime_percent) : undefined,
  };
}

// Some list endpoints wrap their payload as { data: [...] } or { items: [...] }.
function unwrapList<T>(raw: unknown, ...keys: string[]): T[] {
  if (Array.isArray(raw)) return raw as T[];
  if (raw && typeof raw === "object") {
    for (const k of keys) {
      const v = (raw as Record<string, unknown>)[k];
      if (Array.isArray(v)) return v as T[];
    }
  }
  return [];
}

export const api = {
  auth: {
    login: (username: string, password: string) =>
      request<AuthResponse>("/auth/login", {
        method: "POST",
        body: { username, password },
        noAuthRedirect: true,
      }),
    register: (username: string, email: string, password: string) =>
      request<AuthResponse>("/auth/register", {
        method: "POST",
        body: { username, email, password },
        noAuthRedirect: true,
      }),
    me: () => request<User>("/auth/me"),
  },

  marketplace: {
    list: async (signal?: AbortSignal): Promise<GpuListing[]> => {
      const raw = await request<unknown>("/billing/marketplace", { signal });
      return unwrapList<Partial<GpuListing>>(raw, "gpus", "listings", "data", "items").map(normalizeListing);
    },
  },

  wallet: {
    balance: async (userId: string): Promise<WalletBalance> =>
      normalizeBalance(await request<unknown>(`/billing/user/${userId}/balance`)),
    get: async (userId: string): Promise<WalletBalance> =>
      normalizeBalance(await request<unknown>(`/billing/user/${userId}/wallet`)),
    create: () => request<WalletBalance>("/billing/wallet", { method: "POST", body: {} }),
    deposit: (walletId: string, amountUsdc: number, signature?: string) =>
      request<Transaction>(`/billing/wallet/${walletId}/deposit`, {
        method: "POST",
        // The billing service decodes the on-chain proof under `solana_signature`;
        // sending `signature` would be silently dropped and break reconciliation.
        body: { amount: amountUsdc, solana_signature: signature },
      }),
    withdraw: (walletId: string, amountUsdc: number, toAddress: string) =>
      request<Transaction>(`/billing/wallet/${walletId}/withdraw`, {
        method: "POST",
        body: { wallet_id: walletId, amount: amountUsdc, to_address: toAddress },
      }),
    transactions: async (walletId: string): Promise<Transaction[]> => {
      const raw = await request<unknown>(`/billing/wallet/${walletId}/transactions`);
      return unwrapList<Transaction>(raw, "transactions", "data", "items");
    },
  },

  pricing: {
    rates: async (): Promise<PricingRate[]> => {
      const raw = await request<unknown>("/billing/pricing/rates");
      return unwrapList<PricingRate>(raw, "rates", "data", "items");
    },
    estimate: (payload: {
      gpu_model: string;
      vram_required_mb?: number;
      estimated_hours: number;
      estimated_power_w?: number;
    }) => request<EstimateResponse>("/billing/pricing/estimate", { method: "POST", body: payload }),
  },

  provider: {
    earnings: (providerId: string) => request<ProviderEarnings>(`/billing/provider/${providerId}/earnings`),
    payout: (providerId: string, amountUsdc?: number) =>
      request<Transaction>(`/billing/provider/${providerId}/payout`, {
        method: "POST",
        body: amountUsdc != null ? { amount: amountUsdc } : {},
      }),
  },

  jobs: {
    submit: (payload: SubmitJobPayload) =>
      request<{ job_id: string; status: string }>("/jobs", { method: "POST", body: payload }),
    status: (jobId: string, signal?: AbortSignal) => request<Job>(`/jobs/${jobId}`, { signal }),
    cancel: (jobId: string) => request<{ status: string }>(`/jobs/${jobId}`, { method: "DELETE" }),
  },
};
