import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { nodePolyfills } from 'vite-plugin-node-polyfills'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [
    react(),
    nodePolyfills({
      // Include specific polyfills
      include: ['buffer', 'process', 'util'],
      // Whether to polyfill `node:` protocol imports.
      protocolImports: true,
      // Override the default polyfills
      overrides: {
        // Since `fs` is not supported in browsers, we can use the `memfs` package to polyfill it.
        fs: 'memfs',
      },
      // Exclude specific polyfills. 
      exclude: [],
      // Whether to polyfill specific globals.
      globals: {
        Buffer: true,
        global: true,
        process: true,
      },
    }),
  ],
  define: {
    global: 'globalThis',
    'process.env': {},
  },
  resolve: {
    alias: {
      buffer: 'buffer',
      process: 'process/browser',
      util: 'util',
    }
  },
  optimizeDeps: {
    include: ['buffer', 'process'],
  },
  server: {
    port: 5173,
    host: true,
  },
  build: {
    rollupOptions: {
      plugins: [],
    },
  },
}) 