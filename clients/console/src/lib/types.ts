// Shared domain types for the DanteGPU console. These mirror the JSON the
// api-gateway proxies from the auth, gpu, billing and scheduler services. Where
// a backend field is optional or still evolving we keep it optional here so the
// UI degrades gracefully instead of throwing on a missing key.

export interface User {
  id: string;
  username: string;
  email?: string;
  role?: string;
  created_at?: string;
}

export interface AuthResponse {
  token: string;
  user?: User;
  // Some backends return the user fields flattened next to the token.
  id?: string;
  username?: string;
  email?: string;
}

export type GpuVendor = "nvidia" | "amd" | "apple" | "intel" | "unknown";

export interface GpuListing {
  id: string;
  provider_id: string;
  provider_name?: string;
  model_name: string;
  vendor: GpuVendor;
  vram_mb: number;
  power_consumption_w?: number;
  // Hourly price quoted in USDC. The backend may also expose a per-second rate.
  price_usdc_hour: number;
  price_usdc_second?: number;
  location?: string;
  status: "available" | "rented" | "offline";
  cuda_cores?: number;
  driver_version?: string;
  uptime_percent?: number;
}

export interface WalletBalance {
  // All balances are USDC unless explicitly the dGPU incentive token.
  usdc_available: number;
  usdc_reserved: number;
  dgpu_rewards?: number;
  wallet_id?: string;
  address?: string;
}

export interface Transaction {
  id: string;
  type: "deposit" | "withdraw" | "charge" | "reward" | "refund";
  amount_usdc: number;
  status: "pending" | "confirmed" | "failed";
  signature?: string;
  created_at: string;
  memo?: string;
}

export type JobState =
  | "pending"
  | "searching"
  | "assigning"
  | "dispatched"
  | "running"
  | "completed"
  | "failed"
  | "cancelled";

export interface Job {
  job_id: string;
  state: JobState;
  provider_id?: string;
  attempts?: number;
  last_error?: string;
  received_at?: string;
  updated_at?: string;
  // Enriched client-side from the original rental request.
  gpu_model?: string;
  name?: string;
}

export interface RentalSession {
  session_id: string;
  job_id: string;
  provider_id: string;
  gpu_model: string;
  state: JobState;
  started_at: string;
  // Live metering, refreshed by polling.
  elapsed_seconds: number;
  cost_usdc: number;
  rate_usdc_hour: number;
  vram_mb?: number;
  power_draw_w?: number;
  utilization_percent?: number;
}

export interface PricingRate {
  gpu_class: string;
  base_usdc_hour: number;
  vram_surcharge_usdc_gb_hour?: number;
  power_surcharge_usdc_kwh?: number;
}

export interface CostEstimate {
  estimated_usdc: number;
  rate_usdc_hour: number;
  duration_hours: number;
  breakdown?: Record<string, number>;
}

// Raw response from the gateway EstimateJobCost endpoint. `estimated_cost` is
// whatever the pricing engine returns (a number or a nested object), so callers
// normalize it through parseEstimatedCost.
export interface EstimateResponse {
  estimated_cost: number | Record<string, unknown> | string;
  currency?: string;
  breakdown?: Record<string, unknown>;
}

// Provider earnings as returned by the billing service. Field names vary across
// versions, so everything is optional and callers probe via pickNumber.
export interface ProviderEarnings {
  provider_id?: string;
  total_earned_usdc?: number;
  total_earnings?: number;
  pending_usdc?: number;
  pending_payout?: number;
  paid_out_usdc?: number;
  active_rentals?: number;
  total_sessions?: number;
  [k: string]: unknown;
}

// Shape posted to the gateway to start a rental / submit a job.
export interface SubmitJobPayload {
  type: string;
  name: string;
  gpu_type?: string;
  gpu_count?: number;
  min_vram_mb?: number;
  min_power_w?: number;
  priority?: number;
  params: Record<string, unknown>;
}
