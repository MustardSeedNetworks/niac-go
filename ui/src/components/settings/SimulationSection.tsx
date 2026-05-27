/**
 * SimulationSection Component
 *
 * Settings section for configuring simulation parameters.
 * Allows selecting network interface and configuration source.
 *
 * Features:
 * - Interface selector (filtered to usable: eth, wifi, loopback)
 * - Config source tabs (Templates / My Configs / Upload)
 * - Template picker with search
 * - User config picker
 * - File upload for quick config override
 */

import { FileUp, FolderOpen, LayoutTemplate, Network, PlugZap } from 'lucide-react';
import type { ReactElement } from 'react';
import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { fetchTemplates, fetchUsableInterfaces, fetchUserConfigs } from '../../api/client';
import type { NetworkInterface, Template, UserConfig } from '../../api/types';
import { type ConfigSource, useUIStore } from '../../stores/ui-store';
import { cn } from '../../styles/theme';

type ConfigTab = 'templates' | 'configs' | 'upload';

interface ConfigTabButton {
  id: ConfigTab;
  // Label resolved per-render via t(`simulation.tab${id}`) so the static
  // array stays language-agnostic.
  labelKey: 'tabTemplates' | 'tabMyConfigs' | 'tabUpload';
  icon: ReactElement;
  source: ConfigSource;
}

const CONFIG_TABS: ConfigTabButton[] = [
  {
    id: 'templates',
    labelKey: 'tabTemplates',
    icon: <LayoutTemplate className="w-4 h-4" />,
    source: 'template',
  },
  {
    id: 'configs',
    labelKey: 'tabMyConfigs',
    icon: <FolderOpen className="w-4 h-4" />,
    source: 'userConfig',
  },
  {
    id: 'upload',
    labelKey: 'tabUpload',
    icon: <FileUp className="w-4 h-4" />,
    source: 'upload',
  },
];

export function SimulationSection(): ReactElement {
  const { t } = useTranslation('settings');
  const { simulationSettings, setSimulationSettings } = useUIStore();
  const [interfaces, setInterfaces] = useState<NetworkInterface[]>([]);
  const [templates, setTemplates] = useState<Template[]>([]);
  const [userConfigs, setUserConfigs] = useState<UserConfig[]>([]);
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState<ConfigTab>(() => {
    // Set initial tab based on current config source
    switch (simulationSettings.configSource) {
      case 'template':
        return 'templates';
      case 'userConfig':
        return 'configs';
      case 'upload':
        return 'upload';
      default:
        return 'templates';
    }
  });

  // Fetch data on mount
  useEffect(() => {
    const loadData = async () => {
      setLoading(true);
      try {
        const [interfacesResp, templatesResp, configsResp] = await Promise.all([
          fetchUsableInterfaces(),
          fetchTemplates(),
          fetchUserConfigs(),
        ]);
        setInterfaces(interfacesResp.interfaces);
        setTemplates(templatesResp);
        setUserConfigs(configsResp.configs);
      } catch (err) {
        console.error('Failed to load simulation settings data:', err);
      } finally {
        setLoading(false);
      }
    };

    loadData();
  }, []);

  const handleInterfaceChange = useCallback(
    (e: React.ChangeEvent<HTMLSelectElement>) => {
      setSimulationSettings({ selectedInterface: e.target.value });
    },
    [setSimulationSettings],
  );

  const handleTabChange = useCallback(
    (tab: ConfigTab) => {
      setActiveTab(tab);
      const tabConfig = CONFIG_TABS.find((t) => t.id === tab);
      if (tabConfig) {
        setSimulationSettings({ configSource: tabConfig.source });
      }
    },
    [setSimulationSettings],
  );

  const handleTemplateSelect = useCallback(
    (template: Template) => {
      setSimulationSettings({
        configSource: 'template',
        configName: template.name,
        configPath: undefined,
      });
    },
    [setSimulationSettings],
  );

  const handleUserConfigSelect = useCallback(
    (config: UserConfig) => {
      setSimulationSettings({
        configSource: 'userConfig',
        configName: config.name,
        configPath: config.path,
      });
    },
    [setSimulationSettings],
  );

  return (
    <div className="stack-lg">
      {/* Section Header */}
      <div className="flex items-center gap-compact">
        <PlugZap className="w-5 h-5 text-brand-accent" aria-hidden="true" />
        <h3 className="text-sm font-semibold text-text-primary">{t('simulation.sectionTitle')}</h3>
      </div>

      {/* Interface Selector */}
      <div className="stack-sm">
        <label htmlFor="sim-interface" className="block text-sm text-text-muted">
          {t('simulation.interfaceLabel')}
        </label>
        <div className="relative">
          <Network className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-text-muted" />
          <select
            id="sim-interface"
            value={simulationSettings.selectedInterface}
            onChange={handleInterfaceChange}
            disabled={loading}
            className={cn(
              'w-full pl-10 pr-4 py-row text-sm',
              'bg-bg-elevated border border-surface-border rounded-lg',
              'text-text-primary focus:outline-none focus:ring-2 focus:ring-brand-primary/50',
              'disabled:opacity-50 disabled:cursor-not-allowed',
            )}
          >
            <option value="">{t('simulation.interfacePlaceholder')}</option>
            {interfaces.map((iface) => (
              <option key={iface.name} value={iface.name}>
                {iface.name}
                {iface.addresses.length > 0 ? ` (${iface.addresses[0]})` : ''}
              </option>
            ))}
          </select>
        </div>
        <p className="text-xs text-text-muted">{t('simulation.interfaceHelper')}</p>
      </div>

      {/* Config Source Tabs */}
      <div className="stack">
        <span className="block text-sm text-text-muted">{t('simulation.configurationLabel')}</span>
        <div
          className="flex border border-surface-border rounded-lg overflow-hidden"
          role="tablist"
        >
          {CONFIG_TABS.map((tab) => (
            <button
              key={tab.id}
              type="button"
              onClick={() => handleTabChange(tab.id)}
              className={cn(
                'flex-1 flex-center gap-1.5 px-3 py-row text-xs font-medium',
                'transition-colors',
                activeTab === tab.id
                  ? 'bg-brand-primary text-text-primary'
                  : 'bg-bg-elevated text-text-muted hover:bg-bg-elevated hover:text-text-primary',
              )}
            >
              {tab.icon}
              <span className="hidden sm:inline">{t(`simulation.${tab.labelKey}`)}</span>
            </button>
          ))}
        </div>
      </div>

      {/* Tab Content */}
      <div className="min-h-[120px]">
        {loading && (
          <div className="flex-center py-8 text-text-muted text-sm">{t('simulation.loading')}</div>
        )}

        {!loading && activeTab === 'templates' && (
          <TemplateList
            templates={templates}
            selectedName={
              simulationSettings.configSource === 'template' ? simulationSettings.configName : ''
            }
            onSelect={handleTemplateSelect}
          />
        )}

        {!loading && activeTab === 'configs' && (
          <UserConfigList
            configs={userConfigs}
            selectedName={
              simulationSettings.configSource === 'userConfig' ? simulationSettings.configName : ''
            }
            onSelect={handleUserConfigSelect}
          />
        )}

        {!loading && activeTab === 'upload' && <UploadSection />}
      </div>

      {/* Current Selection Display */}
      {simulationSettings.configName && (
        <div className="pad-sm bg-brand-primary/20 border border-brand-primary/30 rounded-lg">
          <p className="text-xs text-text-muted">{t('simulation.selectedConfigLabel')}</p>
          <p className="text-sm text-text-primary font-medium mt-tight">
            {simulationSettings.configName}
            <span className="text-text-muted ml-inline">
              (
              {simulationSettings.configSource === 'template'
                ? t('simulation.selectedConfigSourceTemplate')
                : t('simulation.selectedConfigSourceUserConfig')}
              )
            </span>
          </p>
        </div>
      )}
    </div>
  );
}

// =============================================================================
// Sub-components
// =============================================================================

interface TemplateListProps {
  templates: Template[];
  selectedName: string;
  onSelect: (template: Template) => void;
}

function TemplateList({ templates, selectedName, onSelect }: TemplateListProps): ReactElement {
  const { t } = useTranslation('settings');
  const [search, setSearch] = useState('');

  const filteredTemplates = templates.filter(
    (tpl) =>
      tpl.name.toLowerCase().includes(search.toLowerCase()) ||
      tpl.description.toLowerCase().includes(search.toLowerCase()),
  );

  return (
    <div className="stack-sm">
      <input
        type="text"
        placeholder={t('simulation.searchTemplatesPlaceholder')}
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        className={cn(
          'w-full px-3 py-row text-sm',
          'bg-bg-elevated border border-surface-border rounded-lg',
          'text-text-primary placeholder:text-text-muted',
          'focus:outline-none focus:ring-2 focus:ring-brand-primary/50',
        )}
      />
      <div className="max-h-[200px] overflow-y-auto stack-xs">
        {filteredTemplates.length === 0 && (
          <p className="text-sm text-text-muted py-4 text-center">
            {search ? t('simulation.noTemplatesMatchSearch') : t('simulation.noTemplatesAvailable')}
          </p>
        )}
        {filteredTemplates.map((template) => (
          <button
            key={template.name}
            type="button"
            onClick={() => onSelect(template)}
            className={cn(
              'w-full text-left px-3 py-row rounded-lg transition-colors',
              selectedName === template.name
                ? 'bg-brand-primary/30 border border-brand-primary/50'
                : 'bg-surface-hover hover:bg-surface-hover border border-transparent',
            )}
          >
            <div className="text-sm text-text-primary font-medium">{template.name}</div>
            <div className="text-xs text-text-muted truncate">{template.description}</div>
            <div className="text-xs text-text-muted mt-tight">
              {t('simulation.deviceCount', { count: template.deviceCount })}
            </div>
          </button>
        ))}
      </div>
    </div>
  );
}

interface UserConfigListProps {
  configs: UserConfig[];
  selectedName: string;
  onSelect: (config: UserConfig) => void;
}

function UserConfigList({ configs, selectedName, onSelect }: UserConfigListProps): ReactElement {
  const { t } = useTranslation('settings');
  if (configs.length === 0) {
    return (
      <div className="text-center py-8">
        <FolderOpen className="w-8 h-8 text-text-disabled mx-auto mb-2" />
        <p className="text-sm text-text-muted">{t('simulation.noUserConfigs')}</p>
        <p className="text-xs text-text-disabled mt-tight">{t('simulation.noUserConfigsHint')}</p>
      </div>
    );
  }

  return (
    <div className="max-h-[200px] overflow-y-auto stack-xs">
      {configs.map((config) => (
        <button
          key={config.name}
          type="button"
          onClick={() => onSelect(config)}
          className={cn(
            'w-full text-left px-3 py-row rounded-lg transition-colors',
            selectedName === config.name
              ? 'bg-brand-primary/30 border border-brand-primary/50'
              : 'bg-surface-hover hover:bg-surface-hover border border-transparent',
          )}
        >
          <div className="text-sm text-text-primary font-medium">{config.name}</div>
          <div className="text-xs text-text-muted">
            {t('simulation.deviceCount', { count: config.deviceCount })}
          </div>
        </button>
      ))}
    </div>
  );
}

function UploadSection(): ReactElement {
  const { t } = useTranslation('settings');
  const { setSimulationSettings } = useUIStore();
  const [selectedFile, setSelectedFile] = useState<File | null>(null);

  const handleFileChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const file = e.target.files?.[0];
      if (!file) {
        setSelectedFile(null);
        return;
      }

      // Validate file type
      if (!file.name.match(/\.(yaml|yml)$/i)) {
        alert(t('simulation.alertInvalidFile'));
        e.target.value = '';
        return;
      }

      // Validate file size (10MB max)
      const maxSize = 10 * 1024 * 1024;
      if (file.size > maxSize) {
        alert(t('simulation.alertFileTooLarge'));
        e.target.value = '';
        return;
      }

      setSelectedFile(file);
      setSimulationSettings({
        configSource: 'upload',
        configName: file.name.replace(/\.(yaml|yml)$/i, ''),
        configPath: undefined,
      });
    },
    [setSimulationSettings, t],
  );

  return (
    <div className="stack">
      <div
        className={cn(
          'border-2 border-dashed border-surface-border rounded-lg pad',
          'hover:border-brand-primary/50 transition-colors',
        )}
      >
        <input
          type="file"
          id="config-upload"
          accept=".yaml,.yml"
          onChange={handleFileChange}
          className="sr-only"
        />
        <label htmlFor="config-upload" className="flex flex-col items-center cursor-pointer">
          <FileUp className="w-8 h-8 text-text-muted mb-2" />
          <span className="text-sm text-text-muted">
            {selectedFile ? selectedFile.name : t('simulation.uploadPrompt')}
          </span>
          <span className="text-xs text-text-disabled mt-tight">
            {t('simulation.uploadFileTypes')}
          </span>
        </label>
      </div>
      <p className="text-xs text-text-muted">{t('simulation.uploadHelper')}</p>
    </div>
  );
}

export default SimulationSection;
