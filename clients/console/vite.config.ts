import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import path from "node:path";

// The console talks to the dante api-gateway. In dev we proxy /api to the local
// gateway so the browser never hits CORS; in prod the gateway origin is baked in
// via VITE_API_BASE_URL. Solana web3.js pulls in a few Node globals (Buffer),
// so we alias them to browser shims that the wallet adapters expect.
export default defineConfig(({ mode }) => {
  const gateway = process.env.VITE_API_BASE_URL || "http://localhost:8080";
  return {
    plugins: [react(), tailwindcss()],
    resolve: {
      alias: {
        "@": path.resolve(__dirname, "src"),
      },
    },
    define: {
      // Some Solana deps reference process.env; keep them happy in the browser.
      "process.env": {},
      global: "globalThis",
    },
    server: {
      port: 5273,
      proxy:
        mode === "development"
          ? {
              "/api": {
                target: gateway,
                changeOrigin: true,
              },
            }
          : undefined,
    },
    build: {
      target: "es2022",
      sourcemap: mode !== "production",
      chunkSizeWarningLimit: 1200,
    },
  };
});
