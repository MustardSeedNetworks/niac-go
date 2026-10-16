import { useTranslation } from 'react-i18next';
import type { ConnectionState } from '../hooks/useConnectionStatus';
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
  // The rail prints the short word ("Online") and keeps the longer phrase
  // ("Connected to backend") as the description. WCAG 2.5.3 requires every
  // visible word to occur, in order, inside the accessible name — an
  // aria-label of only the longer phrase fails label-content-name-mismatch.
  // Compact mode hides the word (sr-only), and "..." has no words a speech
  // command could match, so those keep the description as the name.
  const name = !compact && visible !== '...' ? `${visible}, ${description}` : description;
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
