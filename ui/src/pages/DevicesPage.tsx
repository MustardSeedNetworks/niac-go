import { FileCog, Server } from 'lucide-react';
import { type ChangeEvent, type FC, memo, useEffect, useState } from 'react';
import { fetchConfig, fetchDevices, fetchFiles, updateConfig } from '../api/client';
import type { DeviceSummary, FileEntry } from '../api/types';
import { POLL_INTERVALS } from '../constants/polling';
import { useApiResource } from '../hooks/useApiResource';
import { useVirtualScroll } from '../hooks/useVirtualScroll';
import { Button } from '../ui/Button';
import { Card, CardContent } from '../ui/Card';
import { Tag } from '../ui/Tag';
import { H2, SmallText } from '../ui/Typography';
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
    <Card className="border-white/5 bg-gray-900/70">
      <CardContent className="space-y-4">
        <H2 className="mb-0 flex items-center gap-2">
          <Server className="h-5 w-5 text-cyan-300" />
          Config workspace
        </H2>
        {loading && <SmallText className="text-gray-400">Loading devices...</SmallText>}
        {error && (
          <SmallText className="text-red-400">Unable to load devices: {error.message}</SmallText>
        )}
        {!(loading || error) && <DeviceTable devices={devices ?? []} />}
        <SmallText className="text-gray-400">
          Devices are rendered directly from the active YAML config so the CLI/TUI and Web UI always
          agree.
        </SmallText>
      </CardContent>
    </Card>
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
      <div className="rounded-xl border border-white/5 bg-gray-950/50 p-8 text-center text-gray-400">
        No devices defined in the loaded configuration.
      </div>
    );
  }

  if (!useVirtualization) {
    return (
      <div className="overflow-x-auto rounded-xl border border-white/5">
        <table className="min-w-full divide-y divide-white/10 text-sm">
          <thead className="bg-gray-900/60 text-xs uppercase tracking-wide text-gray-400">
            <tr>
              <th className="px-4 py-3 text-left">Device</th>
              <th className="px-4 py-3 text-left">Type</th>
              <th className="px-4 py-3 text-left">IP addresses</th>
              <th className="px-4 py-3 text-left">Protocols</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-white/5 text-gray-300">
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
      <div className="bg-gray-900/60 px-4 py-2 text-xs text-gray-400">
        Showing {virtualScroll.visibleItems.length} of {devices.length} devices (virtual scrolling
        enabled)
      </div>
      <div {...virtualScroll.containerProps} className="overflow-auto">
        <div {...virtualScroll.spacerProps}>
          <div {...virtualScroll.contentProps}>
            <table className="min-w-full divide-y divide-white/10 text-sm">
              <thead className="bg-gray-900/60 text-xs uppercase tracking-wide text-gray-400">
                <tr>
                  <th className="px-4 py-3 text-left">Device</th>
                  <th className="px-4 py-3 text-left">Type</th>
                  <th className="px-4 py-3 text-left">IP addresses</th>
                  <th className="px-4 py-3 text-left">Protocols</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-white/5 text-gray-300">
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
    <td className="px-4 py-3 font-semibold text-white">{device.name}</td>
    <td className="px-4 py-3">{device.type}</td>
    <td className="px-4 py-3 font-mono text-xs">{device.ips.join(', ') || '—'}</td>
    <td className="px-4 py-3">
      <div className="flex flex-wrap gap-2">
        {device.protocols.map((proto) => (
          <Tag key={`${device.name}-${proto}`} colorScheme="purple">
            {proto}
          </Tag>
        ))}
        {device.protocols.length === 0 && <SmallText className="text-gray-400">None</SmallText>}
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
  const { data: walkFiles } = useApiResource(() => fetchFiles('walks'), [], {
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

  const handleWalkCopy = async (path: string) => {
    try {
      await copyToClipboard(path);
      setStatus({ tone: 'success', message: `Copied ${path}` });
    } catch (err) {
      setStatus({
        tone: 'error',
        message: getErrorMessage(err) || 'Unable to copy path',
      });
    }
  };

  return (
    <Card className="border-white/5 bg-gray-900/70">
      <CardContent className="space-y-4">
        <H2 className="mb-0 flex items-center gap-2">
          <FileCog className="h-5 w-5 text-emerald-300" />
          YAML editor
        </H2>
        {loading && <SmallText className="text-gray-400">Loading configuration...</SmallText>}
        {error && (
          <SmallText className="text-red-400">Unable to load config: {error.message}</SmallText>
        )}
        {data && (
          <>
            <div className="flex flex-wrap gap-4 text-xs text-gray-400">
              <span>
                Path: <code className="font-mono text-white">{data.path}</code>
              </span>
              <span>Updated: {formatTime(data.modifiedAt)}</span>
              <span>Size: {formatBytes(data.sizeBytes)}</span>
            </div>
            <textarea
              className="h-72 w-full rounded-xl border border-white/10 bg-gray-950/70 p-3 font-mono text-sm text-white shadow-inner focus:border-violet-400 focus:outline-none"
              value={value}
              onChange={handleChange}
              spellCheck={false}
              disabled={loading || saving}
            />
            {status && (
              <SmallText
                className={status.tone === 'success' ? 'text-emerald-300' : 'text-red-400'}
              >
                {status.message}
              </SmallText>
            )}
            <div className="flex flex-wrap gap-3">
              <Button tone="violet" disabled={!dirty || saving} onClick={handleSave}>
                {saving ? 'Saving…' : 'Save changes'}
              </Button>
              <Button variant="outline" disabled={!dirty || saving} onClick={handleReset}>
                Discard
              </Button>
            </div>
            <SmallText className="text-gray-400">
              Saving runs full validation (same as `niac validate`) before persisting so runtime
              changes stay safe.
            </SmallText>
            <WalkFileBrowser files={walkFiles ?? []} onCopy={handleWalkCopy} />
          </>
        )}
      </CardContent>
    </Card>
  );
};

/**
 * Walk File Browser - Browse available SNMP walks
 */
const WalkFileBrowser: FC<{
  files: FileEntry[];
  onCopy: (path: string) => void;
}> = ({ files, onCopy }) => {
  if (files.length === 0) {
    return null;
  }
  return (
    <div className="space-y-2">
      <SmallText className="text-gray-400">Available SNMP walks</SmallText>
      <div className="max-h-48 space-y-1 overflow-y-auto rounded-xl border border-white/10 bg-gray-950/50 p-2 text-sm text-gray-300">
        {files.map((file) => (
          <div
            key={file.path}
            className="flex items-center justify-between gap-2 rounded-lg border border-white/5 bg-gray-900/50 px-3 py-2"
          >
            <div>
              <p className="text-white">{file.name}</p>
              <SmallText className="text-gray-500">{file.path}</SmallText>
            </div>
            <Button size="sm" variant="outline" onClick={() => onCopy(file.path)}>
              Copy path
            </Button>
          </div>
        ))}
      </div>
    </div>
  );
};

export default DevicesPage;
