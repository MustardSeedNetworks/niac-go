import type { FC } from 'react';
import { useTranslation } from 'react-i18next';
import type { SimulationStatus } from '../../api/types';
import { type RollupState, StatusRollup } from '../../ui/StatusRollup';
import { formatUptime } from '../../utils/format';

export interface RuntimeRollupProps {
  /** Null once the poll has settled means no daemon, not an idle daemon. */
  simStatus: SimulationStatus | null;
  /** True only while the first poll is in flight. */
  loading: boolean;
}

/**
 * RuntimeRollup — the status band that opens the Simulation page.
 *
 * Lives here rather than inline because the state derivation is a chain of
 * conditions, and RuntimeControlPage is already at the Biome cognitive-
 * complexity ceiling; the page reads better delegating the question to a
 * component named after it.
 *
 * No daemon is the honest unknown, and it is why the rollup earns its place
 * here: without one there is no run to describe, so the figures are not zero —
 * they are unmeasurable, and StatusRollup prints em dashes instead of a
 * reassuring row of nothing. The daemon-mode banner below keeps the fix; the
 * rollup only states the condition.
 *
 * `degraded` is the daemon's own word for a run that cannot reach the wire, so
 * its reason becomes the headline rather than a category invented here. Idle is
 * calm rather than green: with a daemon up and nothing running, nothing is
 * wrong.
 */
export const RuntimeRollup: FC<RuntimeRollupProps> = ({ simStatus, loading }) => {
  const { t } = useTranslation('pages');

  const state: RollupState = loading || !simStatus ? 'unknown' : simStatus.degraded ? 'warn' : 'ok';

  const headline = loading
    ? t('runtime.rollup.checking')
    : !simStatus
      ? t('runtime.rollup.noDaemon')
      : simStatus.degraded
        ? (simStatus.degradedReason ?? t('runtime.rollup.degraded'))
        : simStatus.running
          ? t('runtime.rollup.running', {
              count: simStatus.deviceCount,
              iface: simStatus.interface ?? t('runtime.rollup.ifaceLabel'),
            })
          : t('runtime.rollup.idle');

  const body = loading
    ? undefined
    : !simStatus
      ? t('runtime.rollup.noDaemonBody')
      : simStatus.degraded
        ? t('runtime.rollup.degradedBody')
        : simStatus.running
          ? undefined
          : t('runtime.rollup.idleBody');

  return (
    <StatusRollup
      state={state}
      headline={headline}
      body={body}
      figures={[
        { label: t('runtime.rollup.ifaceLabel'), value: simStatus?.interface ?? '—' },
        {
          label: t('runtime.rollup.devicesLabel'),
          value: simStatus ? String(simStatus.deviceCount) : '—',
        },
        {
          label: t('runtime.rollup.uptimeLabel'),
          value: simStatus?.uptimeSeconds ? formatUptime(simStatus.uptimeSeconds) : '—',
        },
      ]}
    />
  );
};
