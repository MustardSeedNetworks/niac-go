import { memo } from 'react';
import { useTranslation } from 'react-i18next';
import type { DeviceSummary } from '../api/types';
import { useVirtualScroll } from '../hooks/useVirtualScroll';
import { Tag } from '../ui/Tag';
import { SmallText } from '../ui/Typography';

/**
 * DeviceTable renders a DeviceSummary[] as a table, switching to virtual
 * scrolling above 100 rows. Shared by DevicesPage (the running config's
 * flat device list) and SegmentsPage (the same device shape grouped by
 * VLAN segment, ADR 0008) so both surfaces render devices identically.
 */
export const DeviceTable = memo(({ devices }: { devices: DeviceSummary[] }) => {
  const { t } = useTranslation('pages');
  const useVirtualization = devices.length >= 100;
  const virtualScroll = useVirtualScroll(devices, {
    itemHeight: 60,
    containerHeight: 600,
    overscan: 5,
  });

  if (devices.length === 0) {
    return (
      <div className="rounded-xl border border-surface-border bg-bg-base/50 pad-xl text-center text-text-muted">
        No devices defined in the loaded configuration.
      </div>
    );
  }

  if (!useVirtualization) {
    return (
      <div className="overflow-x-auto rounded-xl border border-surface-border">
        <table className="min-w-full divide-y divide-knob/10 text-sm">
          <thead className="bg-bg-surface/60 text-xs uppercase tracking-wide text-text-muted">
            <tr>
              <th className="px-4 py-row-lg text-left">Device</th>
              <th className="px-4 py-row-lg text-left">Type</th>
              <th className="px-4 py-row-lg text-left">{t('devices.ipAddressesHeader')}</th>
              <th className="px-4 py-row-lg text-left">Protocols</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-knob/5 text-text-secondary">
            {devices.map((device) => (
              <DeviceRow key={device.name} device={device} />
            ))}
          </tbody>
        </table>
      </div>
    );
  }

  return (
    <div className="rounded-xl border border-surface-border">
      <div className="bg-bg-surface/60 px-4 py-row text-xs text-text-muted">
        Showing {virtualScroll.visibleItems.length} of {devices.length} devices (virtual scrolling
        enabled)
      </div>
      <div {...virtualScroll.containerProps} className="overflow-auto">
        <div {...virtualScroll.spacerProps}>
          <div {...virtualScroll.contentProps}>
            <table className="min-w-full divide-y divide-knob/10 text-sm">
              <thead className="bg-bg-surface/60 text-xs uppercase tracking-wide text-text-muted">
                <tr>
                  <th className="px-4 py-row-lg text-left">Device</th>
                  <th className="px-4 py-row-lg text-left">Type</th>
                  <th className="px-4 py-row-lg text-left">{t('devices.ipAddressesHeader')}</th>
                  <th className="px-4 py-row-lg text-left">Protocols</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-knob/5 text-text-secondary">
                {virtualScroll.visibleItems.map(({ item: device }) => (
                  <DeviceRow key={device.name} device={device} />
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  );
});

DeviceTable.displayName = 'DeviceTable';

/**
 * Single device row
 */
const DeviceRow = memo(({ device }: { device: DeviceSummary }) => (
  <tr>
    <td className="px-4 py-row-lg font-semibold text-text-primary">{device.name}</td>
    <td className="px-4 py-row-lg">{device.type}</td>
    <td className="px-4 py-row-lg font-mono text-xs">{device.ips.join(', ') || '—'}</td>
    <td className="px-4 py-row-lg">
      <div className="flex flex-wrap gap-compact">
        {device.protocols.map((proto) => (
          <Tag key={`${device.name}-${proto}`} colorScheme="purple">
            {proto}
          </Tag>
        ))}
        {device.protocols.length === 0 && <SmallText className="text-text-muted">None</SmallText>}
      </div>
    </td>
  </tr>
));

DeviceRow.displayName = 'DeviceRow';
