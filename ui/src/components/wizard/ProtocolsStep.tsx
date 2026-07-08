import { ShieldCheck } from 'lucide-react';
import type { FC } from 'react';
import { useTranslation } from 'react-i18next';
import { fetchDevices } from '../../api/client';
import type { DeviceSummary } from '../../api/types';
import { DeviceTable } from '../../components/DeviceTable';
import { iconSizes } from '../../constants/sizes';
import { useApiResource } from '../../hooks/useApiResource';
import { BaseCard } from '../../ui/BaseCard';

/**
 * Step 3 — a focused, read-only view of per-device protocol config. Reuses
 * `DeviceTable`, which already renders a Protocols column (see
 * `DevicesPage`'s "Running Devices" card) — no new controls, just this
 * step's own fetch of the config the wizard just built in step 2.
 */
export const ProtocolsStep: FC = () => {
  const { t } = useTranslation('pages');
  const { data: devices, loading, error } = useApiResource(fetchDevices, []);

  return (
    <BaseCard<DeviceSummary[]>
      title={t('newSimWizard.protocols.title')}
      subtitle={t('newSimWizard.protocols.subtitle')}
      icon={<ShieldCheck className={`${iconSizes.lg} text-status-info`} />}
      data={devices}
      loading={loading && !devices}
      error={error?.message}
      getStatus={(d) => (d.length > 0 ? 'success' : 'unknown')}
      emptyMessage={t('devices.emptyMessage')}
    >
      {(data) => <DeviceTable devices={data} />}
    </BaseCard>
  );
};
