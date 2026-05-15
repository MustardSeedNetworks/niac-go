import {
  ChevronDown,
  ChevronRight,
  FileCode,
  FileUp,
  LayoutGrid,
  List,
  Search,
} from 'lucide-react';
import { type FC, useEffect, useMemo, useState } from 'react';
import {
  fetchTemplateContent,
  fetchTemplates,
  fetchUserConfigs,
  importConfig,
} from '../../api/client';
import type { Template, TemplateContent, UserConfig } from '../../api/types';
import { iconSizes } from '../../constants/sizes';
import { SmallText } from '../../ui/Typography';
import { TemplatePreviewModal } from '../TemplatePreviewModal';
import { JavaDslImportCard } from '../templates/JavaDslImportCard';
import {
  type ConfigItem,
  type ConfigPickerProps,
  readPref,
  SOURCE_FILTER_PREF_KEY,
  type SourceFilter,
  VIEW_PREF_KEY,
  type ViewMode,
  writePref,
} from './ConfigPicker.types';
import { SourceChip, ViewToggle } from './ConfigPickerControls';
import { ConfigsList } from './ConfigPickerList';

export type { ConfigPickerProps } from './ConfigPicker.types';

/**
 * ConfigPicker is the single config-source picker for the Simulation
 * page. Built-in templates, user-saved configs, and a one-shot local
 * upload all live in the same list with a "source" filter chip; there
 * are no tabs. Subcomponents live alongside this file:
 *
 *   ConfigPicker.types.ts     — ConfigItem, props, persisted-pref helpers
 *   ConfigPickerControls.tsx  — source-filter chip + grid/list toggle
 *   ConfigPickerList.tsx      — card grid + compact list presenters
 */
export const ConfigPicker: FC<ConfigPickerProps> = ({
  selection,
  onSelectTemplate,
  onSelectUserConfig,
  onUpload,
  uploadFile,
}) => {
  const [templates, setTemplates] = useState<Template[]>([]);
  const [userConfigs, setUserConfigs] = useState<UserConfig[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [viewMode, setViewMode] = useState<ViewMode>(() => readPref(VIEW_PREF_KEY, 'grid'));
  const [sourceFilter, setSourceFilter] = useState<SourceFilter>(() =>
    readPref(SOURCE_FILTER_PREF_KEY, 'all'),
  );
  const [showJavaDsl, setShowJavaDsl] = useState(false);

  // Preview modal state (built-in templates only)
  const [previewTemplate, setPreviewTemplate] = useState<Template | null>(null);
  const [previewContent, setPreviewContent] = useState<TemplateContent | null>(null);
  const [previewLoading, setPreviewLoading] = useState(false);
  const [previewError, setPreviewError] = useState<Error | null>(null);

  useEffect(() => {
    let cancelled = false;
    Promise.all([fetchTemplates(), fetchUserConfigs()])
      .then(([t, u]) => {
        if (cancelled) return;
        setTemplates(t);
        setUserConfigs(u.configs);
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
        description: `${(uploadFile.size / 1024).toFixed(1)} KB local file`,
        deviceCount: 0,
        file: uploadFile,
      });
    }
    for (const c of userConfigs) {
      list.push({
        kind: 'saved',
        key: `saved:${c.path}`,
        name: c.name,
        description: c.path,
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

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase();
    return items.filter((item) => {
      if (sourceFilter === 'builtin' && item.kind !== 'builtin') return false;
      if (sourceFilter === 'saved' && item.kind !== 'saved') return false;
      if (sourceFilter === 'local' && item.kind !== 'local') return false;
      if (!q) return true;
      return item.name.toLowerCase().includes(q) || item.description.toLowerCase().includes(q);
    });
  }, [items, sourceFilter, search]);

  const counts = useMemo(
    () => ({
      all: items.length,
      builtin: items.filter((i) => i.kind === 'builtin').length,
      saved: items.filter((i) => i.kind === 'saved').length,
      local: items.filter((i) => i.kind === 'local').length,
    }),
    [items],
  );

  const updateViewMode = (mode: ViewMode) => {
    setViewMode(mode);
    writePref(VIEW_PREF_KEY, mode);
  };
  const updateSourceFilter = (f: SourceFilter) => {
    setSourceFilter(f);
    writePref(SOURCE_FILTER_PREF_KEY, f);
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
      updateSourceFilter('local');
      return;
    }
    setConvertingDsl(true);
    try {
      const text = await file.text();
      const result = await importConfig({ format: 'java-dsl', content: text });
      const yamlName = `${file.name.replace(/\.(cfg|conf|txt)$/i, '')}.yaml`;
      const yamlFile = new File([result.yaml], yamlName, { type: 'application/x-yaml' });
      onUpload(yamlFile);
      updateSourceFilter('local');
    } catch (err) {
      setConvertError(
        err instanceof Error
          ? `Couldn't auto-convert Java DSL: ${err.message}`
          : "Couldn't auto-convert Java DSL",
      );
    } finally {
      setConvertingDsl(false);
    }
  };

  return (
    <div className="space-y-3">
      {/* Source filter chips + Upload button */}
      <div className="flex flex-wrap items-center gap-2">
        <SourceChip
          active={sourceFilter === 'all'}
          onClick={() => updateSourceFilter('all')}
          label="All"
          count={counts.all}
        />
        <SourceChip
          active={sourceFilter === 'builtin'}
          onClick={() => updateSourceFilter('builtin')}
          label="Built-in"
          count={counts.builtin}
        />
        <SourceChip
          active={sourceFilter === 'saved'}
          onClick={() => updateSourceFilter('saved')}
          label="Saved"
          count={counts.saved}
        />
        {counts.local > 0 && (
          <SourceChip
            active={sourceFilter === 'local'}
            onClick={() => updateSourceFilter('local')}
            label="Local"
            count={counts.local}
          />
        )}
        <div className="flex-1" />
        <label
          htmlFor="config-upload"
          className={`flex items-center gap-1.5 rounded border border-white/10 bg-gray-900/60 px-3 py-1 text-xs font-medium text-gray-200 hover:bg-white/5 ${
            convertingDsl ? 'cursor-wait opacity-60' : 'cursor-pointer'
          }`}
          title="Pick a config from disk. YAML is used as-is; legacy Java-DSL (.cfg) is auto-converted."
        >
          <FileUp className={iconSizes.sm} />
          {convertingDsl ? 'Converting…' : uploadFile ? 'Replace local file' : 'Upload local file…'}
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
            className="text-xs font-medium text-red-300 hover:text-red-200"
          >
            Clear
          </button>
        )}
      </div>
      {convertError && (
        <SmallText className="text-red-300" role="alert">
          {convertError}
        </SmallText>
      )}

      {/* Search + view toggle */}
      <div className="flex items-center gap-2">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-500" />
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search configs…"
            className="w-full rounded border border-white/10 bg-gray-900/60 py-2 pl-9 pr-3 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none"
          />
        </div>
        <fieldset
          className="flex rounded border border-white/10 bg-gray-900/60 p-0.5"
          aria-label="View density"
        >
          <ViewToggle
            active={viewMode === 'grid'}
            onClick={() => updateViewMode('grid')}
            icon={<LayoutGrid className={iconSizes.md} />}
            label="Card view"
          />
          <ViewToggle
            active={viewMode === 'list'}
            onClick={() => updateViewMode('list')}
            icon={<List className={iconSizes.md} />}
            label="Compact list"
          />
        </fieldset>
      </div>

      {/* List / grid */}
      <ConfigsList
        items={filtered}
        loading={loading}
        viewMode={viewMode}
        isSelected={isItemSelected}
        onSelect={handleSelectItem}
        onView={handleViewItem}
        onClearLocal={handleClearLocal}
      />

      {/* Java-DSL import — collapsed by default since most users won't need it */}
      <details
        className="rounded border border-white/10 bg-gray-950/40 p-3"
        open={showJavaDsl}
        onToggle={(e) => setShowJavaDsl(e.currentTarget.open)}
      >
        <summary className="flex cursor-pointer items-center gap-2 text-sm text-gray-300 hover:text-white">
          {showJavaDsl ? (
            <ChevronDown className={iconSizes.md} />
          ) : (
            <ChevronRight className={iconSizes.md} />
          )}
          <FileCode className={iconSizes.md} />
          <span>
            Convert a legacy Java-DSL <code className="text-xs">.cfg</code> to YAML
          </span>
        </summary>
        <div className="mt-3">
          <JavaDslImportCard />
        </div>
      </details>

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
            // No-op; the modal expects a Promise<void>.
          }}
        />
      )}
    </div>
  );
};
