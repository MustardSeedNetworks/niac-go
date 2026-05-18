// Copyright (c) 2025 Mustard Seed Networks. All rights reserved.

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
import { fetchTemplates, fetchUsableInterfaces, fetchUserConfigs } from '../../api/client';
import type { NetworkInterface, Template, UserConfig } from '../../api/types';
import { type ConfigSource, useUIStore } from '../../stores/ui-store';
import { cn } from '../../styles/theme';

type ConfigTab = 'templates' | 'configs' | 'upload';

interface ConfigTabButton {
  id: ConfigTab;
  label: string;
  icon: ReactElement;
  source: ConfigSource;
}

const CONFIG_TABS: ConfigTabButton[] = [
  {
    id: 'templates',
    label: 'Templates',
    icon: <LayoutTemplate className="w-4 h-4" />,
    source: 'template',
  },
  {
    id: 'configs',
    label: 'My Configs',
    icon: <FolderOpen className="w-4 h-4" />,
    source: 'userConfig',
  },
  { id: 'upload', label: 'Upload', icon: <FileUp className="w-4 h-4" />, source: 'upload' },
];

export function SimulationSection(): ReactElement {
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
    <div className="space-y-4">
      {/* Section Header */}
      <div className="flex items-center gap-2">
        <PlugZap className="w-5 h-5 text-brand-400" aria-hidden="true" />
        <h3 className="text-sm font-semibold text-text-primary">Simulation</h3>
      </div>

      {/* Interface Selector */}
      <div className="space-y-2">
        <label htmlFor="sim-interface" className="block text-sm text-text-muted">
          Network Interface
        </label>
        <div className="relative">
          <Network className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-text-muted" />
          <select
            id="sim-interface"
            value={simulationSettings.selectedInterface}
            onChange={handleInterfaceChange}
            disabled={loading}
            className={cn(
              'w-full pl-10 pr-4 py-2 text-sm',
              'bg-bg-elevated border border-white/10 rounded-lg',
              'text-text-primary focus:outline-none focus:ring-2 focus:ring-brand-500/50',
              'disabled:opacity-50 disabled:cursor-not-allowed',
            )}
          >
            <option value="">Select interface...</option>
            {interfaces.map((iface) => (
              <option key={iface.name} value={iface.name}>
                {iface.name}
                {iface.addresses.length > 0 ? ` (${iface.addresses[0]})` : ''}
              </option>
            ))}
          </select>
        </div>
        <p className="text-xs text-text-muted">
          Only ethernet, WiFi, and loopback interfaces are shown.
        </p>
      </div>

      {/* Config Source Tabs */}
      <div className="space-y-3">
        <span className="block text-sm text-text-muted">Configuration</span>
        <div className="flex border border-white/10 rounded-lg overflow-hidden" role="tablist">
          {CONFIG_TABS.map((tab) => (
            <button
              key={tab.id}
              type="button"
              onClick={() => handleTabChange(tab.id)}
              className={cn(
                'flex-1 flex items-center justify-center gap-1.5 px-3 py-2 text-xs font-medium',
                'transition-colors',
                activeTab === tab.id
                  ? 'bg-brand-600 text-text-primary'
                  : 'bg-bg-elevated text-text-muted hover:bg-bg-elevated hover:text-text-primary',
              )}
            >
              {tab.icon}
              <span className="hidden sm:inline">{tab.label}</span>
            </button>
          ))}
        </div>
      </div>

      {/* Tab Content */}
      <div className="min-h-[120px]">
        {loading && (
          <div className="flex items-center justify-center py-8 text-text-muted text-sm">
            Loading...
          </div>
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
        <div className="p-3 bg-brand-900/20 border border-brand-500/30 rounded-lg">
          <p className="text-xs text-text-muted">Selected configuration:</p>
          <p className="text-sm text-text-primary font-medium mt-1">
            {simulationSettings.configName}
            <span className="text-text-muted ml-2">
              ({simulationSettings.configSource === 'template' ? 'Template' : 'User Config'})
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
  const [search, setSearch] = useState('');

  const filteredTemplates = templates.filter(
    (t) =>
      t.name.toLowerCase().includes(search.toLowerCase()) ||
      t.description.toLowerCase().includes(search.toLowerCase()),
  );

  return (
    <div className="space-y-2">
      <input
        type="text"
        placeholder="Search templates..."
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        className={cn(
          'w-full px-3 py-2 text-sm',
          'bg-bg-elevated border border-white/10 rounded-lg',
          'text-text-primary placeholder:text-text-muted',
          'focus:outline-none focus:ring-2 focus:ring-brand-500/50',
        )}
      />
      <div className="max-h-[200px] overflow-y-auto space-y-1">
        {filteredTemplates.length === 0 && (
          <p className="text-sm text-text-muted py-4 text-center">
            {search ? 'No templates match your search' : 'No templates available'}
          </p>
        )}
        {filteredTemplates.map((template) => (
          <button
            key={template.name}
            type="button"
            onClick={() => onSelect(template)}
            className={cn(
              'w-full text-left px-3 py-2 rounded-lg transition-colors',
              selectedName === template.name
                ? 'bg-brand-600/30 border border-brand-500/50'
                : 'bg-white/5 hover:bg-white/10 border border-transparent',
            )}
          >
            <div className="text-sm text-text-primary font-medium">{template.name}</div>
            <div className="text-xs text-text-muted truncate">{template.description}</div>
            <div className="text-xs text-text-muted mt-1">
              {template.deviceCount} device{template.deviceCount !== 1 ? 's' : ''}
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
  if (configs.length === 0) {
    return (
      <div className="text-center py-8">
        <FolderOpen className="w-8 h-8 text-text-disabled mx-auto mb-2" />
        <p className="text-sm text-text-muted">No user configs uploaded yet</p>
        <p className="text-xs text-text-disabled mt-1">
          Use the Upload tab to add your own configs
        </p>
      </div>
    );
  }

  return (
    <div className="max-h-[200px] overflow-y-auto space-y-1">
      {configs.map((config) => (
        <button
          key={config.name}
          type="button"
          onClick={() => onSelect(config)}
          className={cn(
            'w-full text-left px-3 py-2 rounded-lg transition-colors',
            selectedName === config.name
              ? 'bg-brand-600/30 border border-brand-500/50'
              : 'bg-white/5 hover:bg-white/10 border border-transparent',
          )}
        >
          <div className="text-sm text-text-primary font-medium">{config.name}</div>
          <div className="text-xs text-text-muted">
            {config.deviceCount} device{config.deviceCount !== 1 ? 's' : ''}
          </div>
        </button>
      ))}
    </div>
  );
}

function UploadSection(): ReactElement {
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
        alert('Please select a YAML file (.yaml or .yml)');
        e.target.value = '';
        return;
      }

      // Validate file size (10MB max)
      const maxSize = 10 * 1024 * 1024;
      if (file.size > maxSize) {
        alert('File too large. Maximum size is 10MB');
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
    [setSimulationSettings],
  );

  return (
    <div className="space-y-3">
      <div
        className={cn(
          'border-2 border-dashed border-white/10 rounded-lg p-4',
          'hover:border-brand-500/50 transition-colors',
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
            {selectedFile ? selectedFile.name : 'Click to upload a YAML config'}
          </span>
          <span className="text-xs text-text-disabled mt-1">.yaml or .yml files, max 10MB</span>
        </label>
      </div>
      <p className="text-xs text-text-muted">
        Upload a config file to use for this simulation session. The file will be sent when you
        start the simulation.
      </p>
    </div>
  );
}

export default SimulationSection;
