// Copyright (c) 2025 Mustard Seed Networks. All rights reserved.

/**
 * SettingsDrawer Component
 *
 * Main configuration panel for NIAC.
 *
 * Features:
 * - Appearance settings (theme, language)
 * - Network interface display
 * - Debug logging controls
 * - About/version information
 *
 * The frame is the shared RightDrawer and the sections are SettingsTabs.
 */

import {
  Bug,
  ChevronRight,
  Info,
  Monitor,
  Moon,
  Network,
  Palette,
  PlugZap,
  Sun,
} from 'lucide-react';
import type { ReactElement, ReactNode } from 'react';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router';
import { fetchInterfaces } from '../api/client';
import type { NetworkInterface } from '../api/types';
import type { ConnectionState } from '../hooks/useConnectionStatus';
import { type Theme, useTheme } from '../hooks/useTheme';
import { languages } from '../i18n';
import { FormField } from '../shared/FormField';
import { RightDrawer } from '../shared/RightDrawer';
import { SegmentedControl } from '../shared/SegmentedControl';
import { Select } from '../shared/Select';
import { SettingsTabs } from '../shared/SettingsTabs';
import { Skeleton } from '../shared/Skeleton';
import { cn, layout } from '../styles/theme';
import { ConnectionStatus } from '../ui/ConnectionStatus';
import { SimulationSection } from './settings/SimulationSection';

interface SettingsDrawerProps {
  connectionStatus: ConnectionState;
  isOpen: boolean;
  onClose: () => void;
  version?: string;
}

type SettingsTab = 'simulation' | 'appearance' | 'network' | 'debug' | 'about';

interface TabConfig {
  id: SettingsTab;
  icon: ReactNode;
}

// Tab labels resolved per-render via the t() function below; declaring
// the static config without `label` keeps the array immutable and lets
// the i18n hook supply the locale-aware label.
const TABS: TabConfig[] = [
  { id: 'simulation', icon: <PlugZap className="w-4 h-4" /> },
  { id: 'appearance', icon: <Palette className="w-4 h-4" /> },
  { id: 'network', icon: <Network className="w-4 h-4" /> },
  { id: 'debug', icon: <Bug className="w-4 h-4" /> },
  { id: 'about', icon: <Info className="w-4 h-4" /> },
];

export function SettingsDrawer({
  connectionStatus,
  isOpen,
  onClose,
  version = '0.0.0',
}: SettingsDrawerProps): ReactElement {
  const { t } = useTranslation('settings');
  const [activeTab, setActiveTab] = useState<SettingsTab>('simulation');

  return (
    <RightDrawer
      open={isOpen}
      onClose={onClose}
      title={t('drawer.title')}
      closeLabel={t('drawer.closeAriaLabel')}
      data-testid="settings-drawer"
    >
      <SettingsTabs
        tabs={TABS.map((tab) => ({ id: tab.id, label: t(`tabs.${tab.id}`), icon: tab.icon }))}
        value={activeTab}
        onChange={(id) => {
          const tab = TABS.find((candidate) => candidate.id === id);
          if (tab) setActiveTab(tab.id);
        }}
        label={t('drawer.tabsLabel')}
      >
        <div className="stack-xl">
          {activeTab === 'simulation' && <SimulationSection />}
          {activeTab === 'appearance' && <AppearanceSection />}
          {activeTab === 'network' && <NetworkSection status={connectionStatus} />}
          {activeTab === 'debug' && <DebugSection onClose={onClose} />}
          {activeTab === 'about' && <AboutSection version={version} />}
        </div>
      </SettingsTabs>
    </RightDrawer>
  );
}

// =============================================================================
// Section Components
// =============================================================================

interface SectionProps {
  title: string;
  description?: string;
  children: ReactNode;
}

const Section = ({ title, description, children }: SectionProps): ReactElement => (
  <div className="stack">
    <div>
      <h3 className="text-sm font-semibold text-text-primary">{title}</h3>
      {description && <p className="text-xs text-text-muted mt-0.5">{description}</p>}
    </div>
    <div className="stack-sm">{children}</div>
  </div>
);

interface SettingRowProps {
  label: string;
  description?: string;
  children: ReactNode;
}

const SettingRow = ({ label, description, children }: SettingRowProps): ReactElement => (
  <div className="flex-between gap-comfortable py-row px-3 bg-surface-hover rounded-lg">
    <div className="flex-1 min-w-0">
      <div className="text-sm text-text-primary">{label}</div>
      {description && <div className="text-xs text-text-muted truncate">{description}</div>}
    </div>
    <div className="flex-shrink-0">{children}</div>
  </div>
);

// =============================================================================
// Appearance Section
// =============================================================================

function AppearanceSection(): ReactElement {
  const { t, i18n } = useTranslation('settings');
  // 'en-US' from the browser detector resolves to the supported 'en'.
  const activeLanguage = languages.find(
    (language) => language.code === i18n.language.split('-')[0],
  )?.code;
  const { theme, setTheme } = useTheme();

  const options: Array<{ value: Theme; label: string; icon: ReactNode }> = [
    { value: 'system', label: t('appearance.system'), icon: <Monitor className="h-4 w-4" /> },
    { value: 'light', label: t('appearance.light'), icon: <Sun className="h-4 w-4" /> },
    { value: 'dark', label: t('appearance.dark'), icon: <Moon className="h-4 w-4" /> },
  ];

  return (
    <Section title={t('appearance.sectionTitle')} description={t('appearance.sectionDescription')}>
      <SegmentedControl
        options={options}
        value={theme}
        onChange={(value) => {
          const option = options.find((candidate) => candidate.value === value);
          if (option) setTheme(option.value);
        }}
        label={t('appearance.sectionTitle')}
      />
      <p className="text-xs text-text-muted">{t('appearance.persistenceNote')}</p>
      <FormField label={t('appearance.language')} hint={t('appearance.languageDescription')}>
        {(control) => (
          <Select
            {...control}
            value={activeLanguage ?? 'en'}
            options={languages.map((language) => ({
              value: language.code,
              label: language.nativeLabel,
            }))}
            onValueChange={(value) => void i18n.changeLanguage(value)}
            data-testid="language-select"
          />
        )}
      </FormField>
    </Section>
  );
}

// =============================================================================
// Network Section
// =============================================================================

const virtualInterfacePatterns = /^(nflog|nfqueue|dbus-|bluetooth-monitor)/;

function deriveInterfaceType(name: string): string {
  if (/^(eth|enp|eno|ens)/.test(name)) return 'Ethernet';
  if (/^(wl|wlan)/.test(name)) return 'WiFi';
  if (name === 'lo' || name === 'lo0') return 'Loopback';
  if (/^(br|bridge)/.test(name)) return 'Bridge';
  if (/^(veth|docker|virbr)/.test(name)) return 'Virtual';
  if (/^(tun|tap)/.test(name)) return 'Tunnel';
  return 'Other';
}

function NetworkSection({ status }: { status: ConnectionState }): ReactElement {
  const { t } = useTranslation('settings');
  const [interfaces, setInterfaces] = useState<NetworkInterface[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchInterfaces()
      .then((resp) => {
        const filtered = resp.interfaces.filter(
          (iface) => !virtualInterfacePatterns.test(iface.name),
        );
        setInterfaces(filtered);
      })
      .catch(() => setInterfaces([]))
      .finally(() => setLoading(false));
  }, []);

  return (
    <>
      <Section
        title={t('network.interfacesTitle')}
        description={t('network.interfacesDescription')}
      >
        <div className="stack-sm">
          {loading && (
            <div role="status" aria-label={t('network.loadingInterfaces')} className="stack-sm">
              <Skeleton className="h-12 w-full" />
              <Skeleton className="h-12 w-full" />
            </div>
          )}
          {!loading && interfaces.length === 0 && (
            <p className="text-xs text-text-muted">{t('network.noInterfacesFound')}</p>
          )}
          {interfaces.map((iface) => (
            <InterfaceItem
              key={iface.name}
              name={iface.name}
              type={deriveInterfaceType(iface.name)}
              status={(iface.addresses?.length ?? 0) > 0 ? 'active' : 'inactive'}
              ip={iface.addresses?.[0] ?? '—'}
            />
          ))}
        </div>
        <p className="text-xs text-text-muted mt-inline">{t('network.interfaceManagedFrom')}</p>
      </Section>

      <Section
        title={t('network.connectionTitle')}
        description={t('network.connectionDescription')}
      >
        <SettingRow
          label={t('network.backendUrl')}
          description={t('network.backendUrlDescription')}
        >
          <code className="text-xs text-brand-primary-strong bg-brand-primary/10 px-cell py-compact rounded">
            {window.location.origin}
          </code>
        </SettingRow>
        <SettingRow
          label={t('network.connectionLabel')}
          description={t('network.websocketDescription')}
        >
          <ConnectionStatus status={status} />
        </SettingRow>
      </Section>
    </>
  );
}

interface InterfaceItemProps {
  name: string;
  type: string;
  status: 'active' | 'inactive';
  ip: string;
}

function InterfaceItem({ name, type, status, ip }: InterfaceItemProps): ReactElement {
  return (
    <div className="flex-between py-row px-3 bg-surface-hover rounded-lg">
      <div className={layout.inline.default}>
        <Network
          className={cn('w-4 h-4', status === 'active' ? 'text-status-success' : 'text-text-muted')}
        />
        <div>
          <div className="text-sm text-text-primary font-mono">{name}</div>
          <div className="text-xs text-text-muted">{type}</div>
        </div>
      </div>
      <div className="text-right">
        <div className="text-sm text-text-secondary font-mono">{ip}</div>
        <div
          className={cn('text-xs', status === 'active' ? 'text-status-success' : 'text-text-muted')}
        >
          {status}
        </div>
      </div>
    </div>
  );
}

// =============================================================================
// Debug Section
// =============================================================================

interface DebugSectionProps {
  onClose: () => void;
}

function DebugSection({ onClose }: DebugSectionProps): ReactElement {
  const { t } = useTranslation('settings');
  const navigate = useNavigate();

  return (
    <Section title={t('debug.devToolsTitle')} description={t('debug.devToolsDescription')}>
      <button
        type="button"
        onClick={() => {
          onClose();
          navigate('/debug');
        }}
        className={cn(
          layout.flex.between,
          'w-full py-row px-3 bg-surface-hover rounded-lg',
          'hover:bg-surface-hover transition-colors text-left',
        )}
      >
        <div>
          <div className="text-sm text-text-primary">{t('debug.debugConsole')}</div>
          <div className="text-xs text-text-muted">{t('debug.debugConsoleDescription')}</div>
        </div>
        <ChevronRight className="w-4 h-4 text-text-muted" />
      </button>
    </Section>
  );
}

// =============================================================================
// About Section
// =============================================================================

interface AboutSectionProps {
  version: string;
}

function AboutSection({ version }: AboutSectionProps): ReactElement {
  const { t } = useTranslation('settings');
  return (
    <>
      <Section title={t('about.applicationTitle')}>
        <div className="stack">
          <div>
            <h4 className="text-lg font-bold text-text-primary">NIAC</h4>
            <p className="text-sm text-text-muted">{t('about.subtitle')}</p>
          </div>
          <div className="grid grid-cols-2 gap-compact pt-2 border-t border-surface-border">
            <InfoItem label={t('about.version')} value={version} />
            <InfoItem label={t('about.build')} value={t('about.production')} />
            <InfoItem label="React" value={__REACT_VERSION__} />
            <InfoItem label="TypeScript" value={__TYPESCRIPT_VERSION__} />
          </div>
        </div>
      </Section>

      <Section title={t('about.legalTitle')}>
        <div className="stack-sm text-xs text-text-muted">
          <p>{t('about.copyright', { year: new Date().getFullYear() })}</p>
          <p>{t('about.proprietaryNotice')}</p>
        </div>
      </Section>

      <Section title={t('about.linksTitle')}>
        <div className="stack-sm">
          <a
            href="https://github.com/mustardseednetworks"
            target="_blank"
            rel="noopener noreferrer"
            className={cn(
              layout.flex.between,
              'w-full py-row px-3 bg-surface-hover rounded-lg',
              'hover:bg-surface-hover transition-colors',
            )}
          >
            <span className="text-sm text-text-primary">{t('about.github')}</span>
            <ChevronRight className="w-4 h-4 text-text-muted" />
          </a>
          <a
            href="https://mustardseednetworks.com"
            target="_blank"
            rel="noopener noreferrer"
            className={cn(
              layout.flex.between,
              'w-full py-row px-3 bg-surface-hover rounded-lg',
              'hover:bg-surface-hover transition-colors',
            )}
          >
            <span className="text-sm text-text-primary">{t('about.website')}</span>
            <ChevronRight className="w-4 h-4 text-text-muted" />
          </a>
        </div>
      </Section>
    </>
  );
}

interface InfoItemProps {
  label: string;
  value: string;
}

function InfoItem({ label, value }: InfoItemProps): ReactElement {
  return (
    <div>
      <div className="text-xs text-text-muted">{label}</div>
      <div className="text-sm text-text-primary font-mono">{value}</div>
    </div>
  );
}

export default SettingsDrawer;
