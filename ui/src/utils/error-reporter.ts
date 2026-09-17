/**
 * Client-Side Error Reporter
 *
 * Provides structured error reporting for the frontend application.
 * Batches errors to avoid flooding and logs to console in development.
 *
 * FIX #190: No client-side error logging
 */

import { sendBackgroundReport } from '../api/requestCore';

interface ErrorReport {
  message: string;
  stack?: string;
  context: string;
  timestamp: number;
  url: string;
}

const BATCH_INTERVAL_MS = 5000;
const MAX_BATCH_SIZE = 10;

const batch: ErrorReport[] = [];
let timer: ReturnType<typeof setTimeout> | null = null;

function flush(): void {
  if (batch.length === 0) return;

  const reports = batch.splice(0, MAX_BATCH_SIZE);

  if (import.meta.env.DEV) {
    for (const report of reports) {
      console.error(`[ErrorReporter] ${report.context}:`, report.message, report.stack ?? '');
    }
    return;
  }

  // In production, send to the client telemetry endpoint. The route is
  // authenticated and CSRF-protected, so the report goes through the shared
  // request layer's header builder rather than a bare fetch -- without it
  // every report was rejected with a 401 nobody saw.
  void sendBackgroundReport('/api/v1/client-errors', { errors: reports });
}

function scheduleFlush(): void {
  if (timer !== null) return;
  timer = setTimeout(() => {
    timer = null;
    flush();
  }, BATCH_INTERVAL_MS);
}

export function reportError(error: unknown, context = 'unknown'): void {
  const report: ErrorReport = {
    message: error instanceof Error ? error.message : String(error),
    stack: error instanceof Error ? error.stack : undefined,
    context,
    timestamp: Date.now(),
    url: globalThis.location?.href ?? '',
  };

  batch.push(report);

  if (batch.length >= MAX_BATCH_SIZE) {
    flush();
  } else {
    scheduleFlush();
  }
}
