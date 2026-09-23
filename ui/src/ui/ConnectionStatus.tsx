import type { FC } from 'react';
import type { ConnectionState } from '../hooks/useConnectionStatus';
import { Tooltip } from './Tooltip';

const STATUS_STYLES: Record<ConnectionState, string> = {
  connected: 'bg-status-success',
  disconnected: 'bg-status-error',
  checking: 'bg-status-warning animate-pulse',
};

const STATUS_LABELS: Record<ConnectionState, string> = {
  connected: 'Connected to backend',
  disconnected: 'Backend unreachable',
  checking: 'Checking connection...',
};

export const ConnectionStatus: FC<{ status: ConnectionState; compact?: boolean }> = ({
  status,
  compact = false,
}) => {
  return (
    <Tooltip text={STATUS_LABELS[status]}>
      <button
        type="button"
        data-testid="connection-status"
        aria-label={STATUS_LABELS[status]}
        className={`flex min-h-11 min-w-11 items-center gap-compact rounded focus-visible:outline-2 focus-visible:outline-brand-accent ${compact ? 'w-full justify-center' : ''}`}
      >
        <span role="status" className="flex items-center gap-compact">
          <span className={`h-2 w-2 rounded-full ${STATUS_STYLES[status]}`} />
          <span className={compact ? 'sr-only' : 'text-xs text-text-muted'}>
            {status === 'connected' ? 'Online' : status === 'disconnected' ? 'Offline' : '...'}
          </span>
        </span>
      </button>
    </Tooltip>
  );
};
