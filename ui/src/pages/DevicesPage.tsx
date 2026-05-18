import { FileCog, Server } from 'lucide-react';
import { type ChangeEvent, type FC, memo, useEffect, useState } from 'react';
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
  <div className="grid gap-6 xl:grid-cols-2">
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
  const useVirtualization = devices.length >= 100;
  const virtualScroll = useVirtualScroll(devices, {
    itemHeight: 60,
    containerHeight: 600,
    overscan: 5,
  });

  if (devices.length === 0) {
    return (
      <div className="rounded-xl border border-white/5 bg-bg-base/50 p-8 text-center text-text-muted">
        No devices defined in the loaded configuration.
      </div>
    );
  }

  if (!useVirtualization) {
    return (
      <div className="overflow-x-auto rounded-xl border border-white/5">
        <table className="min-w-full divide-y divide-white/10 text-sm">
          <thead className="bg-bg-surface/60 text-xs uppercase tracking-wide text-text-muted">
            <tr>
              <th className="px-4 py-3 text-left">Device</th>
              <th className="px-4 py-3 text-left">Type</th>
              <th className="px-4 py-3 text-left">IP addresses</th>
              <th className="px-4 py-3 text-left">Protocols</th>
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
    <div className="rounded-xl border border-white/5">
      <div className="bg-bg-surface/60 px-4 py-2 text-xs text-text-muted">
        Showing {virtualScroll.visibleItems.length} of {devices.length} devices (virtual scrolling
        enabled)
      </div>
      <div {...virtualScroll.containerProps} className="overflow-auto">
        <div {...virtualScroll.spacerProps}>
          <div {...virtualScroll.contentProps}>
            <table className="min-w-full divide-y divide-white/10 text-sm">
              <thead className="bg-bg-surface/60 text-xs uppercase tracking-wide text-text-muted">
                <tr>
                  <th className="px-4 py-3 text-left">Device</th>
                  <th className="px-4 py-3 text-left">Type</th>
                  <th className="px-4 py-3 text-left">IP addresses</th>
                  <th className="px-4 py-3 text-left">Protocols</th>
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
    <td className="px-4 py-3 font-semibold text-text-primary">{device.name}</td>
    <td className="px-4 py-3">{device.type}</td>
    <td className="px-4 py-3 font-mono text-xs">{device.ips.join(', ') || '—'}</td>
    <td className="px-4 py-3">
      <div className="flex flex-wrap gap-2">
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
            className="mt-3 h-72 w-full rounded-xl border border-white/10 bg-bg-base/70 p-3 font-mono text-sm text-text-primary shadow-inner focus:border-brand-400 focus:outline-none"
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
          <div className="flex flex-wrap gap-3 mt-3">
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
          <SmallText className="mt-2 text-text-muted">
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
  if (files.length === 0) {
    return null;
  }
  return (
    <div className="space-y-2">
      <SmallText className="text-text-muted">Available SNMP walks</SmallText>
      <div className="max-h-48 space-y-1 overflow-y-auto rounded-xl border border-white/10 bg-bg-base/50 p-2 text-sm text-text-secondary">
        {files.map((file) => (
          <div
            key={file.name}
            className="flex items-center justify-between gap-2 rounded-lg border border-white/5 bg-bg-surface/50 px-3 py-2"
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
