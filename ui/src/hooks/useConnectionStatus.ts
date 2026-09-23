import { useEffect, useState } from 'react';
import { request } from '../api/requestCore';

export type ConnectionState = 'connected' | 'disconnected' | 'checking';

const CHECK_INTERVAL_MS = 15_000;
const TIMEOUT_MS = 5_000;

export function useConnectionStatus(): ConnectionState {
  const [status, setStatus] = useState<ConnectionState>('checking');
  useEffect(() => {
    let disposed = false;
    let controller: AbortController;
    let timeout: ReturnType<typeof setTimeout>;
    const check = async () => {
      controller = new AbortController();
      timeout = setTimeout(() => controller.abort(), TIMEOUT_MS);
      try {
        await request('/api/v1/version', { signal: controller.signal });
        if (!disposed) setStatus('connected');
      } catch {
        if (!disposed) setStatus('disconnected');
      } finally {
        clearTimeout(timeout);
      }
    };
    void check();
    const interval = setInterval(check, CHECK_INTERVAL_MS);
    return () => {
      disposed = true;
      clearInterval(interval);
      clearTimeout(timeout);
      controller.abort();
    };
  }, []);
  return status;
}
