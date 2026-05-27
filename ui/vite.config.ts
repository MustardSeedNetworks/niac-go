import path from 'node:path';
import react from '@vitejs/plugin-react';
import { visualizer } from 'rollup-plugin-visualizer';
import { defineConfig, loadEnv, type PluginOption } from 'vite';

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '');
  const proxyTarget = env.VITE_DEV_API_PROXY || 'http://localhost:8080';
  const analyze = env.ANALYZE === 'true';

  return {
    plugins: [
      react(),
      // FIX #181: Bundle analysis when ANALYZE=true
      analyze &&
        (visualizer({
          open: true,
          filename: 'dist/bundle-stats.html',
          gzipSize: true,
        }) as PluginOption),
    ],
    resolve: {
      alias: {
        '@': path.resolve(__dirname, 'src'),
        // Cross-product canonical alias for shared Go-embedded locale files.
        // Mirrors seed + stem so the same import path resolves identically:
        //   import enCommon from '@locales/en/common.json';
        // The Go binary is the single source of truth for translation
        // content (embedded via //go:embed in internal/i18n/).
        '@locales': path.resolve(__dirname, '../internal/i18n/locales'),
      },
      dedupe: ['react', 'react-dom', 'react-router-dom', 'lucide-react'],
    },
    // FIX #180: Code splitting for optimal bundle sizes
    build: {
      // Output directly to embed directory - no syncing needed.
      // emptyOutDir intentionally omitted: outDir is outside Vite's
      // project root, so Vite defaults to false and preserves the
      // tracked `.gitkeep` placeholder. CLAUDE.md mandate.
      outDir: '../internal/api/ui',
      // Vite 7: Target modern browsers for smaller bundles
      target: 'es2022',
      // Enable CSS code splitting
      cssCodeSplit: true,
      // Smaller chunk size warning threshold
      chunkSizeWarningLimit: 300,
      // Never inline assets as data: URLs (Vite default is 4096 bytes). Required
      // because @fontsource-variable ships small metric-override shim fonts that
      // would otherwise be inlined and violate the production `font-src 'self'`
      // CSP. With this set to 0, every asset bundles as a file under /assets/,
      // served from same-origin and properly HTTP-cacheable.
      assetsInlineLimit: 0,
      // Module preload optimization
      modulePreload: {
        polyfill: false, // Modern browsers don't need polyfill
      },
      rollupOptions: {
        output: {
          manualChunks: (id: string) => {
            if (
              id.includes('/node_modules/react/') ||
              id.includes('/node_modules/react-dom/') ||
              id.includes('/node_modules/react-router-dom/') ||
              id.includes('/node_modules/scheduler/')
            )
              return 'vendor-react';
            if (id.includes('/node_modules/@tanstack/react-query/')) return 'vendor-query';
            if (
              id.includes('/node_modules/@codemirror/') ||
              id.includes('/node_modules/@lezer/highlight/')
            )
              return 'vendor-codemirror';
            if (id.includes('/node_modules/@xyflow/react/')) return 'vendor-xyflow';
            if (
              id.includes('/node_modules/lucide-react/') ||
              id.includes('/node_modules/tailwind-merge/')
            )
              return 'vendor-ui';
            return undefined;
          },
        },
      },
    },
    server: {
      proxy: {
        '/api': {
          target: proxyTarget,
          changeOrigin: true,
        },
      },
    },
  };
});
