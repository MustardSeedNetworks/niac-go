import { type FC, useCallback, useEffect, useRef, useState } from 'react';

type Status = 'connected' | 'disconnected' | 'checking';

const STATUS_STYLES: Record<Status, string> = {
  connected: 'bg-status-success',
  disconnected: 'bg-status-error',
  checking: 'bg-status-warning animate-pulse',
};

const STATUS_LABELS: Record<Status, string> = {
  connected: 'Connected to backend',
  disconnected: 'Backend unreachable',
  checking: 'Checking connection...',
};

const CHECK_INTERVAL_MS = 15_000;
const TIMEOUT_MS = 5_000;

export const ConnectionStatus: FC = () => {
  const [status, setStatus] = useState<Status>('checking');
  const intervalRef = useRef<ReturnType<typeof setInterval>>(null);

  const checkConnection = useCallback(async () => {
    try {
      const controller = new AbortController();
      const timeout = setTimeout(() => controller.abort(), TIMEOUT_MS);
      const response = await fetch('/api/v1/version', { signal: controller.signal });
      clearTimeout(timeout);
      setStatus(response.ok ? 'connected' : 'disconnected');
    } catch {
      setStatus('disconnected');
    }
  }, []);

  useEffect(() => {
    checkConnection();
    intervalRef.current = setInterval(checkConnection, CHECK_INTERVAL_MS);
    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
      }
    };
  }, [checkConnection]);

  return (
    <div className="flex items-center gap-2" title={STATUS_LABELS[status]}>
      <span className={`h-2 w-2 rounded-full ${STATUS_STYLES[status]}`} />
      <span className="text-xs text-text-muted">
        {status === 'connected' ? 'Online' : status === 'disconnected' ? 'Offline' : '...'}
      </span>
    </div>
  );
};
