import type { FC } from 'react';
import { useTranslation } from 'react-i18next';
import { useSimulationStatus } from '../hooks/useSimulationStatus';

/**
 * SimulationStatusChip — small shell-level indicator of whether the NIAC
 * simulation is currently running, mirrored next to ConnectionStatus in
 * HeaderBar. Backed by the single shared useSimulationStatus() poll (see
 * hooks/useSimulationStatus.ts) so adding this chip doesn't add a fourth
 * independent poller against `GET /api/v1/simulation`.
 */
export const SimulationStatusChip: FC = () => {
  const { t } = useTranslation('common');
  const { data: simStatus, loading } = useSimulationStatus();
  const running = simStatus?.running ?? false;
  const label = loading ? t('status.loading') : running ? t('status.running') : t('status.stopped');

  return (
    <div
      className="flex items-center gap-compact"
      title={label}
      data-testid="simulation-status-chip"
    >
      <span
        className={`h-2 w-2 rounded-full ${running ? 'bg-status-success' : 'bg-bg-muted'}`}
        aria-hidden="true"
      />
      <span className="text-xs text-text-muted">{label}</span>
    </div>
  );
};
