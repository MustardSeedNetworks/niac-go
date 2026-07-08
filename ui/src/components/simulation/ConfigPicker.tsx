import { FileUp, LayoutGrid, List, Search } from 'lucide-react';
import { type FC, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { fetchTemplateContent, fetchTemplates, importConfig } from '../../api/client';
import { fetchLibraryNetworks } from '../../api/library-client';
import type { LibraryNetwork, Template, TemplateContent } from '../../api/types';
import { iconSizes } from '../../constants/sizes';
import { useFavorites } from '../../hooks/useFavorites';
import { SmallText } from '../../ui/Typography';
import { copyToClipboard } from '../../utils/file';
import { TemplatePreviewModal } from '../TemplatePreviewModal';
import {
  type ConfigItem,
  type ConfigPickerProps,
  FAVORITES_STORAGE_KEY,
  readPref,
  VIEW_PREF_KEY,
  type ViewMode,
  writePref,
} from './ConfigPicker.types';
import { ViewToggle } from './ConfigPickerControls';
import { ConfigsList } from './ConfigPickerList';

export type { ConfigPickerProps } from './ConfigPicker.types';

/**
 * ConfigPicker is the single network-picker for the Simulation page.
 * Built-in and user-saved configs are presented as one flat list; the
 * user only sees "favorites" vs "everything else", with a one-shot
 * local upload pinned above both when present. Subcomponents:
 *
 *   ConfigPicker.types.ts     — ConfigItem, props, persisted-pref helpers
 *   ConfigPickerControls.tsx  — grid/list view toggle
 *   ConfigPickerList.tsx      — card grid + compact list presenters
 */
export const ConfigPicker: FC<ConfigPickerProps> = ({
  selection,
  onSelectTemplate,
  onSelectUserConfig,
  onUpload,
  uploadFile,
}) => {
  const { t } = useTranslation('pages');
  const [templates, setTemplates] = useState<Template[]>([]);
  const [userConfigs, setUserConfigs] = useState<LibraryNetwork[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [viewMode, setViewMode] = useState<ViewMode>(() => readPref(VIEW_PREF_KEY, 'grid'));
  const { isFavorite, toggleFavorite } = useFavorites(FAVORITES_STORAGE_KEY);

  // Preview modal state (built-in templates only)
  const [previewTemplate, setPreviewTemplate] = useState<Template | null>(null);
  const [previewContent, setPreviewContent] = useState<TemplateContent | null>(null);
  const [previewLoading, setPreviewLoading] = useState(false);
  const [previewError, setPreviewError] = useState<Error | null>(null);

  useEffect(() => {
    let cancelled = false;
    Promise.all([fetchTemplates(), fetchLibraryNetworks()])
      .then(([t, u]) => {
        if (cancelled) return;
        setTemplates(t);
        setUserConfigs(u);
      })
      .catch(() => {
        // Best-effort hydration; the empty state covers failure.
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  // Build the unified list — local upload first (most recent action), then
  // saved (user's stuff), then built-ins.
  const items: ConfigItem[] = useMemo(() => {
    const list: ConfigItem[] = [];
    if (uploadFile) {
      list.push({
        kind: 'local',
        key: 'local:current',
        name: uploadFile.name,
        description: t('configPicker.localFileSize', {
          size: (uploadFile.size / 1024).toFixed(1),
        }),
        deviceCount: 0,
        file: uploadFile,
      });
    }
    for (const c of userConfigs) {
      list.push({
        kind: 'saved',
        key: `saved:${c.name}`,
        name: c.name,
        description: c.description || c.useCase || '',
        deviceCount: c.deviceCount,
        config: c,
      });
    }
    for (const t of templates) {
      list.push({
        kind: 'builtin',
        key: `builtin:${t.name}`,
        name: t.name,
        description: t.description,
        deviceCount: t.deviceCount,
        template: t,
      });
    }
    return list;
  }, [templates, userConfigs, uploadFile]);

  /**
   * Searching narrows the list and keeps results alphabetical regardless
   * of star status — otherwise filtering would hide what the user typed
   * behind the favorites zone. Browsing (empty search) sorts favorites
   * first, then non-favorites, both alphabetical. The transient "local"
   * upload always pins to the very top, since it's the user's last
   * action.
   */
  const sections = useMemo(() => {
    const q = search.trim().toLowerCase();
    const matchesQuery = (item: ConfigItem) =>
      !q || item.name.toLowerCase().includes(q) || item.description.toLowerCase().includes(q);

    const byName = (a: ConfigItem, b: ConfigItem) => a.name.localeCompare(b.name);

    const local = items.filter((i) => i.kind === 'local' && matchesQuery(i));
    const matched = items.filter((i) => i.kind !== 'local' && matchesQuery(i));

    if (q) {
      return { local, favorites: [], all: matched.sort(byName) };
    }

    const favorites = matched.filter((i) => isFavorite(i.key)).sort(byName);
    const all = matched.filter((i) => !isFavorite(i.key)).sort(byName);
    return { local, favorites, all };
  }, [items, isFavorite, search]);

  const updateViewMode = (mode: ViewMode) => {
    setViewMode(mode);
    writePref(VIEW_PREF_KEY, mode);
  };

  const handleSelectItem = (item: ConfigItem) => {
    if (item.kind === 'builtin') onSelectTemplate(item.template);
    else if (item.kind === 'saved') onSelectUserConfig(item.config);
    // local items are already "selected" — RuntimeControlPage tracks them as
    // the upload file. Selecting again is a no-op.
  };

  const handleClearLocal = () => {
    onUpload(null);
  };

  const handleViewItem = async (item: ConfigItem) => {
    if (item.kind !== 'builtin') return; // Only built-ins have an inline preview today
    setPreviewTemplate(item.template);
    setPreviewContent(null);
    setPreviewError(null);
    setPreviewLoading(true);
    try {
      const content = await fetchTemplateContent(item.template.name);
      setPreviewContent(content);
    } catch (err) {
      setPreviewError(err as Error);
    } finally {
      setPreviewLoading(false);
    }
  };

  const closePreview = () => {
    setPreviewTemplate(null);
    setPreviewContent(null);
    setPreviewError(null);
  };

  const isItemSelected = (item: ConfigItem): boolean => {
    if (item.kind === 'builtin')
      return selection.source === 'template' && selection.name === item.name;
    if (item.kind === 'saved')
      return selection.source === 'userConfig' && selection.name === item.name;
    return selection.source === 'upload';
  };

  const [convertingDsl, setConvertingDsl] = useState(false);
  const [convertError, setConvertError] = useState<string | null>(null);

  const handleFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0] ?? null;
    setConvertError(null);
    if (!file) {
      onUpload(null);
      return;
    }
    // Sniff the first 4 KB for the legacy Java-DSL signature:
    //   device <name> {
    // If we see it, transparently convert via /api/v1/config/import
    // and hand the user a YAML File they can run directly. Removes the
    // separate converter card from the user's mental model.
    const head = await file.slice(0, 4096).text();
    const isJavaDsl = /^\s*device\s+[\w.-]+\s*\{/m.test(head);
    if (!isJavaDsl) {
      onUpload(file);
      return;
    }
    setConvertingDsl(true);
    try {
      const text = await file.text();
      const result = await importConfig({ format: 'java-dsl', content: text });
      const yamlName = `${file.name.replace(/\.(cfg|conf|txt)$/i, '')}.yaml`;
      const yamlFile = new File([result.yaml], yamlName, { type: 'application/x-yaml' });
      onUpload(yamlFile);
    } catch (err) {
      setConvertError(
        err instanceof Error
          ? t('configPicker.javaDslConvertFailedWithReason', { reason: err.message })
          : t('configPicker.javaDslConvertFailed'),
      );
    } finally {
      setConvertingDsl(false);
    }
  };

  return (
    <div className="stack">
      {/* Upload + clear */}
      <div className="flex flex-wrap items-center gap-compact">
        <div className="flex-1" />
        <label
          htmlFor="config-upload"
          className={`flex items-center gap-1.5 rounded border border-surface-border bg-bg-surface/60 px-3 py-compact text-xs font-medium text-text-primary hover:bg-surface-hover ${
            convertingDsl ? 'cursor-wait opacity-60' : 'cursor-pointer'
          }`}
          title={t('configPicker.pickFromDiskTitle')}
        >
          <FileUp className={iconSizes.sm} />
          {convertingDsl
            ? t('configPicker.converting')
            : uploadFile
              ? t('configPicker.replaceLocalFile')
              : t('configPicker.uploadLocalFile')}
        </label>
        <input
          id="config-upload"
          type="file"
          accept=".yaml,.yml,.cfg,.conf,.txt"
          onChange={handleFileChange}
          disabled={convertingDsl}
          className="sr-only"
        />
        {uploadFile && (
          <button
            type="button"
            onClick={handleClearLocal}
            className="text-xs font-medium text-status-error hover:text-status-error"
          >
            {t('configPicker.clearButton')}
          </button>
        )}
      </div>
      {convertError && (
        <SmallText className="text-status-error" role="alert">
          {convertError}
        </SmallText>
      )}

      {/* Search + view toggle */}
      <div className="flex items-center gap-compact">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-text-muted" />
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder={t('configPicker.searchPlaceholder')}
            className="w-full rounded border border-surface-border bg-bg-surface/60 py-row pl-9 pr-3 text-sm text-text-primary placeholder:text-text-muted focus:border-brand-accent focus:outline-none"
          />
        </div>
        <fieldset
          className="flex rounded border border-surface-border bg-bg-surface/60 p-0.5"
          aria-label={t('configPicker.viewDensityLabel')}
        >
          <ViewToggle
            active={viewMode === 'grid'}
            onClick={() => updateViewMode('grid')}
            icon={<LayoutGrid className={iconSizes.md} />}
            label={t('configPicker.cardView')}
          />
          <ViewToggle
            active={viewMode === 'list'}
            onClick={() => updateViewMode('list')}
            icon={<List className={iconSizes.md} />}
            label={t('configPicker.compactList')}
          />
        </fieldset>
      </div>

      {/* Sectioned list — local upload pinned, then favorites, then everything else */}
      <ConfigsList
        sections={sections}
        loading={loading}
        viewMode={viewMode}
        isSelected={isItemSelected}
        isFavorite={isFavorite}
        onSelect={handleSelectItem}
        onToggleFavorite={toggleFavorite}
        onView={handleViewItem}
        onClearLocal={handleClearLocal}
        searching={search.trim().length > 0}
      />

      {previewTemplate && (
        <TemplatePreviewModal
          template={previewTemplate}
          content={previewContent}
          loading={previewLoading}
          error={previewError}
          onClose={closePreview}
          onUse={(t) => {
            closePreview();
            onSelectTemplate(t);
          }}
          onCopy={async () => {
            await copyToClipboard(previewContent?.content ?? '');
          }}
        />
      )}
    </div>
  );
};
