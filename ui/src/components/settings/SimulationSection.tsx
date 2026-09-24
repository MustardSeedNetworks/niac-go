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

import { AlertCircle, FileUp, FolderOpen, LayoutTemplate, PlugZap } from 'lucide-react';
import type { ReactElement } from 'react';
import { useCallback, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { fetchTemplates } from '../../api/client';
import { fetchLibraryNetworks } from '../../api/library-client';
import type { LibraryNetwork, Template } from '../../api/types';
import { useApiResource } from '../../hooks/useApiResource';
import { useUsableInterfacesResource } from '../../hooks/usePageResources';
import { type ConfigSource, useUIStore } from '../../stores/ui-store';
import { cn } from '../../styles/theme';
import { Select } from '../../ui/Input';
import { getErrorMessage } from '../../utils/format';

type ConfigTab = 'templates' | 'configs' | 'upload';

const tabId = (tab: ConfigTab) => `sim-config-tab-${tab}`;
const panelId = 'sim-config-panel';

interface ConfigTabButton {
  id: ConfigTab;
  labelKey: 'simulation.tabTemplates' | 'simulation.tabMyConfigs' | 'simulation.tabUpload';
  icon: ReactElement;
  source: ConfigSource;
}

const CONFIG_TABS: ConfigTabButton[] = [
  {
    id: 'templates',
    labelKey: 'simulation.tabTemplates',
    icon: <LayoutTemplate className="w-4 h-4" />,
    source: 'template',
  },
  {
    id: 'configs',
    labelKey: 'simulation.tabMyConfigs',
    icon: <FolderOpen className="w-4 h-4" />,
    source: 'userConfig',
  },
  {
    id: 'upload',
    labelKey: 'simulation.tabUpload',
    icon: <FileUp className="w-4 h-4" />,
    source: 'upload',
  },
];

export function SimulationSection(): ReactElement {
  const { t } = useTranslation('settings');
  const { simulationSettings, setSimulationSettings } = useUIStore();
  // Three independent resources, not one `Promise.all`: a failure of any one
  // of them used to blank all three lists and read as "you have none" (#2177).
  const {
    data: interfacesData,
    loading: interfacesLoading,
    error: interfacesError,
    refetch: refetchInterfaces,
  } = useUsableInterfacesResource();
  const {
    data: templates,
    loading: templatesLoading,
    error: templatesError,
    refetch: refetchTemplates,
  } = useApiResource(fetchTemplates, ['templates']);
  const {
    data: userConfigs,
    loading: userConfigsLoading,
    error: userConfigsError,
    refetch: refetchUserConfigs,
  } = useApiResource(fetchLibraryNetworks, ['library', 'networks']);
  const interfaces = interfacesData?.interfaces ?? [];
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

  // Automatic activation is immediate: switching tabs does not initiate a fetch.
  const handleTabKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLButtonElement>) => {
      const ids = CONFIG_TABS.map((tab) => tab.id);
      const current = ids.indexOf(activeTab);
      const target = {
        ArrowRight: ids[(current + 1) % ids.length],
        ArrowLeft: ids[(current - 1 + ids.length) % ids.length],
        Home: ids[0],
        End: ids[ids.length - 1],
      }[e.key];
      if (target === undefined) {
        return;
      }
      e.preventDefault();
      handleTabChange(target);
      document.getElementById(tabId(target))?.focus();
    },
    [activeTab, handleTabChange],
  );

  const handleTemplateSelect = useCallback(
    (template: Template) => {
      setSimulationSettings({
        configSource: 'template',
        configName: template.name,
      });
    },
    [setSimulationSettings],
  );

  const handleUserConfigSelect = useCallback(
    (config: LibraryNetwork) => {
      setSimulationSettings({
        configSource: 'userConfig',
        configName: config.name,
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
        <Select
          id="sim-interface"
          data-testid="simulation-interface"
          value={simulationSettings.selectedInterface}
          onChange={(selectedInterface) => setSimulationSettings({ selectedInterface })}
          disabled={interfacesLoading}
          className="min-h-11 text-sm"
          options={[
            { value: '', label: t('simulation.interfacePlaceholder') },
            ...interfaces.map((iface) => ({
              value: iface.name,
              label: `${iface.name}${iface.addresses.length > 0 ? ` (${iface.addresses[0]})` : ''}`,
            })),
          ]}
        />
        {interfacesError ? (
          <LoadError
            testId="simulation-interfaces"
            message={t('simulation.loadFailedInterfaces', {
              error: getErrorMessage(interfacesError),
            })}
            onRetry={refetchInterfaces}
          />
        ) : (
          <p className="text-xs text-text-muted">{t('simulation.interfaceHelper')}</p>
        )}
      </div>

      {/* Config Source Tabs */}
      <div className="stack">
        <span id="sim-config-label" className="block text-sm text-text-muted">
          {t('simulation.configurationLabel')}
        </span>
        <div
          className="flex border border-surface-border rounded-lg overflow-hidden"
          role="tablist"
          aria-labelledby="sim-config-label"
        >
          {CONFIG_TABS.map((tab) => (
            <button
              key={tab.id}
              type="button"
              role="tab"
              id={tabId(tab.id)}
              aria-selected={activeTab === tab.id}
              aria-controls={panelId}
              tabIndex={activeTab === tab.id ? 0 : -1}
              onClick={() => handleTabChange(tab.id)}
              onKeyDown={handleTabKeyDown}
              className={cn(
                'flex-1 flex-center gap-1.5 px-3 py-row text-xs font-medium',
                'transition-colors',
                activeTab === tab.id
                  ? 'bg-brand-primary text-on-brand'
                  : 'bg-bg-elevated text-text-muted hover:bg-bg-elevated hover:text-text-primary',
              )}
            >
              {tab.icon}
              <span className="sr-only sm:not-sr-only">{t(tab.labelKey)}</span>
            </button>
          ))}
        </div>
      </div>

      {/* Tab Content */}
      <div
        className="min-h-[120px]"
        role="tabpanel"
        // biome-ignore lint/a11y/noNoninteractiveTabindex: Owner-approved; empty panels need a keyboard entry point per https://www.w3.org/WAI/ARIA/apg/patterns/tabs/
        tabIndex={0}
        id={panelId}
        aria-labelledby={tabId(activeTab)}
      >
        {activeTab === 'templates' &&
          (templatesLoading ? (
            <Loading />
          ) : templatesError ? (
            <LoadError
              testId="simulation-templates"
              message={t('simulation.loadFailedTemplates', {
                error: getErrorMessage(templatesError),
              })}
              onRetry={refetchTemplates}
            />
          ) : (
            <TemplateList
              templates={templates ?? []}
              selectedName={
                simulationSettings.configSource === 'template' ? simulationSettings.configName : ''
              }
              onSelect={handleTemplateSelect}
            />
          ))}

        {activeTab === 'configs' &&
          (userConfigsLoading ? (
            <Loading />
          ) : userConfigsError ? (
            <LoadError
              testId="simulation-configs"
              message={t('simulation.loadFailedUserConfigs', {
                error: getErrorMessage(userConfigsError),
              })}
              onRetry={refetchUserConfigs}
            />
          ) : (
            <UserConfigList
              configs={userConfigs ?? []}
              selectedName={
                simulationSettings.configSource === 'userConfig'
                  ? simulationSettings.configName
                  : ''
              }
              onSelect={handleUserConfigSelect}
            />
          ))}

        {activeTab === 'upload' && <UploadSection />}
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

function Loading(): ReactElement {
  const { t } = useTranslation('settings');
  return <div className="flex-center py-8 text-text-muted text-sm">{t('simulation.loading')}</div>;
}

interface LoadErrorProps {
  testId: string;
  message: string;
  onRetry: () => void;
}

/** A load failure states that it failed and offers the one action that helps. */
function LoadError({ testId, message, onRetry }: LoadErrorProps): ReactElement {
  const { t } = useTranslation('settings');
  return (
    <div
      role="alert"
      data-testid={`${testId}-error`}
      className="flex items-start gap-compact rounded-lg border border-status-error/30 bg-status-error/20 pad-sm"
    >
      <AlertCircle className="mt-tight w-4 h-4 shrink-0 text-status-error" aria-hidden="true" />
      <div className="stack-xs">
        <p className="text-sm text-status-error-strong">{message}</p>
        <button
          type="button"
          onClick={onRetry}
          data-testid={`${testId}-retry`}
          className="text-xs font-medium text-status-error-strong underline underline-offset-2"
        >
          {t('simulation.retry')}
        </button>
      </div>
    </div>
  );
}

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
  configs: LibraryNetwork[];
  selectedName: string;
  onSelect: (config: LibraryNetwork) => void;
}

function UserConfigList({ configs, selectedName, onSelect }: UserConfigListProps): ReactElement {
  const { t } = useTranslation('settings');
  if (configs.length === 0) {
    return (
      <div className="text-center py-8">
        <FolderOpen className="w-8 h-8 text-text-disabled mx-auto mb-2" />
        <p className="text-sm text-text-muted">{t('simulation.noUserConfigs')}</p>
        <p className="text-xs text-text-muted mt-tight">{t('simulation.noUserConfigsHint')}</p>
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
          <span className="text-xs text-text-muted mt-tight">
            {t('simulation.uploadFileTypes')}
          </span>
        </label>
      </div>
      <p className="text-xs text-text-muted">{t('simulation.uploadHelper')}</p>
    </div>
  );
}

export default SimulationSection;
