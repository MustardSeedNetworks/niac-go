import type { FC } from 'react';
import { useTranslation } from 'react-i18next';
import type { ConnectionState } from '../hooks/useConnectionStatus';
import { Tooltip } from './Tooltip';

const STATUS_STYLES: Record<ConnectionState, string> = {
  connected: 'bg-status-success',
  disconnected: 'bg-status-error',
  checking: 'bg-status-warning animate-pulse',
};

export const ConnectionStatus: FC<{ status: ConnectionState; compact?: boolean }> = ({
  status,
  compact = false,
}) => {
  const { t } = useTranslation('common');
  const labels: Record<ConnectionState, string> = {
    connected: t('connection.connected'),
    disconnected: t('connection.disconnected'),
    checking: t('connection.checking'),
  };
  const label = labels[status];
  return (
    <Tooltip text={label}>
      <button
        type="button"
        data-testid="connection-status"
        aria-label={label}
        className={`flex min-h-11 min-w-11 items-center gap-compact rounded focus-visible:outline-2 focus-visible:outline-brand-accent ${compact ? 'w-full justify-center' : ''}`}
      >
        <span role="status" className="flex items-center gap-compact">
          <span className={`h-2 w-2 rounded-full ${STATUS_STYLES[status]}`} />
          <span className={compact ? 'sr-only' : 'text-xs text-text-muted'}>
            {status === 'connected'
              ? t('connection.online')
              : status === 'disconnected'
                ? t('connection.offline')
                : '...'}
          </span>
        </span>
      </button>
    </Tooltip>
  );
};
