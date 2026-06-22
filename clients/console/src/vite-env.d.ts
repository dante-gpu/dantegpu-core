/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_API_BASE_URL?: string;
  readonly VITE_SOLANA_CLUSTER?: string;
  readonly VITE_SOLANA_RPC_URL?: string;
  readonly VITE_USDC_MINT?: string;
  readonly VITE_USDC_DECIMALS?: string;
  readonly VITE_PLATFORM_DEPOSIT_ADDRESS?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}

interface Window {
  Buffer: typeof import("buffer").Buffer;
}
