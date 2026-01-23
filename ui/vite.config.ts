import path from "node:path";
import react from "@vitejs/plugin-react";
import { visualizer } from "rollup-plugin-visualizer";
import { type PluginOption, defineConfig, loadEnv } from "vite";

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), "");
  const proxyTarget = env.VITE_DEV_API_PROXY || "http://localhost:8080";
  const analyze = env.ANALYZE === "true";

  return {
    plugins: [
      react(),
      // FIX #181: Bundle analysis when ANALYZE=true
      analyze &&
        (visualizer({
          open: true,
          filename: "dist/bundle-stats.html",
          gzipSize: true,
        }) as PluginOption),
    ],
    resolve: {
      alias: {
        "@": path.resolve(__dirname, "src"),
      },
      dedupe: ["react", "react-dom", "react-router-dom", "lucide-react"],
    },
    // FIX #180: Code splitting for optimal bundle sizes
    build: {
      rollupOptions: {
        output: {
          manualChunks: {
            "vendor-react": ["react", "react-dom", "react-router-dom"],
            "vendor-query": ["@tanstack/react-query"],
            "vendor-codemirror": [
              "@codemirror/commands",
              "@codemirror/lang-yaml",
              "@codemirror/language",
              "@codemirror/state",
              "@codemirror/view",
              "@lezer/highlight",
            ],
            "vendor-xyflow": ["@xyflow/react"],
            "vendor-ui": ["lucide-react", "tailwind-merge"],
          },
        },
      },
    },
    server: {
      proxy: {
        "/api": {
          target: proxyTarget,
          changeOrigin: true,
        },
      },
    },
  };
});
