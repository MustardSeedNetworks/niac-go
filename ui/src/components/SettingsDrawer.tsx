// Copyright (c) 2025 Mustard Seed Networks. All rights reserved.

/**
 * SettingsDrawer Component
 *
 * Main configuration panel for NIAC.
 *
 * Features:
 * - Appearance settings (theme toggle placeholder for future)
 * - Network interface display
 * - Debug logging controls
 * - About/version information
 *
 * Uses theme tokens and useFocusTrap for accessibility.
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
  Settings,
  Sun,
  X,
} from 'lucide-react';
import type { ReactElement, ReactNode } from 'react';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import { fetchInterfaces } from '../api/client';
import type { NetworkInterface } from '../api/types';
import { iconSizes } from '../constants/sizes';
import { useFocusTrap } from '../hooks/useFocusTrap';
import { type Theme, useTheme } from '../hooks/useTheme';
import { cn, drawer, layout, spacing } from '../styles/theme';
import { ConnectionStatus } from '../ui/ConnectionStatus';
import { SimulationSection } from './settings/SimulationSection';

interface SettingsDrawerProps {
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
  isOpen,
  onClose,
  version = '0.0.0',
}: SettingsDrawerProps): ReactElement | null {
  const { t } = useTranslation('settings');
  const [activeTab, setActiveTab] = useState<SettingsTab>('simulation');

  const drawerRef = useFocusTrap<HTMLDivElement>({
    isActive: isOpen,
    onEscape: onClose,
  });

  if (!isOpen) {
    return null;
  }

  return (
    <>
      {/* Backdrop */}
      <div className={drawer.overlay}>
        <button
          type="button"
          className={cn(drawer.backdrop, 'cursor-default')}
          onClick={onClose}
          aria-label={t('drawer.backdropAriaLabel')}
        />

        {/* Drawer */}
        <div
          ref={drawerRef}
          role="dialog"
          aria-modal="true"
          aria-labelledby="settings-drawer-title"
          data-testid="settings-drawer"
          className={cn(drawer.content, drawer.size.lg, 'animate-slide-in-right')}
        >
          {/* Header */}
          <div className="sticky top-0 bg-bg-surface border-b border-surface-border px-4 py-row-lg flex-between z-10">
            <div className={layout.inline.default}>
              <Settings className="w-5 h-5 text-brand-accent" aria-hidden="true" />
              <h2 id="settings-drawer-title" className="heading-3 text-text-primary">
                {t('drawer.title')}
              </h2>
            </div>
            <button
              type="button"
              onClick={onClose}
              data-testid="settings-drawer-close"
              className={cn(
                'pad-xs hover:bg-surface-hover rounded-lg transition-colors',
                'text-text-muted hover:text-text-primary',
              )}
              aria-label={t('drawer.closeAriaLabel')}
            >
              <X className="w-5 h-5" aria-hidden="true" />
            </button>
          </div>

          {/* Tab Navigation */}
          <div className="border-b border-surface-border px-cell">
            <nav className="flex gap-tight -mb-px">
              {TABS.map((tab) => (
                <button
                  key={tab.id}
                  type="button"
                  role="tab"
                  aria-selected={activeTab === tab.id}
                  onClick={() => setActiveTab(tab.id)}
                  className={cn(
                    'flex items-center gap-compact px-3 py-2.5 text-sm font-medium transition-colors',
                    'border-b-2 -mb-[2px]',
                    activeTab === tab.id
                      ? 'border-brand-primary text-text-primary'
                      : 'border-transparent text-text-muted hover:text-text-primary hover:border-surface-border',
                  )}
                >
                  {tab.icon}
                  <span>{t(`tabs.${tab.id}`)}</span>
                </button>
              ))}
            </nav>
          </div>

          {/* Content */}
          <div className={cn(spacing.drawer, 'stack-xl')}>
            {activeTab === 'simulation' && <SimulationSection />}
            {activeTab === 'appearance' && <AppearanceSection />}
            {activeTab === 'network' && <NetworkSection />}
            {activeTab === 'debug' && <DebugSection onClose={onClose} />}
            {activeTab === 'about' && <AboutSection version={version} />}
          </div>
        </div>
      </div>
    </>
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
  const { t } = useTranslation('settings');
  const { theme, setTheme, isDark, toggleTheme } = useTheme();

  const options: Array<{
    id: Theme;
    labelKey: 'appearance.dark' | 'appearance.light' | 'appearance.system';
    icon: ReactNode;
  }> = [
    {
      id: 'dark',
      labelKey: 'appearance.dark',
      icon: <Moon className="w-4 h-4" aria-hidden="true" />,
    },
    {
      id: 'light',
      labelKey: 'appearance.light',
      icon: <Sun className="w-4 h-4" aria-hidden="true" />,
    },
    {
      id: 'system',
      labelKey: 'appearance.system',
      icon: <Monitor className="w-4 h-4" aria-hidden="true" />,
    },
  ];

  return (
    <Section title={t('appearance.sectionTitle')} description={t('appearance.sectionDescription')}>
      <div className="grid grid-cols-3 gap-compact">
        {options.map((option) => {
          const selected = theme === option.id;
          return (
            <button
              key={option.id}
              type="button"
              aria-pressed={selected}
              onClick={() => setTheme(option.id)}
              className={cn(
                'flex flex-col items-center gap-compact pad-sm rounded-lg border transition-all',
                selected
                  ? 'border-brand-primary bg-brand-primary/10'
                  : 'border-surface-border hover:border-brand-primary/40 hover:bg-surface-hover',
              )}
            >
              <div
                className={cn(
                  'w-8 h-8 rounded-lg flex-center',
                  selected
                    ? 'bg-brand-primary/15 text-brand-primary'
                    : 'bg-surface-hover text-text-secondary',
                )}
              >
                {option.icon}
              </div>
              <span
                className={cn(
                  'text-xs font-medium',
                  selected ? 'text-text-primary' : 'text-text-muted',
                )}
              >
                {t(option.labelKey)}
              </span>
            </button>
          );
        })}
      </div>
      <button
        type="button"
        onClick={toggleTheme}
        title={isDark ? t('appearance.switchToLight') : t('appearance.switchToDark')}
        className={cn(
          'mt-heading w-full flex-between gap-compact p-2.5 rounded-lg border transition-colors',
          'border-surface-border bg-surface-hover hover:bg-surface-base text-text-primary',
        )}
      >
        <span className="text-sm">{t('appearance.quickToggle')}</span>
        <span className="flex items-center gap-1.5 text-xs text-text-muted">
          {isDark ? (
            <>
              <Sun className="w-4 h-4" aria-hidden="true" />
              {t('appearance.light')}
            </>
          ) : (
            <>
              <Moon className="w-4 h-4" aria-hidden="true" />
              {t('appearance.dark')}
            </>
          )}
        </span>
      </button>
      <p className="text-xs text-text-muted mt-inline">{t('appearance.persistenceNote')}</p>
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

function NetworkSection(): ReactElement {
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
          {loading && <p className="text-xs text-text-muted">{t('network.loadingInterfaces')}</p>}
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
          <code className="text-xs text-brand-accent bg-brand-primary/10 px-cell py-compact rounded">
            {window.location.origin}
          </code>
        </SettingRow>
        <SettingRow label="Connection" description={t('network.websocketDescription')}>
          <ConnectionStatus />
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
        <div className="bg-surface-hover rounded-lg pad stack">
          <div className="flex items-center gap-default">
            <div className="h-12 w-12 rounded-xl bg-gradient-to-br from-brand-primary to-brand-primary flex-center shadow-lg shadow-brand-primary/30">
              <Network className={`${iconSizes.xl} text-text-primary`} />
            </div>
            <div>
              <h4 className="text-lg font-bold text-text-primary">NIAC</h4>
              <p className="text-sm text-text-muted">{t('about.subtitle')}</p>
            </div>
          </div>
          <div className="grid grid-cols-2 gap-compact pt-2 border-t border-surface-border">
            <InfoItem label={t('about.version')} value={version} />
            <InfoItem label={t('about.build')} value="Production" />
            <InfoItem label="React" value="19.2" />
            <InfoItem label="TypeScript" value="5.9" />
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
