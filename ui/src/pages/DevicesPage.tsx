import { FileCog, Server } from 'lucide-react';
import { type ChangeEvent, type FC, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Link } from 'react-router-dom';
import {
  fetchConfig,
  fetchDevices,
  fetchLibraryWalks,
  type LibraryFileEntry,
  updateConfig,
} from '../api/client';
import { isApiError } from '../api/errors';
import type { DeviceSummary } from '../api/types';
import { DeviceTable } from '../components/DeviceTable';
import { POLL_INTERVALS } from '../constants/polling';
import { iconSizes } from '../constants/sizes';
import { useApiResource } from '../hooks/useApiResource';
import { BaseCard } from '../ui/BaseCard';
import { Button } from '../ui/Button';
import { CardRow } from '../ui/Card';
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
  const { t } = useTranslation('pages');
  const {
    data: devices,
    loading,
    error,
  } = useApiResource(fetchDevices, [], { intervalMs: POLL_INTERVALS.slow });

  return (
    <BaseCard<DeviceSummary[]>
      title={t('devices.title')}
      subtitle={t('devices.subtitle')}
      icon={<Server className={`${iconSizes.lg} text-status-info`} />}
      data={devices}
      loading={loading && !devices}
      error={error?.message}
      getStatus={(d) => (d.length > 0 ? 'success' : 'unknown')}
      emptyMessage={t('devices.emptyMessage')}
    >
      {(data) => (
        <>
          <DeviceTable devices={data} />
          <Link
            to="/device-config"
            className="mt-heading inline-block text-sm text-brand-accent hover:underline"
          >
            {t('devices.editInLibrary')} &rarr;
          </Link>
        </>
      )}
    </BaseCard>
  );
};

/**
 * Config Editor Card - YAML configuration editor
 */
const ConfigEditorCard: FC = () => {
  const { t } = useTranslation('pages');
  const { t: tCommon } = useTranslation('common');
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
      setStatus({ tone: 'success', message: t('devices.configSaved') });
    } catch (err) {
      setStatus({ tone: 'error', message: getErrorMessage(err) });
    } finally {
      setSaving(false);
    }
  };

  const handleWalkCopy = async (name: string) => {
    try {
      await copyToClipboard(name);
      setStatus({ tone: 'success', message: t('devices.walkCopied', { name }) });
    } catch (err) {
      setStatus({
        tone: 'error',
        message: getErrorMessage(err) || t('devices.walkCopyError'),
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
        title={t('devices.yamlEditorTitle')}
        subtitle={t('devices.yamlEditorSubtitle')}
        icon={<FileCog className={`${iconSizes.lg} text-status-success`} />}
        data={null}
        emptyMessage={t('devices.noConfigMessage')}
        getStatus={() => 'success'}
      >
        {() => null}
      </BaseCard>
    );
  }

  return (
    <BaseCard<{ content: string; path: string; modifiedAt: string; sizeBytes: number }>
      title={t('devices.yamlEditorTitle')}
      subtitle={t('devices.yamlEditorSubtitle')}
      icon={<FileCog className={`${iconSizes.lg} text-status-success`} />}
      data={data}
      loading={loading && !data}
      error={error?.message}
      getStatus={() => (dirty ? 'warning' : 'success')}
    >
      {(cfg) => (
        <>
          <CardRow label={tCommon('labels.path')} value={cfg.path} mono />
          <CardRow label={tCommon('labels.updatedAt')} value={formatTime(cfg.modifiedAt)} />
          <CardRow label={tCommon('labels.size')} value={formatBytes(cfg.sizeBytes)} />
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
              title={t('devices.saveReloadTitle')}
            >
              {saving ? t('devices.savingLabel') : t('devices.saveReloadButton')}
            </Button>
            <Button variant="outline" disabled={!dirty || saving} onClick={handleReset}>
              {t('devices.discardChangesButton')}
            </Button>
          </div>
          <SmallText className="mt-inline text-text-muted">
            {t('devices.saveHelpTextPrefix')} <code>niac validate</code>
            {t('devices.saveHelpTextSuffix')}
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
            <div className="flex items-center gap-tight">
              <Button size="sm" variant="outline" onClick={() => onCopy(file.name)}>
                {t('devices.copyNameButton')}
              </Button>
              <Link
                to="/device-config/new#snmp"
                className="text-sm text-brand-accent hover:underline whitespace-nowrap"
              >
                {t('devices.useInEditor')}
              </Link>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};

export default DevicesPage;
