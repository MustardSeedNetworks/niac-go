import type { FC, ReactElement } from 'react';
import { useTranslation } from 'react-i18next';
import { useAppContext } from '../contexts/AppContext';
import { Select } from '../ui/Input';

/**
 * SessionSwitcher — which scenario this browser is reading, in the shell.
 *
 * A NIAC daemon runs several scenarios at once, and every runtime read is
 * scoped to one of them. The switcher lives in the rail rather than on the
 * runtime page so the answer to "which scenario am I looking at?" is on screen
 * wherever the operator is, and so switching does not mean navigating away
 * from the page they are reading.
 *
 * Selection is this browser's, held in AppContext: switching here repoints
 * Devices, Topology and Packets without touching what any other tab reads.
 */
export const SessionSwitcher: FC = (): ReactElement => {
  const { t } = useTranslation('common');
  const { sessionId, setSessionId, sessions } = useAppContext();

  return (
    <div className="flex items-center gap-compact min-w-0" data-testid="session-switcher">
      <span
        className={`h-2 w-2 rounded-full shrink-0 ${sessionId ? 'bg-status-success' : 'bg-bg-muted'}`}
        aria-hidden="true"
      />
      {sessions.length > 1 ? (
        <Select
          value={sessionId ?? ''}
          onChange={setSessionId}
          aria-label={t('session.switchLabel')}
          data-testid="session-switcher-select"
          containerClassName="min-w-0 flex-1"
          className="min-h-11 text-xs"
          options={sessions.flatMap(({ sessionId: id }) => (id ? [{ value: id, label: id }] : []))}
        />
      ) : (
        <span className="text-xs text-text-muted truncate">{sessionId ?? t('session.none')}</span>
      )}
    </div>
  );
};
