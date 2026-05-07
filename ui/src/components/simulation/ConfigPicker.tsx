import {
  Building2,
  ChevronDown,
  ChevronRight,
  Eye,
  FileCode,
  FileUp,
  FolderOpen,
  Globe,
  HardDrive,
  LayoutGrid,
  List,
  Router,
  Search,
  Server,
  Wifi,
} from 'lucide-react';
import { type FC, useEffect, useMemo, useState } from 'react';
import { fetchTemplateContent, fetchTemplates, fetchUserConfigs } from '../../api/client';
import type { Template, TemplateContent, UserConfig } from '../../api/types';
import { Tag } from '../../ui/Tag';
import { SmallText } from '../../ui/Typography';
import { TemplatePreviewModal } from '../TemplatePreviewModal';
import { JavaDslImportCard } from '../templates/JavaDslImportCard';

type ViewMode = 'grid' | 'list';
type SourceFilter = 'all' | 'builtin' | 'saved' | 'local';

const VIEW_PREF_KEY = 'niac.configs.viewMode';
const SOURCE_FILTER_PREF_KEY = 'niac.configs.sourceFilter';

const TEMPLATE_TYPE_ICON: Record<Template['type'], FC<{ className?: string }>> = {
  basic: Globe,
  router: Router,
  switch: FileCode,
  'access-point': Wifi,
  server: Server,
  complete: Building2,
  custom: FileCode,
};

const TEMPLATE_TYPE_TINT: Record<Template['type'], string> = {
  basic: 'bg-blue-500/15 text-blue-200 border-blue-400/30',
  router: 'bg-orange-500/15 text-orange-200 border-orange-400/30',
  switch: 'bg-emerald-500/15 text-emerald-200 border-emerald-400/30',
  'access-point': 'bg-purple-500/15 text-purple-200 border-purple-400/30',
  server: 'bg-cyan-500/15 text-cyan-200 border-cyan-400/30',
  complete: 'bg-amber-500/15 text-amber-200 border-amber-400/30',
  custom: 'bg-pink-500/15 text-pink-200 border-pink-400/30',
};

// One row in the unified Configs list.
type ConfigItem =
  | {
      kind: 'builtin';
      key: string;
      name: string;
      description: string;
      deviceCount: number;
      template: Template;
    }
  | {
      kind: 'saved';
      key: string;
      name: string;
      description: string;
      deviceCount: number;
      config: UserConfig;
    }
  | {
      kind: 'local';
      key: string;
      name: string;
      description: string;
      deviceCount: number;
      file: File;
    };

interface Selection {
  source: 'template' | 'userConfig' | 'upload' | null;
  name: string;
  path?: string;
}

export interface ConfigPickerProps {
  /** The currently selected config. */
  selection: Selection;
  /** Called when the user picks a built-in template. */
  onSelectTemplate: (template: Template) => void;
  /** Called when the user picks a saved (user) config. */
  onSelectUserConfig: (config: UserConfig) => void;
  /** Called when the user uploads a file (or clears it). */
  onUpload: (file: File | null) => void;
  /** The current upload file, if any. */
  uploadFile: File | null;
}

/**
 * ConfigPicker is the single config-source picker for the Simulation page.
 * Built-in templates, user-saved configs, and a one-shot local upload all
 * live in the same list with a "source" filter chip; there are no tabs.
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

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0] ?? null;
    onUpload(file);
    // Force the source filter to "local" so the user immediately sees their
    // file in the list.
    if (file) updateSourceFilter('local');
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
          className="flex cursor-pointer items-center gap-1.5 rounded border border-white/10 bg-gray-900/60 px-3 py-1 text-xs font-medium text-gray-200 hover:bg-white/5"
          title="Pick a YAML from disk to use for this run (one-shot, not saved to your library)."
        >
          <FileUp className="h-3.5 w-3.5" />
          {uploadFile ? 'Replace local file' : 'Upload local file…'}
        </label>
        <input
          id="config-upload"
          type="file"
          accept=".yaml,.yml"
          onChange={handleFileChange}
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
            icon={<LayoutGrid className="h-4 w-4" />}
            label="Card view"
          />
          <ViewToggle
            active={viewMode === 'list'}
            onClick={() => updateViewMode('list')}
            icon={<List className="h-4 w-4" />}
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
          {showJavaDsl ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
          <FileCode className="h-4 w-4" />
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

function readPref<T extends string>(key: string, fallback: T): T {
  if (typeof window === 'undefined') return fallback;
  const saved = window.localStorage.getItem(key);
  return (saved as T) || fallback;
}

function writePref(key: string, value: string) {
  if (typeof window !== 'undefined') {
    window.localStorage.setItem(key, value);
  }
}

const SourceChip: FC<{
  active: boolean;
  onClick: () => void;
  label: string;
  count: number;
}> = ({ active, onClick, label, count }) => (
  <button
    type="button"
    onClick={onClick}
    aria-pressed={active}
    className={`flex items-center gap-2 rounded-full px-3 py-1 text-xs font-medium transition-colors ${
      active
        ? 'bg-violet-500/20 text-violet-100 ring-1 ring-violet-400/40'
        : 'bg-gray-800/60 text-gray-400 ring-1 ring-white/10 hover:bg-white/5 hover:text-gray-200'
    }`}
  >
    <span>{label}</span>
    <span
      className={`rounded-full px-1.5 py-0.5 text-[10px] ${
        active ? 'bg-violet-500/30 text-violet-100' : 'bg-white/10 text-gray-300'
      }`}
    >
      {count}
    </span>
  </button>
);

const ViewToggle: FC<{
  active: boolean;
  onClick: () => void;
  icon: React.ReactNode;
  label: string;
}> = ({ active, onClick, icon, label }) => (
  <button
    type="button"
    onClick={onClick}
    aria-pressed={active}
    title={label}
    className={`rounded px-2 py-1 transition-colors ${
      active
        ? 'bg-violet-500/20 text-violet-100'
        : 'text-gray-400 hover:bg-white/5 hover:text-gray-200'
    }`}
  >
    {icon}
  </button>
);

const ConfigsList: FC<{
  items: ConfigItem[];
  loading: boolean;
  viewMode: ViewMode;
  isSelected: (item: ConfigItem) => boolean;
  onSelect: (item: ConfigItem) => void;
  onView: (item: ConfigItem) => void;
  onClearLocal: () => void;
}> = ({ items, loading, viewMode, isSelected, onSelect, onView, onClearLocal }) => {
  if (loading) {
    return <SmallText className="text-gray-500">Loading configs…</SmallText>;
  }
  if (items.length === 0) {
    return (
      <SmallText className="text-gray-500">
        Nothing matches. Try a different search, switch the source filter, or upload a local YAML
        with the button above.
      </SmallText>
    );
  }

  if (viewMode === 'grid') {
    return (
      <div className="grid max-h-[420px] grid-cols-1 gap-3 overflow-y-auto pr-1 sm:grid-cols-2 xl:grid-cols-3">
        {items.map((item) => (
          <ConfigCard
            key={item.key}
            item={item}
            selected={isSelected(item)}
            onSelect={onSelect}
            onView={onView}
            onClearLocal={onClearLocal}
          />
        ))}
      </div>
    );
  }

  return (
    <ul className="max-h-72 divide-y divide-white/5 overflow-y-auto rounded-lg border border-white/10 bg-gray-950/40">
      {items.map((item) => (
        <ConfigRow
          key={item.key}
          item={item}
          selected={isSelected(item)}
          onSelect={onSelect}
          onView={onView}
          onClearLocal={onClearLocal}
        />
      ))}
    </ul>
  );
};

function sourceTagFor(item: ConfigItem) {
  if (item.kind === 'builtin')
    return (
      <Tag colorScheme="purple" className="text-[10px]">
        Built-in
      </Tag>
    );
  if (item.kind === 'saved')
    return (
      <Tag colorScheme="green" className="text-[10px]">
        Saved
      </Tag>
    );
  return (
    <Tag colorScheme="blue" className="text-[10px]">
      Local
    </Tag>
  );
}

const ConfigCard: FC<{
  item: ConfigItem;
  selected: boolean;
  onSelect: (item: ConfigItem) => void;
  onView: (item: ConfigItem) => void;
  onClearLocal: () => void;
}> = ({ item, selected, onSelect, onView, onClearLocal }) => {
  const Icon =
    item.kind === 'builtin'
      ? (TEMPLATE_TYPE_ICON[item.template.type] ?? FileCode)
      : item.kind === 'saved'
        ? FolderOpen
        : HardDrive;
  const tint =
    item.kind === 'builtin'
      ? (TEMPLATE_TYPE_TINT[item.template.type] ?? TEMPLATE_TYPE_TINT.custom)
      : item.kind === 'saved'
        ? 'bg-emerald-500/15 text-emerald-200 border-emerald-400/30'
        : 'bg-blue-500/15 text-blue-200 border-blue-400/30';

  return (
    <div
      className={`flex flex-col gap-3 rounded-lg border p-3 transition-colors ${
        selected
          ? 'border-violet-400/50 bg-violet-500/10'
          : 'border-white/10 bg-gray-950/40 hover:border-violet-500/30'
      }`}
    >
      <div className="flex items-start justify-between gap-2">
        <div className={`rounded-md border p-2 ${tint}`}>
          <Icon className="h-5 w-5" />
        </div>
        {sourceTagFor(item)}
      </div>
      <div>
        <div className="font-semibold text-white">{item.name}</div>
        <SmallText className="mt-0.5 line-clamp-2 text-gray-400">
          {item.description || 'No description'}
        </SmallText>
      </div>
      <div className="flex flex-wrap items-center gap-1.5">
        {item.kind !== 'local' && (
          <Tag colorScheme="purple" className="text-[10px]">
            {item.deviceCount} {item.deviceCount === 1 ? 'device' : 'devices'}
          </Tag>
        )}
        {item.kind === 'builtin' &&
          item.template.tags?.slice(0, 2).map((tag) => (
            <Tag key={tag} colorScheme="gray" className="text-[10px]">
              {tag}
            </Tag>
          ))}
      </div>
      <div className="flex gap-2">
        <button
          type="button"
          onClick={() => onSelect(item)}
          className="flex-1 rounded bg-violet-500/20 px-2 py-1.5 text-xs font-medium text-violet-100 ring-1 ring-violet-400/40 hover:bg-violet-500/30"
        >
          {selected ? 'Selected' : 'Use'}
        </button>
        {item.kind === 'builtin' && (
          <button
            type="button"
            onClick={() => onView(item)}
            className="rounded border border-white/10 bg-gray-900/60 px-2 py-1.5 text-xs font-medium text-gray-200 hover:bg-white/10"
            title="Preview YAML"
          >
            <Eye className="h-3.5 w-3.5" />
          </button>
        )}
        {item.kind === 'local' && (
          <button
            type="button"
            onClick={onClearLocal}
            className="rounded border border-red-400/30 bg-red-500/10 px-2 py-1.5 text-xs font-medium text-red-200 hover:bg-red-500/20"
            title="Drop the local file"
          >
            Clear
          </button>
        )}
      </div>
    </div>
  );
};

const ConfigRow: FC<{
  item: ConfigItem;
  selected: boolean;
  onSelect: (item: ConfigItem) => void;
  onView: (item: ConfigItem) => void;
  onClearLocal: () => void;
}> = ({ item, selected, onSelect, onView, onClearLocal }) => (
  <li
    className={`flex items-center gap-3 px-3 py-2 transition-colors ${
      selected ? 'bg-violet-500/10' : 'hover:bg-white/5'
    }`}
  >
    <button
      type="button"
      onClick={() => onSelect(item)}
      className="flex-1 text-left"
      title={`Use ${item.name}`}
    >
      <div className="flex items-center gap-2">
        <span className="font-medium text-white">{item.name}</span>
        {sourceTagFor(item)}
        {item.kind !== 'local' && (
          <Tag colorScheme="purple" className="text-[10px]">
            {item.deviceCount} {item.deviceCount === 1 ? 'device' : 'devices'}
          </Tag>
        )}
        {item.kind === 'builtin' && (
          <Tag colorScheme="gray" className="text-[10px] capitalize">
            {item.template.type}
          </Tag>
        )}
      </div>
      {item.description && (
        <SmallText
          className={`mt-0.5 line-clamp-1 text-gray-400 ${
            item.kind === 'saved' ? 'font-mono text-[11px] text-gray-500' : ''
          }`}
        >
          {item.description}
        </SmallText>
      )}
    </button>
    {item.kind === 'builtin' && (
      <button
        type="button"
        onClick={() => onView(item)}
        className="rounded p-1.5 text-gray-400 hover:bg-white/10 hover:text-white"
        title="Preview the template YAML"
      >
        <Eye className="h-4 w-4" />
      </button>
    )}
    {item.kind === 'local' && (
      <button
        type="button"
        onClick={onClearLocal}
        className="text-xs font-medium text-red-300 hover:text-red-200"
        title="Drop the local file"
      >
        Clear
      </button>
    )}
  </li>
);
