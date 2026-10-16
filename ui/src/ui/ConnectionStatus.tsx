import { useTranslation } from 'react-i18next';
import type { ConnectionState } from '../hooks/useConnectionStatus';
import { accessibleNameIncluding } from '../lib/accessibleName';
import { Tooltip } from './Tooltip';

const STATUS_STYLES: Record<ConnectionState, string> = {
  connected: 'bg-status-success',
  disconnected: 'bg-status-error',
  checking: 'bg-status-warning animate-pulse',
};

export function ConnectionStatus({
  status,
  compact = false,
}: {
  status: ConnectionState;
  compact?: boolean;
}) {
  const { t } = useTranslation('common');
  const labels: Record<ConnectionState, string> = {
    connected: t('connection.connected'),
    disconnected: t('connection.disconnected'),
    checking: t('connection.checking'),
  };
  const description = labels[status];
  const visible =
    status === 'connected'
      ? t('connection.online')
      : status === 'disconnected'
        ? t('connection.offline')
        : '...';
  // The rail draws the short word ("Online") and the longer phrase is what a
  // screen reader should hear. The name has to keep that word (WCAG 2.5.3);
  // a description that replaces it fails label-content-name-mismatch.
  // Compact mode hides the word, so the description alone is the name.
  const name = accessibleNameIncluding(description, compact ? '' : visible);
  return (
    <Tooltip text={description}>
      <button
        type="button"
        data-testid="connection-status"
        aria-label={name}
        className={`flex min-h-11 min-w-11 items-center gap-compact rounded focus-visible:outline-2 focus-visible:outline-brand-accent ${compact ? 'w-full justify-center' : ''}`}
      >
        <span role="status" className="flex items-center gap-compact">
          <span className={`h-2 w-2 rounded-full ${STATUS_STYLES[status]}`} />
          <span className={compact ? 'sr-only' : 'text-xs text-text-muted'}>{visible}</span>
        </span>
      </button>
    </Tooltip>
  );
}
