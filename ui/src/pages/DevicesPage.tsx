import { FileCog, Server } from 'lucide-react';
import { type ChangeEvent, type FC, memo, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  fetchConfig,
  fetchDevices,
  fetchLibraryWalks,
  type LibraryFileEntry,
  updateConfig,
} from '../api/client';
import { isApiError } from '../api/errors';
import type { DeviceSummary } from '../api/types';
import { POLL_INTERVALS } from '../constants/polling';
import { iconSizes } from '../constants/sizes';
import { useApiResource } from '../hooks/useApiResource';
import { useVirtualScroll } from '../hooks/useVirtualScroll';
import { BaseCard } from '../ui/BaseCard';
import { Button } from '../ui/Button';
import { CardRow } from '../ui/Card';
import { Tag } from '../ui/Tag';
import { SmallText } from '../ui/Typography';
import { copyToClipboard } from '../utils/file';
import { formatBytes, formatTime, getErrorMessage } from '../utils/format';

/**
 * Devices Page - Config Workspace
 *
 * Review YAML-derived devices, SNMP walks, DHCP/DNS personas, and packet playback targets.
 */
export const DevicesPage: FC = () => (
  <div className="grid gap-spacious xl:grid-cols-2">
    <DeviceListCard />
    <ConfigEditorCard />
  </div>
);

/**
 * Device List Card - Shows devices from current config
 */
const DeviceListCard: FC = () => {
  const {
    data: devices,
    loading,
    error,
  } = useApiResource(fetchDevices, [], { intervalMs: POLL_INTERVALS.slow });

  return (
    <BaseCard<DeviceSummary[]>
      title="Config workspace"
      subtitle="Devices rendered from active YAML config"
      icon={<Server className={`${iconSizes.lg} text-status-info`} />}
      data={devices}
      loading={loading && !devices}
      error={error?.message}
      getStatus={(d) => (d.length > 0 ? 'success' : 'unknown')}
      emptyMessage="No devices defined in the loaded configuration"
    >
      {(data) => <DeviceTable devices={data} />}
    </BaseCard>
  );
};

/**
 * Device Table with virtual scrolling for large device lists
 */
const DeviceTable = memo(({ devices }: { devices: DeviceSummary[] }) => {
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
        <table className="min-w-full divide-y divide-white/10 text-sm">
          <thead className="bg-bg-surface/60 text-xs uppercase tracking-wide text-text-muted">
            <tr>
              <th className="px-4 py-row-lg text-left">Device</th>
              <th className="px-4 py-row-lg text-left">Type</th>
              <th className="px-4 py-row-lg text-left">{t('devices.ipAddressesHeader')}</th>
              <th className="px-4 py-row-lg text-left">Protocols</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-white/5 text-text-secondary">
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
            <table className="min-w-full divide-y divide-white/10 text-sm">
              <thead className="bg-bg-surface/60 text-xs uppercase tracking-wide text-text-muted">
                <tr>
                  <th className="px-4 py-row-lg text-left">Device</th>
                  <th className="px-4 py-row-lg text-left">Type</th>
                  <th className="px-4 py-row-lg text-left">{t('devices.ipAddressesHeader')}</th>
                  <th className="px-4 py-row-lg text-left">Protocols</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-white/5 text-text-secondary">
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

/**
 * Config Editor Card - YAML configuration editor
 */
const ConfigEditorCard: FC = () => {
  const { data, loading, error } = useApiResource(fetchConfig, [], {
    intervalMs: POLL_INTERVALS.verySlow,
  });
  const { data: walkFiles } = useApiResource(fetchLibraryWalks, [], {
    intervalMs: POLL_INTERVALS.verySlow,
  });
  const [value, setValue] = useState('');
  const [dirty, setDirty] = useState(false);
  const [saving, setSaving] = useState(false);
  const [status, setStatus] = useState<{
    tone: 'success' | 'error';
    message: string;
  } | null>(null);

  useEffect(() => {
    if (data && !dirty) {
      setValue(data.content);
    }
  }, [data, dirty]);

  const handleChange = (event: ChangeEvent<HTMLTextAreaElement>) => {
    setValue(event.target.value);
    setDirty(true);
    setStatus(null);
  };

  const handleReset = () => {
    if (data) {
      setValue(data.content);
      setDirty(false);
      setStatus(null);
    }
  };

  const handleSave = async () => {
    if (!dirty || saving) {
      return;
    }
    setSaving(true);
    setStatus(null);
    try {
      const updated = await updateConfig({ content: value });
      setValue(updated.content);
      setDirty(false);
      setStatus({ tone: 'success', message: 'Configuration saved' });
    } catch (err) {
      setStatus({ tone: 'error', message: getErrorMessage(err) });
    } finally {
      setSaving(false);
    }
  };

  const handleWalkCopy = async (name: string) => {
    try {
      await copyToClipboard(name);
      setStatus({ tone: 'success', message: `Copied ${name}` });
    } catch (err) {
      setStatus({
        tone: 'error',
        message: getErrorMessage(err) || 'Unable to copy walk name',
      });
    }
  };

  // The daemon returns config_read_failed when no simulation has been
  // started yet (no path is loaded). That's a normal empty state, not
  // an error worth surfacing — render a clean prompt to go pick one.
  const noConfigLoaded =
    isApiError(error) && (error.code === 'config_read_failed' || error.status === 400);

  if (noConfigLoaded) {
    return (
      <BaseCard
        title="YAML editor"
        subtitle="Edit active configuration"
        icon={<FileCog className={`${iconSizes.lg} text-status-success`} />}
        data={null}
        emptyMessage="No simulation is running. Pick a network on Simulation and start it to see and edit the running YAML here."
        getStatus={() => 'success'}
      >
        {() => null}
      </BaseCard>
    );
  }

  return (
    <BaseCard<{ content: string; path: string; modifiedAt: string; sizeBytes: number }>
      title="YAML editor"
      subtitle="Edit active configuration"
      icon={<FileCog className={`${iconSizes.lg} text-status-success`} />}
      data={data}
      loading={loading && !data}
      error={error?.message}
      getStatus={() => (dirty ? 'warning' : 'success')}
    >
      {(cfg) => (
        <>
          <CardRow label="Path" value={cfg.path} mono />
          <CardRow label="Updated" value={formatTime(cfg.modifiedAt)} />
          <CardRow label="Size" value={formatBytes(cfg.sizeBytes)} />
          <textarea
            className="mt-heading h-72 w-full rounded-xl border border-surface-border bg-bg-base/70 pad-sm font-mono text-sm text-text-primary shadow-inner focus:border-brand-accent focus:outline-none"
            value={value}
            onChange={handleChange}
            spellCheck={false}
            disabled={loading || saving}
          />
          {status && (
            <SmallText
              className={status.tone === 'success' ? 'text-status-success' : 'text-status-error'}
            >
              {status.message}
            </SmallText>
          )}
          <div className="flex flex-wrap gap-default mt-heading">
            <Button
              tone="violet"
              disabled={!dirty || saving}
              onClick={handleSave}
              title="Validate, write to disk, and diff-reload the running simulation"
            >
              {saving ? 'Saving…' : 'Save & reload simulation'}
            </Button>
            <Button variant="outline" disabled={!dirty || saving} onClick={handleReset}>
              Discard changes
            </Button>
          </div>
          <SmallText className="mt-inline text-text-muted">
            Save runs the same validation as <code>niac validate</code>, writes the YAML to disk,
            then diff-reloads the running stack — added devices spin up, removed devices stop,
            existing devices are updated in place.
          </SmallText>
          <WalkFileBrowser files={walkFiles ?? []} onCopy={handleWalkCopy} />
        </>
      )}
    </BaseCard>
  );
};

/**
 * Walk File Browser - Browse available SNMP walks
 */
const WalkFileBrowser: FC<{
  files: LibraryFileEntry[];
  onCopy: (path: string) => void;
}> = ({ files, onCopy }) => {
  const { t } = useTranslation('pages');
  if (files.length === 0) {
    return null;
  }
  return (
    <div className="stack-sm">
      <SmallText className="text-text-muted">{t('devices.availableSnmpWalks')}</SmallText>
      <div className="max-h-48 stack-xs overflow-y-auto rounded-xl border border-surface-border bg-bg-base/50 pad-xs text-sm text-text-secondary">
        {files.map((file) => (
          <div
            key={file.name}
            className="flex-between gap-compact rounded-lg border border-surface-border bg-bg-surface/50 px-3 py-row"
          >
            <div>
              <p className="text-text-primary">{file.name}</p>
              <SmallText className="text-text-muted capitalize">{file.source}</SmallText>
            </div>
            <Button size="sm" variant="outline" onClick={() => onCopy(file.name)}>
              Copy name
            </Button>
          </div>
        ))}
      </div>
    </div>
  );
};

export default DevicesPage;
