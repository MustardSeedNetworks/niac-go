import { Play } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { DeviceActionPanel } from '../components/DeviceActionPanel';
import { DeviceFaultPanel } from '../components/DeviceFaultPanel';
import { ErrorInjectionPanel } from '../components/ErrorInjectionPanel';
import { ReplayControlPanel } from '../components/ReplayControlPanel';
import { iconSizes } from '../constants/sizes';
import { useSimulationStatus } from '../hooks/useSimulationStatus';
import { ButtonLink } from '../shared/Button';
import { linkTo } from '../ui/linkTo';
import { StatusRollup } from '../ui/StatusRollup';

export function TrafficInjectionPage() {
  const { t } = useTranslation('pages');
  const { data: simStatus, loading } = useSimulationStatus();

  /* Every control here acts on a running simulation's devices. With none
     running they were all disabled at once, with nothing saying why and no
     way to reach a simulation from this page (#2188). It now answers the same
     three things the Dashboard's rollup does — what state this is, why, and
     the one action that changes it — and the panels come back with the
     simulation rather than sitting dead above the fold.

     While the poll is still out the state is unknown, not idle: printing
     "nothing is running" over a stack that is running would be a worse lie
     than a rollup that arrives a moment late. */
  if (!(loading || simStatus?.running)) {
    return (
      <StatusRollup
        state="idle"
        headline={t('traffic.idle.headline')}
        body={t('traffic.idle.body')}
        actions={
          <ButtonLink
            variant="primary"
            renderLink={linkTo('/runtime')}
            icon={<Play className={iconSizes.md} />}
          >
            {t('traffic.idle.startAction')}
          </ButtonLink>
        }
      />
    );
  }

  return (
    <div className="stack-xl">
      <ErrorInjectionPanel />
      <DeviceFaultPanel />
      <DeviceActionPanel />
      <ReplayControlPanel />
    </div>
  );
}
