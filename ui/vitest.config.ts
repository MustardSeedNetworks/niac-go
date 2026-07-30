/**
 * Vitest Configuration
 *
 * Configures the Vitest test framework for the NIAC frontend.
 *
 * Configuration:
 * - Globals: Enable global test functions (describe, it, expect)
 * - Environment: jsdom - Simulates browser DOM for React component testing
 * - Setup files: Loads test/setup.ts for global mocks
 * - Coverage: V8 provider with multiple report formats
 *
 * Usage:
 * ```bash
 * npm test              # Run all tests
 * npm test -- --watch   # Run with file watching
 * npm test -- --coverage  # Generate coverage reports
 * ```
 */

import { fileURLToPath, URL } from 'node:url';
import react from '@vitejs/plugin-react';
import { defineConfig } from 'vitest/config';

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
      '@locales': fileURLToPath(new URL('../internal/i18n/locales', import.meta.url)),
    },
  },
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts'],
    include: ['src/**/*.{test,spec}.{ts,tsx}'],
    maxWorkers: 4,
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html', 'lcov'],
      exclude: ['node_modules/', 'src/test/', '**/*.d.ts', '**/*.config.*', 'dist/'],
      // Anti-regression floor (set ~2pp below current measurement).
      // Above CLAUDE.md's 50% minimum on lines/branches/statements;
      // functions lagging. Current: lines 77, branches 69, functions
      // 48, stmts 70.
      thresholds: {
        lines: 75,
        branches: 67,
        functions: 45,
        statements: 67,
      },
    },
  },
});
