import { FileCog, Server } from 'lucide-react';
import { type FC, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Link } from 'react-router';
import { parseDocument } from 'yaml';
import { fetchConfig, fetchDevices, updateConfig } from '../api/client';
import { isApiError } from '../api/errors';
import { fetchLibraryWalks, type LibraryFileEntry } from '../api/library-client';
import type { DeviceSummary } from '../api/types';
import { YamlEditor } from '../components/config/YamlEditor';
import { DeviceTable } from '../components/DeviceTable';
import { POLL_INTERVALS } from '../constants/polling';
import { iconSizes } from '../constants/sizes';
import { useAppContext } from '../contexts/AppContext';
import { useApiResource } from '../hooks/useApiResource';
import { BaseCard } from '../ui/BaseCard';
import { Button } from '../ui/Button';
import { CardRow } from '../ui/Card';
import { SmallText } from '../ui/Typography';
import { findDeviceFragment, spliceDeviceFragment } from '../utils/device-fragment';
import { copyToClipboard } from '../utils/file';
import { formatBytes, formatTime, getErrorMessage } from '../utils/format';

/**
 * Devices Page - list + detail over the loaded config.
 *
 * The list and the editor used to be two cards that did not talk to each
 * other: selecting a device meant nothing, and the only way to change one was
 * to find it by eye in the whole-config YAML. Selecting a device now opens
 * that device's own block in the detail pane; saving splices it back into the
 * config, which is still what the daemon accepts. Clearing the selection
 * returns the pane to whole-config editing, which is the only way to reach
 * anything that is not a device.
 */
export const DevicesPage: FC = () => {
  const [selected, setSelected] = useState<string | null>(null);

  return (
    <div className="grid gap-spacious xl:grid-cols-2">
      <DeviceListCard selected={selected} onSelect={setSelected} />
      <ConfigEditorCard selected={selected} onClearSelection={() => setSelected(null)} />
    </div>
  );
};

/**
 * Device List Card - Shows devices from current config
 */
const DeviceListCard: FC<{
  selected: string | null;
  onSelect: (name: string) => void;
}> = ({ selected, onSelect }) => {
  const { t } = useTranslation('pages');
  const { sessionId } = useAppContext();
  const {
    data: devices,
    loading,
    error,
  } = useApiResource(() => fetchDevices(sessionId ?? ''), [sessionId], {
    intervalMs: POLL_INTERVALS.slow,
    enabled: sessionId !== null,
  });

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
      testId="devices-card"
    >
      {(data) => (
        <>
          <DeviceTable
            devices={data}
            selectedName={selected}
            onSelect={(device) => onSelect(device.name)}
          />
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
 * fragmentProblem names why a device fragment cannot be saved, as an i18n key
 * suffix, or null when it is fine. Parsing here rather than letting the splice
 * run means a malformed edit is refused whole, instead of being written into
 * the config for the daemon to reject after it has already replaced the file.
 */
function fragmentProblem(
  text: string,
): 'deviceFragmentInvalid' | 'deviceFragmentNameMissing' | null {
  const doc = parseDocument(text);
  if (doc.errors.length > 0) {
    return 'deviceFragmentInvalid';
  }
  const name = doc.get('name');
  if (typeof name !== 'string' || name.trim() === '') {
    return 'deviceFragmentNameMissing';
  }
  return null;
}

/**
 * Config Editor Card - YAML configuration editor
 */
const ConfigEditorCard: FC<{
  selected: string | null;
  onClearSelection: () => void;
}> = ({ selected, onClearSelection }) => {
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
  // Line reported by a structured YAML parse error (see internal/api
  // ErrorDetail.line), so the editor can highlight and scroll to it.
  const [errorLine, setErrorLine] = useState<number | null>(null);

  // The device's own block, located in the loaded config. Null whenever the
  // pane is showing the whole config — either because nothing is selected, or
  // because the selected device has no block the parser can find, in which
  // case the pane says so rather than editing something else.
  const fragment = useMemo(
    () => (data && selected ? findDeviceFragment(data.content, selected) : null),
    [data, selected],
  );
  const fragmentMissing = selected !== null && data !== null && fragment === null;
  const sourceText = fragment ? fragment.text : (data?.content ?? '');

  useEffect(() => {
    if (data && !dirty) {
      setValue(sourceText);
    }
  }, [data, dirty, sourceText]);

  // Switching subject discards an in-progress edit, so the pane never shows one
  // device's YAML under another device's name.
  useEffect(() => {
    setDirty(false);
    setStatus(null);
    setErrorLine(null);
  }, [selected]);

  const handleChange = (newValue: string) => {
    setValue(newValue);
    setDirty(true);
    setStatus(null);
    setErrorLine(null);
  };

  const handleReset = () => {
    if (data) {
      setValue(data.content);
      setDirty(false);
      setStatus(null);
      setErrorLine(null);
    }
  };

  const handleSave = async () => {
    if (!dirty || saving) {
      return;
    }
    if (fragment) {
      const problem = fragmentProblem(value);
      if (problem) {
        setStatus({ tone: 'error', message: t(`devices.${problem}`) });
        return;
      }
    }
    setSaving(true);
    setStatus(null);
    setErrorLine(null);
    try {
      const content = fragment ? spliceDeviceFragment(data?.content ?? '', fragment, value) : value;
      const updated = await updateConfig({ content });
      // The whole-config pane shows what came back; the device pane keeps the
      // edited block, because the response is the whole file.
      if (!fragment) {
        setValue(updated.content);
      }
      setDirty(false);
      setStatus({
        tone: 'success',
        message: fragment ? t('devices.deviceSaved') : t('devices.configSaved'),
      });
    } catch (err) {
      const detail = isApiError(err) ? err.details[0] : undefined;
      const line = detail?.line ?? null;
      const message =
        line !== null && detail
          ? t('devices.configParseErrorLine', { line, message: detail.issue })
          : getErrorMessage(err);
      setErrorLine(line);
      setStatus({ tone: 'error', message });
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

  const editingDevice = fragment !== null;

  return (
    <BaseCard<{ content: string; path: string; modifiedAt: string; sizeBytes: number }>
      title={editingDevice ? t('devices.deviceEditorTitle') : t('devices.yamlEditorTitle')}
      subtitle={editingDevice ? t('devices.deviceEditorSubtitle') : t('devices.yamlEditorSubtitle')}
      icon={<FileCog className={`${iconSizes.lg} text-status-success`} />}
      data={data}
      loading={loading && !data}
      error={error?.message}
      getStatus={() => (dirty ? 'warning' : 'success')}
      testId="config-editor-card"
    >
      {(cfg) => (
        <>
          {editingDevice ? (
            <CardRow label={t('devices.selectedLabel')} value={selected ?? ''} mono />
          ) : (
            <>
              <CardRow label={tCommon('labels.path')} value={cfg.path} mono />
              <CardRow label={tCommon('labels.updatedAt')} value={formatTime(cfg.modifiedAt)} />
              <CardRow label={tCommon('labels.size')} value={formatBytes(cfg.sizeBytes)} />
            </>
          )}
          {fragmentMissing && (
            <SmallText className="mt-heading text-status-warning" data-testid="fragment-missing">
              {t('devices.deviceFragmentMissing')}
            </SmallText>
          )}
          <YamlEditor
            className="mt-heading"
            height="18rem"
            value={value}
            onChange={handleChange}
            readOnly={loading || saving}
            errorLine={errorLine}
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
            {selected !== null && (
              <Button variant="outline" onClick={onClearSelection} data-testid="edit-whole-config">
                {t('devices.wholeConfigButton')}
              </Button>
            )}
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
