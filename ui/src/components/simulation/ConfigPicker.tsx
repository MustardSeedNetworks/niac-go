import { Eye, FileCode, FileUp, FolderOpen, Search } from 'lucide-react';
import { type FC, useEffect, useMemo, useState } from 'react';
import { fetchTemplateContent, fetchTemplates, fetchUserConfigs } from '../../api/client';
import type { Template, TemplateContent, UserConfig } from '../../api/types';
import { Tag } from '../../ui/Tag';
import { SmallText } from '../../ui/Typography';
import { TemplatePreviewModal } from '../TemplatePreviewModal';
import { JavaDslImportCard } from '../templates/JavaDslImportCard';

type Tab = 'templates' | 'configs' | 'upload';

interface Selection {
  source: 'template' | 'userConfig' | 'upload' | null;
  name: string;
  path?: string;
}

export interface ConfigPickerProps {
  /** The currently selected config (template name, user config name, or null). */
  selection: Selection;
  /** Called when the user picks a template. */
  onSelectTemplate: (template: Template) => void;
  /** Called when the user picks a saved (user) config. */
  onSelectUserConfig: (config: UserConfig) => void;
  /** Called when the user uploads a file (or clears it). */
  onUpload: (file: File | null) => void;
  /** The current upload file, if any. */
  uploadFile: File | null;
}

/**
 * ConfigPicker is the tabbed config-source picker that lives inline on the
 * Simulation Control page. It collapses what used to be three separate
 * surfaces (Templates page, Settings drawer, RuntimeControlPage's "Quick
 * Override Upload") into one source-of-truth chooser.
 */
export const ConfigPicker: FC<ConfigPickerProps> = ({
  selection,
  onSelectTemplate,
  onSelectUserConfig,
  onUpload,
  uploadFile,
}) => {
  const initialTab: Tab =
    selection.source === 'userConfig'
      ? 'configs'
      : selection.source === 'upload'
        ? 'upload'
        : 'templates';
  const [tab, setTab] = useState<Tab>(initialTab);
  const [templates, setTemplates] = useState<Template[]>([]);
  const [userConfigs, setUserConfigs] = useState<UserConfig[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');

  // Preview modal state
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
        // Best-effort hydration; the empty states below already cover the
        // failure case.
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const filteredTemplates = useMemo(() => {
    const q = search.trim().toLowerCase();
    if (!q) return templates;
    return templates.filter(
      (t) =>
        t.name.toLowerCase().includes(q) ||
        t.description.toLowerCase().includes(q) ||
        t.type.toLowerCase().includes(q),
    );
  }, [templates, search]);

  const filteredConfigs = useMemo(() => {
    const q = search.trim().toLowerCase();
    if (!q) return userConfigs;
    return userConfigs.filter((c) => c.name.toLowerCase().includes(q));
  }, [userConfigs, search]);

  const openPreview = async (template: Template) => {
    setPreviewTemplate(template);
    setPreviewContent(null);
    setPreviewError(null);
    setPreviewLoading(true);
    try {
      const content = await fetchTemplateContent(template.name);
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

  return (
    <div className="space-y-3">
      {/* Tabs */}
      <div
        className="flex gap-1 rounded-lg border border-white/10 bg-gray-900/60 p-1"
        role="tablist"
      >
        <TabButton
          active={tab === 'templates'}
          onClick={() => setTab('templates')}
          icon={<FileCode className="h-4 w-4" />}
          label="Templates"
          count={templates.length}
        />
        <TabButton
          active={tab === 'configs'}
          onClick={() => setTab('configs')}
          icon={<FolderOpen className="h-4 w-4" />}
          label="My Configs"
          count={userConfigs.length}
        />
        <TabButton
          active={tab === 'upload'}
          onClick={() => setTab('upload')}
          icon={<FileUp className="h-4 w-4" />}
          label="Upload"
        />
      </div>

      {/* Search bar (templates / configs only) */}
      {tab !== 'upload' && (
        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-500" />
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder={tab === 'templates' ? 'Search templates…' : 'Search configs…'}
            className="w-full rounded border border-white/10 bg-gray-900/60 py-2 pl-9 pr-3 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none"
          />
        </div>
      )}

      {/* Tab content */}
      {tab === 'templates' && (
        <TemplatesTab
          templates={filteredTemplates}
          loading={loading}
          selectedName={selection.source === 'template' ? selection.name : ''}
          onSelect={onSelectTemplate}
          onView={openPreview}
        />
      )}

      {tab === 'configs' && (
        <ConfigsTab
          configs={filteredConfigs}
          loading={loading}
          selectedName={selection.source === 'userConfig' ? selection.name : ''}
          onSelect={onSelectUserConfig}
        />
      )}

      {tab === 'upload' && <UploadTab uploadFile={uploadFile} onUpload={onUpload} />}

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
            // Copy is a no-op — TemplatePreviewModal expects a Promise<void>;
            // it already handles its own clipboard plumbing via the parent.
          }}
        />
      )}
    </div>
  );
};

const TabButton: FC<{
  active: boolean;
  onClick: () => void;
  icon: React.ReactNode;
  label: string;
  count?: number;
}> = ({ active, onClick, icon, label, count }) => (
  <button
    type="button"
    role="tab"
    aria-selected={active}
    onClick={onClick}
    className={`flex flex-1 items-center justify-center gap-2 rounded px-3 py-1.5 text-sm transition-colors ${
      active
        ? 'bg-violet-500/20 text-violet-100 ring-1 ring-violet-400/40'
        : 'text-gray-400 hover:bg-white/5 hover:text-gray-200'
    }`}
  >
    {icon}
    <span>{label}</span>
    {typeof count === 'number' && (
      <span className="rounded bg-white/10 px-1.5 py-0.5 text-[10px] font-medium text-gray-300">
        {count}
      </span>
    )}
  </button>
);

const TemplatesTab: FC<{
  templates: Template[];
  loading: boolean;
  selectedName: string;
  onSelect: (t: Template) => void;
  onView: (t: Template) => void;
}> = ({ templates, loading, selectedName, onSelect, onView }) => {
  if (loading) {
    return <SmallText className="text-gray-500">Loading templates…</SmallText>;
  }
  if (templates.length === 0) {
    return (
      <SmallText className="text-gray-500">
        No templates match. Try a different search or upload a YAML in the Upload tab.
      </SmallText>
    );
  }
  return (
    <ul className="max-h-72 divide-y divide-white/5 overflow-y-auto rounded-lg border border-white/10 bg-gray-950/40">
      {templates.map((t) => (
        <li
          key={t.name}
          className={`flex items-center gap-3 px-3 py-2 transition-colors ${
            selectedName === t.name ? 'bg-violet-500/10' : 'hover:bg-white/5'
          }`}
        >
          <button
            type="button"
            onClick={() => onSelect(t)}
            className="flex-1 text-left"
            title={`Use ${t.name} as the simulation config`}
          >
            <div className="flex items-center gap-2">
              <span className="font-medium text-white">{t.name}</span>
              <Tag colorScheme="purple" className="text-[10px]">
                {t.deviceCount} {t.deviceCount === 1 ? 'device' : 'devices'}
              </Tag>
              <Tag colorScheme="gray" className="text-[10px] capitalize">
                {t.type}
              </Tag>
            </div>
            {t.description && (
              <SmallText className="mt-0.5 text-gray-400 line-clamp-1">{t.description}</SmallText>
            )}
          </button>
          <button
            type="button"
            onClick={() => onView(t)}
            className="rounded p-1.5 text-gray-400 hover:bg-white/10 hover:text-white"
            title="Preview the template YAML"
          >
            <Eye className="h-4 w-4" />
          </button>
        </li>
      ))}
    </ul>
  );
};

const ConfigsTab: FC<{
  configs: UserConfig[];
  loading: boolean;
  selectedName: string;
  onSelect: (c: UserConfig) => void;
}> = ({ configs, loading, selectedName, onSelect }) => {
  if (loading) {
    return <SmallText className="text-gray-500">Loading configs…</SmallText>;
  }
  if (configs.length === 0) {
    return (
      <SmallText className="text-gray-500">
        No saved configs yet. Pick a template above (it'll be saved here) or upload a YAML.
      </SmallText>
    );
  }
  return (
    <ul className="max-h-72 divide-y divide-white/5 overflow-y-auto rounded-lg border border-white/10 bg-gray-950/40">
      {configs.map((c) => (
        <li
          key={c.path}
          className={`flex items-center gap-3 px-3 py-2 transition-colors ${
            selectedName === c.name ? 'bg-violet-500/10' : 'hover:bg-white/5'
          }`}
        >
          <button type="button" onClick={() => onSelect(c)} className="flex-1 text-left">
            <div className="flex items-center gap-2">
              <span className="font-medium text-white">{c.name}</span>
              <Tag colorScheme="purple" className="text-[10px]">
                {c.deviceCount} {c.deviceCount === 1 ? 'device' : 'devices'}
              </Tag>
            </div>
            <SmallText className="mt-0.5 font-mono text-[11px] text-gray-500 line-clamp-1">
              {c.path}
            </SmallText>
          </button>
        </li>
      ))}
    </ul>
  );
};

const UploadTab: FC<{
  uploadFile: File | null;
  onUpload: (file: File | null) => void;
}> = ({ uploadFile, onUpload }) => {
  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0] ?? null;
    onUpload(file);
  };

  return (
    <div className="space-y-3">
      <div className="rounded-lg border border-dashed border-white/15 bg-gray-950/40 p-4">
        <div className="flex items-center gap-3">
          <label
            htmlFor="config-upload"
            className="flex cursor-pointer items-center gap-2 rounded bg-gray-800 px-4 py-2 text-sm text-white hover:bg-gray-700"
          >
            <FileUp className="h-4 w-4 text-gray-400" />
            {uploadFile ? 'Change YAML' : 'Choose YAML…'}
          </label>
          <input
            id="config-upload"
            type="file"
            accept=".yaml,.yml"
            onChange={handleChange}
            className="sr-only"
          />
          {uploadFile && (
            <>
              <span className="text-sm text-gray-300">{uploadFile.name}</span>
              <button
                type="button"
                onClick={() => onUpload(null)}
                className="text-sm text-red-400 hover:text-red-300"
              >
                Clear
              </button>
            </>
          )}
        </div>
        <SmallText className="mt-2 text-gray-500">
          Upload a YAML config to use for this simulation run. The file is sent to the daemon
          inline; it isn't saved to your config library.
        </SmallText>
      </div>

      {/* Java DSL import — also writes a saved config the daemon can run */}
      <JavaDslImportCard />
    </div>
  );
};
