import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { forgetSavedSession } from '../../api/client';
import type { FailedRecoverySession, SimulationRecovery } from '../../api/types';
import { useToast } from '../../hooks/useToast';
import { Card } from '../../shared/Card';
import { ConfirmDialog } from '../../shared/ConfirmDialog';
import { ActionButton } from '../../ui/ActionButton';
import { useDialogLabels } from '../../ui/dialogLabels';
import { SmallText } from '../../ui/Typography';

interface SavedSessionRecoveryCardProps {
  recovery: SimulationRecovery;
  /** Called after a session is forgotten so the status behind `recovery` reloads. */
  onForgotten: () => void;
}

/**
 * The saved sessions the daemon could not restore at startup. Nothing here
 * blocks a new start; the card exists so a stale record, for example one whose
 * USB adapter is unplugged, can be forgotten without editing a root-owned file.
 */
export function SavedSessionRecoveryCard({ recovery, onForgotten }: SavedSessionRecoveryCardProps) {
  const { t } = useTranslation('pages');
  const dialogLabels = useDialogLabels();
  const { showError } = useToast();
  const [target, setTarget] = useState<FailedRecoverySession | null>(null);
  const [forgetting, setForgetting] = useState(false);

  const failed = recovery.failed ?? [];
  if (recovery.state !== 'failed' || failed.length === 0) return null;

  const confirmForget = async () => {
    if (!target) return;
    setForgetting(true);
    try {
      await forgetSavedSession(target.sessionId);
      onForgotten();
    } catch (err) {
      showError(err);
    } finally {
      setForgetting(false);
      setTarget(null);
    }
  };

  return (
    <Card title={t('runtime.recovery.title')} data-testid="saved-session-recovery-card">
      <div className="stack">
        <SmallText>{t('runtime.recovery.help')}</SmallText>
        <ul className="stack-sm">
          {failed.map((session) => (
            <li
              key={session.sessionId}
              className="flex flex-wrap items-center justify-between gap-default rounded border border-surface-border px-3 py-row"
              data-testid={`saved-session-${session.sessionId}`}
            >
              <div>
                <p className="font-semibold text-text-primary">
                  {t('runtime.recovery.session', {
                    session: session.sessionId,
                    interface: session.interface,
                  })}
                </p>
                <SmallText className="text-text-muted">{session.reason}</SmallText>
              </div>
              <ActionButton
                action="admin"
                variant="secondary"
                onClick={() => setTarget(session)}
                aria-label={t('runtime.recovery.forgetLabel', { session: session.sessionId })}
                data-testid={`forget-saved-session-${session.sessionId}`}
              >
                {t('runtime.recovery.forget')}
              </ActionButton>
            </li>
          ))}
        </ul>
      </div>
      <ConfirmDialog
        {...dialogLabels}
        open={target !== null}
        onConfirm={() => void confirmForget()}
        onCancel={() => setTarget(null)}
        title={t('runtime.recovery.confirmTitle')}
        message={t('runtime.recovery.confirmMessage', { session: target?.sessionId ?? '' })}
        confirmLabel={t('runtime.recovery.forget')}
        confirming={forgetting}
      />
    </Card>
  );
}
