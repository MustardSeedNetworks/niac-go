import type { FC } from 'react';
import { useTranslation } from 'react-i18next';
import { fetchSessionClients } from '../../api/client';
import type { AttachmentPort, CompiledAttachment } from '../../api/types';
import { POLL_INTERVALS } from '../../constants/polling';
import { useApiResource } from '../../hooks/useApiResource';
import { SmallText } from '../../ui/Typography';
import { AttachedClientMove } from '../runtime/AttachedClientMove';

interface DeviceAttachmentPoolProps {
  sessionId: string;
  pool: CompiledAttachment;
  device: string;
}

/**
 * Who is plugged into this device's tester ports. Each pool port on the device
 * reads as the client on it, the MAC it is held for, or free; a client here can
 * be moved to any free port of the pool, on this device or another.
 */
export const DeviceAttachmentPool: FC<DeviceAttachmentPoolProps> = ({
  sessionId,
  pool,
  device,
}) => {
  const { t } = useTranslation('pages');
  const {
    data: clients,
    error,
    refetch,
  } = useApiResource(() => fetchSessionClients(sessionId), ['clients', sessionId], {
    intervalMs: POLL_INTERVALS.medium,
  });

  const ports = (pool.ports ?? []).filter((port) => port.device === device);
  const here = (clients ?? []).filter((client) => client.device === device);

  const occupant = (port: AttachmentPort): string => {
    const client = here.find((candidate) => candidate.interface === port.interface);
    if (client) return client.mac;
    const pin = pool.pins?.find(
      (candidate) => candidate.device === device && candidate.interface === port.interface,
    );
    return pin
      ? t('topology.deviceDetails.pool.heldFor', { mac: pin.mac })
      : t('topology.deviceDetails.pool.free');
  };

  return (
    <section className="mb-content stack-sm" data-testid="device-attachment-pool">
      <div>
        <div className="text-xs text-text-muted uppercase tracking-wide mb-tight">
          {t('topology.deviceDetails.pool.title')}
        </div>
        <SmallText>{t('topology.deviceDetails.pool.help')}</SmallText>
      </div>
      {error ? (
        <SmallText className="text-status-error" role="alert">
          {t('runtime.clients.loadError', { error: error.message })}
        </SmallText>
      ) : (
        <ul className="stack-xs">
          {ports.map((port) => (
            <li
              key={port.interface}
              data-testid={`device-attachment-port-${port.interface}`}
              className="flex items-baseline justify-between gap-default text-sm"
            >
              <span className="text-text-primary">
                {port.interface}{' '}
                <span className="text-xs text-text-muted">
                  {port.vlan === undefined
                    ? t('runtime.fabric.untagged')
                    : t('newSimWizard.networks.attachmentPortVlan', { vlan: port.vlan })}
                </span>
              </span>
              <span
                data-testid="device-attachment-port-occupant"
                className="font-mono text-xs text-text-muted"
              >
                {clients === null ? t('runtime.clients.loading') : occupant(port)}
              </span>
            </li>
          ))}
        </ul>
      )}
      {clients && here.length > 0 && (
        <AttachedClientMove
          sessionId={sessionId}
          pool={pool}
          clients={clients}
          device={device}
          onMoved={refetch}
        />
      )}
    </section>
  );
};
