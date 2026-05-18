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
import { fetchInterfaces } from '../api/client';
import type { NetworkInterface } from '../api/types';
import { iconSizes } from '../constants/sizes';
import { useFocusTrap } from '../hooks/useFocusTrap';
import { type Theme, useTheme } from '../hooks/useTheme';
import { badge, cn, drawer, layout, spacing } from '../styles/theme';
import { SimulationSection } from './settings/SimulationSection';

interface SettingsDrawerProps {
  isOpen: boolean;
  onClose: () => void;
  version?: string;
}

type SettingsTab = 'simulation' | 'appearance' | 'network' | 'debug' | 'about';

interface TabConfig {
  id: SettingsTab;
  label: string;
  icon: ReactNode;
}

const TABS: TabConfig[] = [
  { id: 'simulation', label: 'Simulation', icon: <PlugZap className="w-4 h-4" /> },
  { id: 'appearance', label: 'Appearance', icon: <Palette className="w-4 h-4" /> },
  { id: 'network', label: 'Network', icon: <Network className="w-4 h-4" /> },
  { id: 'debug', label: 'Debug', icon: <Bug className="w-4 h-4" /> },
  { id: 'about', label: 'About', icon: <Info className="w-4 h-4" /> },
];

export function SettingsDrawer({
  isOpen,
  onClose,
  version = '0.0.0',
}: SettingsDrawerProps): ReactElement | null {
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
          aria-label="Close settings drawer"
        />

        {/* Drawer */}
        <div
          ref={drawerRef}
          role="dialog"
          aria-modal="true"
          aria-label="Settings"
          className={cn(drawer.content, drawer.size.lg, 'animate-slide-in-right')}
        >
          {/* Header */}
          <div className="sticky top-0 bg-bg-surface border-b border-surface-border px-4 py-3 flex items-center justify-between z-10">
            <div className={layout.inline.default}>
              <Settings className="w-5 h-5 text-brand-accent" aria-hidden="true" />
              <h2 className="text-lg font-semibold text-text-primary">Settings</h2>
            </div>
            <button
              type="button"
              onClick={onClose}
              className={cn(
                'p-2 hover:bg-surface-hover rounded-lg transition-colors',
                'text-text-muted hover:text-text-primary',
              )}
              aria-label="Close settings"
            >
              <X className="w-5 h-5" aria-hidden="true" />
            </button>
          </div>

          {/* Tab Navigation */}
          <div className="border-b border-surface-border px-2">
            <nav className="flex gap-1 -mb-px">
              {TABS.map((tab) => (
                <button
                  key={tab.id}
                  type="button"
                  role="tab"
                  aria-selected={activeTab === tab.id}
                  onClick={() => setActiveTab(tab.id)}
                  className={cn(
                    'flex items-center gap-2 px-3 py-2.5 text-sm font-medium transition-colors',
                    'border-b-2 -mb-[2px]',
                    activeTab === tab.id
                      ? 'border-brand-primary text-text-primary'
                      : 'border-transparent text-text-muted hover:text-text-primary hover:border-surface-border',
                  )}
                >
                  {tab.icon}
                  <span>{tab.label}</span>
                </button>
              ))}
            </nav>
          </div>

          {/* Content */}
          <div className={cn(spacing.drawer, 'space-y-6')}>
            {activeTab === 'simulation' && <SimulationSection />}
            {activeTab === 'appearance' && <AppearanceSection />}
            {activeTab === 'network' && <NetworkSection />}
            {activeTab === 'debug' && <DebugSection />}
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
  <div className="space-y-3">
    <div>
      <h3 className="text-sm font-semibold text-text-primary">{title}</h3>
      {description && <p className="text-xs text-text-muted mt-0.5">{description}</p>}
    </div>
    <div className="space-y-2">{children}</div>
  </div>
);

interface SettingRowProps {
  label: string;
  description?: string;
  children: ReactNode;
}

const SettingRow = ({ label, description, children }: SettingRowProps): ReactElement => (
  <div className="flex items-center justify-between gap-4 py-2 px-3 bg-surface-hover rounded-lg">
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
  const { theme, setTheme, isDark, toggleTheme } = useTheme();

  const options: Array<{ id: Theme; label: string; icon: ReactNode }> = [
    { id: 'dark', label: 'Dark', icon: <Moon className="w-4 h-4" aria-hidden="true" /> },
    { id: 'light', label: 'Light', icon: <Sun className="w-4 h-4" aria-hidden="true" /> },
    {
      id: 'system',
      label: 'System',
      icon: <Monitor className="w-4 h-4" aria-hidden="true" />,
    },
  ];

  return (
    <Section title="Theme" description="Customize the appearance of NIAC">
      <div className="grid grid-cols-3 gap-2">
        {options.map((option) => {
          const selected = theme === option.id;
          return (
            <button
              key={option.id}
              type="button"
              aria-pressed={selected}
              onClick={() => setTheme(option.id)}
              className={cn(
                'flex flex-col items-center gap-2 p-3 rounded-lg border transition-all',
                selected
                  ? 'border-brand-primary bg-brand-primary/10'
                  : 'border-surface-border hover:border-brand-primary/40 hover:bg-surface-hover',
              )}
            >
              <div
                className={cn(
                  'w-8 h-8 rounded-lg flex items-center justify-center',
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
                {option.label}
              </span>
            </button>
          );
        })}
      </div>
      <button
        type="button"
        onClick={toggleTheme}
        title={isDark ? 'Switch to light mode' : 'Switch to dark mode'}
        className={cn(
          'mt-3 w-full flex items-center justify-between gap-2 p-2.5 rounded-lg border transition-colors',
          'border-surface-border bg-surface-hover hover:bg-surface-base text-text-primary',
        )}
      >
        <span className="text-sm">Quick toggle</span>
        <span className="flex items-center gap-1.5 text-xs text-text-muted">
          {isDark ? (
            <>
              <Sun className="w-4 h-4" aria-hidden="true" />
              Light
            </>
          ) : (
            <>
              <Moon className="w-4 h-4" aria-hidden="true" />
              Dark
            </>
          )}
        </span>
      </button>
      <p className="text-xs text-text-muted mt-2">
        Dark mode is the default. Your preference is saved across sessions.
      </p>
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
        title="Network Interfaces"
        description="Available network interfaces for traffic injection"
      >
        <div className="space-y-2">
          {loading && <p className="text-xs text-text-muted">Loading interfaces...</p>}
          {!loading && interfaces.length === 0 && (
            <p className="text-xs text-text-muted">No interfaces found</p>
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
        <p className="text-xs text-text-muted mt-2">
          Interface selection is managed from the Runtime Control page.
        </p>
      </Section>

      <Section title="Connection Settings" description="Configure backend connection">
        <SettingRow label="Backend URL" description="NIAC backend server address">
          <code className="text-xs text-brand-accent bg-brand-primary/10 px-2 py-1 rounded">
            localhost:8080
          </code>
        </SettingRow>
        <SettingRow label="WebSocket" description="Real-time event streaming">
          <span className={cn(badge.base, badge.variant.success, badge.size.sm)}>Connected</span>
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
    <div className="flex items-center justify-between py-2 px-3 bg-surface-hover rounded-lg">
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

function DebugSection(): ReactElement {
  const [logLevel, setLogLevel] = useState<'error' | 'warn' | 'info' | 'debug'>('info');

  return (
    <>
      <Section title="Logging" description="Configure debug output verbosity">
        <SettingRow label="Log Level" description="Console output verbosity">
          <select
            value={logLevel}
            onChange={(e) => setLogLevel(e.target.value as typeof logLevel)}
            className={cn(
              'bg-bg-elevated border border-surface-border rounded-lg px-3 py-1.5 text-sm text-text-primary',
              'focus:outline-none focus:ring-2 focus:ring-brand-primary/50',
            )}
          >
            <option value="error">Error</option>
            <option value="warn">Warning</option>
            <option value="info">Info</option>
            <option value="debug">Debug</option>
          </select>
        </SettingRow>
      </Section>

      <Section title="Developer Tools" description="Advanced debugging features">
        <button
          type="button"
          className={cn(
            layout.flex.between,
            'w-full py-2 px-3 bg-surface-hover rounded-lg',
            'hover:bg-surface-hover transition-colors text-left',
          )}
        >
          <div>
            <div className="text-sm text-text-primary">Protocol Debug Levels</div>
            <div className="text-xs text-text-muted">Configure per-protocol logging</div>
          </div>
          <ChevronRight className="w-4 h-4 text-text-muted" />
        </button>

        <button
          type="button"
          className={cn(
            layout.flex.between,
            'w-full py-2 px-3 bg-surface-hover rounded-lg',
            'hover:bg-surface-hover transition-colors text-left',
          )}
        >
          <div>
            <div className="text-sm text-text-primary">Export Diagnostics</div>
            <div className="text-xs text-text-muted">Download debug information</div>
          </div>
          <ChevronRight className="w-4 h-4 text-text-muted" />
        </button>
      </Section>
    </>
  );
}

// =============================================================================
// About Section
// =============================================================================

interface AboutSectionProps {
  version: string;
}

function AboutSection({ version }: AboutSectionProps): ReactElement {
  return (
    <>
      <Section title="Application">
        <div className="bg-surface-hover rounded-lg p-4 space-y-3">
          <div className="flex items-center gap-3">
            <div className="h-12 w-12 rounded-xl bg-gradient-to-br from-brand-primary to-brand-primary flex items-center justify-center shadow-lg shadow-brand-primary/30">
              <Network className={`${iconSizes.xl} text-text-primary`} />
            </div>
            <div>
              <h4 className="text-lg font-bold text-text-primary">NIAC</h4>
              <p className="text-sm text-text-muted">Network Injection & Analysis Console</p>
            </div>
          </div>
          <div className="grid grid-cols-2 gap-2 pt-2 border-t border-surface-border">
            <InfoItem label="Version" value={version} />
            <InfoItem label="Build" value="Production" />
            <InfoItem label="React" value="19.2" />
            <InfoItem label="TypeScript" value="5.9" />
          </div>
        </div>
      </Section>

      <Section title="Legal">
        <div className="space-y-2 text-xs text-text-muted">
          <p>Copyright (c) 2025 Mustard Seed Networks. All rights reserved.</p>
          <p>
            NIAC is proprietary software. Unauthorized copying, modification, or distribution is
            prohibited.
          </p>
        </div>
      </Section>

      <Section title="Links">
        <div className="space-y-2">
          <a
            href="https://github.com/mustardseednetworks"
            target="_blank"
            rel="noopener noreferrer"
            className={cn(
              layout.flex.between,
              'w-full py-2 px-3 bg-surface-hover rounded-lg',
              'hover:bg-surface-hover transition-colors',
            )}
          >
            <span className="text-sm text-text-primary">GitHub</span>
            <ChevronRight className="w-4 h-4 text-text-muted" />
          </a>
          <a
            href="https://mustardseednetworks.com"
            target="_blank"
            rel="noopener noreferrer"
            className={cn(
              layout.flex.between,
              'w-full py-2 px-3 bg-surface-hover rounded-lg',
              'hover:bg-surface-hover transition-colors',
            )}
          >
            <span className="text-sm text-text-primary">Website</span>
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
