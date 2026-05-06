import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import path from "path";

// Allow overriding the proxy target via env. Useful when api-gateway is
// not running and you want to point dev server at a single service or a
// remote staging environment, e.g.
//   VITE_DEV_PROXY_TARGET=http://localhost:8080 npm run dev
const PROXY_TARGET = process.env.VITE_DEV_PROXY_TARGET || "http://localhost:8000";

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  server: {
    port: 3000,
    host: true,
    proxy: {
      "/api": {
        target: PROXY_TARGET,
        changeOrigin: true,
        ws: true,
        // Don't crash the dev server when upstream is down — surface a
        // clean 503 to the SPA so the UI can show a "backend offline"
        // banner instead of a generic ECONNREFUSED stack.
        configure: (proxy) => {
          proxy.on("error", (err, _req, res) => {
            // res may be undefined for ws upgrades
            if (res && "writeHead" in res && !res.headersSent) {
              try {
                res.writeHead(503, { "Content-Type": "application/json" });
                res.end(
                  JSON.stringify({
                    error: "backend_unreachable",
                    detail: `Cannot reach ${PROXY_TARGET}: ${err.message}`,
                  }),
                );
              } catch {
                // ignore — the connection is already broken
              }
            }
          });
        },
      },
    },
  },
  build: {
    outDir: "dist",
    sourcemap: false,
    rollupOptions: {
      output: {
        manualChunks: {
          vendor: ["react", "react-dom", "react-router-dom"],
        },
      },
    },
  },
});
