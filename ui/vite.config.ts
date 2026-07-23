import path from 'node:path';
import react from '@vitejs/plugin-react';
import { visualizer } from 'rollup-plugin-visualizer';
import { defineConfig, loadEnv, type PluginOption } from 'vite';

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '');
  const proxyTarget = env.VITE_DEV_API_PROXY || 'https://localhost:8445';
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
        '@locales': path.resolve(__dirname, '../internal/i18n/locales'),
      },
      dedupe: [
        'react',
        'react-dom',
        'react-router-dom',
        'lucide-react',
        'i18next',
        'react-i18next',
      ],
    },
    // FIX #180: Code splitting for optimal bundle sizes
    build: {
      // Output directly to embed directory - no syncing needed.
      // outDir is outside Vite's project root; Vite defaults emptyOutDir to
      // false there, which preserves the tracked .gitkeep between builds (the
      // canonical seed/stem pattern). Do NOT set emptyOutDir: true — it wipes
      // .gitkeep and previously needed a workaround plugin to recreate it.
      outDir: '../internal/api/ui',
      // Vite 7: Target modern browsers for smaller bundles
      target: 'es2022',
      // Enable CSS code splitting
      cssCodeSplit: true,
      // CodeMirror is the largest intentional lazy vendor chunk. Keep
      // this below the app shell so an accidental monolithic bundle
      // still trips CI/build output warnings.
      chunkSizeWarningLimit: 350,
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
            if (
              id.includes('/node_modules/i18next/') ||
              id.includes('/node_modules/react-i18next/') ||
              id.includes('/node_modules/i18next-browser-languagedetector/')
            )
              return 'vendor-i18n';
            if (id.includes('/node_modules/zustand/') || id.includes('/node_modules/immer/'))
              return 'vendor-state';
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
          secure: false,
        },
      },
    },
  };
});
