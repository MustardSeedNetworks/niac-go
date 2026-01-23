// Copyright (c) 2025 Mustard Seed Networks. All rights reserved.

/**
 * HelpDrawer Component
 *
 * Help panel with quick reference, glossary, and keyboard shortcuts.
 *
 * Features:
 * - Feature quick reference
 * - Network protocol glossary
 * - Keyboard shortcuts
 * - Search functionality
 *
 * Uses theme tokens and useFocusTrap for accessibility.
 */

import { Book, Command, HelpCircle, Keyboard, Network, Search, X, Zap } from 'lucide-react';
import type { ReactElement, ReactNode } from 'react';
import { useMemo, useState } from 'react';
import { useFocusTrap } from '../hooks/useFocusTrap';
import { badge, cn, drawer, layout, spacing } from '../styles/theme';

interface HelpDrawerProps {
  isOpen: boolean;
  onClose: () => void;
}

type HelpTab = 'features' | 'glossary' | 'shortcuts';

interface TabConfig {
  id: HelpTab;
  label: string;
  icon: ReactNode;
}

const TABS: TabConfig[] = [
  { id: 'features', label: 'Features', icon: <Zap className="w-4 h-4" /> },
  { id: 'glossary', label: 'Glossary', icon: <Book className="w-4 h-4" /> },
  { id: 'shortcuts', label: 'Shortcuts', icon: <Keyboard className="w-4 h-4" /> },
];

export function HelpDrawer({ isOpen, onClose }: HelpDrawerProps): ReactElement | null {
  const [activeTab, setActiveTab] = useState<HelpTab>('features');
  const [searchQuery, setSearchQuery] = useState('');

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
          aria-label="Close help drawer"
        />

        {/* Drawer */}
        <div
          ref={drawerRef}
          role="dialog"
          aria-modal="true"
          aria-label="Help"
          className={cn(drawer.content, drawer.size.lg, 'animate-slide-in-right')}
        >
          {/* Header */}
          <div className="sticky top-0 bg-gray-900 border-b border-white/10 z-10">
            <div className="px-4 py-3 flex items-center justify-between">
              <div className={layout.inline.default}>
                <HelpCircle className="w-5 h-5 text-violet-400" aria-hidden="true" />
                <h2 className="text-lg font-semibold text-white">Help</h2>
              </div>
              <button
                type="button"
                onClick={onClose}
                className={cn(
                  'p-2 hover:bg-white/10 rounded-lg transition-colors',
                  'text-gray-400 hover:text-white',
                )}
                aria-label="Close help"
              >
                <X className="w-5 h-5" aria-hidden="true" />
              </button>
            </div>

            {/* Search */}
            <div className="px-4 pb-3">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-500" />
                <input
                  type="text"
                  placeholder="Search help..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className={cn(
                    'w-full pl-10 pr-4 py-2 bg-white/5 border border-white/10 rounded-lg',
                    'text-sm text-white placeholder:text-gray-500',
                    'focus:outline-none focus:ring-2 focus:ring-violet-500/50',
                  )}
                />
              </div>
            </div>

            {/* Tab Navigation */}
            <div className="border-b border-white/10 px-2">
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
                        ? 'border-violet-500 text-white'
                        : 'border-transparent text-gray-400 hover:text-white hover:border-white/20',
                    )}
                  >
                    {tab.icon}
                    <span>{tab.label}</span>
                  </button>
                ))}
              </nav>
            </div>
          </div>

          {/* Content */}
          <div className={cn(spacing.drawer, 'space-y-6')}>
            {activeTab === 'features' && <FeaturesSection searchQuery={searchQuery} />}
            {activeTab === 'glossary' && <GlossarySection searchQuery={searchQuery} />}
            {activeTab === 'shortcuts' && <ShortcutsSection searchQuery={searchQuery} />}
          </div>
        </div>
      </div>
    </>
  );
}

// =============================================================================
// Features Section
// =============================================================================

interface Feature {
  title: string;
  description: string;
  path: string;
  badge?: string;
}

const FEATURES: Feature[] = [
  {
    title: 'Command Center',
    description: 'Live counters, run snapshots, and automation status for the active NIAC stack.',
    path: '/',
  },
  {
    title: 'Runtime Control',
    description: 'Monitor runtime status, view network interfaces, and manage NIAC configuration.',
    path: '/runtime',
  },
  {
    title: 'Devices & Config',
    description:
      'Review YAML-derived devices, SNMP walks, DHCP/DNS personas, and packet playback targets.',
    path: '/devices',
  },
  {
    title: 'Config Builder',
    description: 'Create, edit, and manage device configurations with a visual interface.',
    path: '/device-config',
    badge: 'New',
  },
  {
    title: 'Topology View',
    description: 'LLDP/CDP/EDP/FDP visibility for verifying intent before exporting to Graphviz.',
    path: '/topology',
  },
  {
    title: 'Analysis & Playback',
    description: 'Replay PCAPs, inspect SNMP walks, and publish bundles directly from the UI.',
    path: '/analysis',
  },
  {
    title: 'Traffic Injection',
    description: 'Configure and execute traffic injection scenarios with real-time feedback.',
    path: '/injection',
  },
  {
    title: 'Packet Inspector',
    description: 'Deep packet inspection with protocol decoding and hex dump viewing.',
    path: '/packets',
  },
  {
    title: 'PCAP Analyzer',
    description: 'Upload and analyze PCAP files with detailed statistics and filtering.',
    path: '/pcap',
    badge: 'Beta',
  },
];

interface FeaturesSectionProps {
  searchQuery: string;
}

function FeaturesSection({ searchQuery }: FeaturesSectionProps): ReactElement {
  const filteredFeatures = useMemo(() => {
    if (!searchQuery.trim()) return FEATURES;
    const query = searchQuery.toLowerCase();
    return FEATURES.filter(
      (f) => f.title.toLowerCase().includes(query) || f.description.toLowerCase().includes(query),
    );
  }, [searchQuery]);

  return (
    <div className="space-y-3">
      <h3 className="text-sm font-semibold text-white">Quick Reference</h3>
      {filteredFeatures.length === 0 ? (
        <p className="text-sm text-gray-500 py-4 text-center">No features match your search.</p>
      ) : (
        <div className="space-y-2">
          {filteredFeatures.map((feature) => (
            <FeatureCard key={feature.path} feature={feature} />
          ))}
        </div>
      )}
    </div>
  );
}

interface FeatureCardProps {
  feature: Feature;
}

function FeatureCard({ feature }: FeatureCardProps): ReactElement {
  return (
    <div className="bg-white/5 rounded-lg p-3 hover:bg-white/10 transition-colors">
      <div className={layout.flex.between}>
        <div className={layout.inline.default}>
          <h4 className="text-sm font-medium text-white">{feature.title}</h4>
          {feature.badge && (
            <span
              className={cn(
                badge.base,
                feature.badge === 'New' ? badge.variant.new : badge.variant.beta,
                badge.size.xs,
              )}
            >
              {feature.badge}
            </span>
          )}
        </div>
        <code className="text-xs text-gray-500">{feature.path}</code>
      </div>
      <p className="text-xs text-gray-400 mt-1">{feature.description}</p>
    </div>
  );
}

// =============================================================================
// Glossary Section
// =============================================================================

interface GlossaryEntry {
  term: string;
  definition: string;
  category: 'protocol' | 'concept' | 'device';
}

const GLOSSARY: GlossaryEntry[] = [
  {
    term: 'ARP',
    definition:
      'Address Resolution Protocol - Maps IP addresses to MAC addresses on a local network.',
    category: 'protocol',
  },
  {
    term: 'CDP',
    definition: 'Cisco Discovery Protocol - Proprietary protocol for discovering Cisco devices.',
    category: 'protocol',
  },
  {
    term: 'DHCP',
    definition:
      'Dynamic Host Configuration Protocol - Automatically assigns IP addresses to devices.',
    category: 'protocol',
  },
  {
    term: 'DNS',
    definition: 'Domain Name System - Translates domain names to IP addresses.',
    category: 'protocol',
  },
  {
    term: 'ICMP',
    definition: 'Internet Control Message Protocol - Used for network diagnostics (ping).',
    category: 'protocol',
  },
  {
    term: 'LLDP',
    definition: 'Link Layer Discovery Protocol - Vendor-neutral protocol for device discovery.',
    category: 'protocol',
  },
  {
    term: 'SNMP',
    definition: 'Simple Network Management Protocol - Monitors and manages network devices.',
    category: 'protocol',
  },
  {
    term: 'MAC Address',
    definition: 'Media Access Control address - Unique hardware identifier for network interfaces.',
    category: 'concept',
  },
  {
    term: 'PCAP',
    definition: 'Packet Capture - File format for storing captured network traffic.',
    category: 'concept',
  },
  {
    term: 'VLAN',
    definition: 'Virtual LAN - Partitions a physical network into multiple virtual networks.',
    category: 'concept',
  },
  {
    term: 'Router',
    definition: 'Network device that forwards packets between different networks.',
    category: 'device',
  },
  {
    term: 'Switch',
    definition: 'Network device that connects devices within the same network (Layer 2).',
    category: 'device',
  },
  {
    term: 'Firewall',
    definition: 'Security device that monitors and filters network traffic.',
    category: 'device',
  },
];

interface GlossarySectionProps {
  searchQuery: string;
}

function GlossarySection({ searchQuery }: GlossarySectionProps): ReactElement {
  const filteredGlossary = useMemo(() => {
    if (!searchQuery.trim()) return GLOSSARY;
    const query = searchQuery.toLowerCase();
    return GLOSSARY.filter(
      (entry) =>
        entry.term.toLowerCase().includes(query) || entry.definition.toLowerCase().includes(query),
    );
  }, [searchQuery]);

  const groupedEntries = useMemo(() => {
    const groups: Record<string, GlossaryEntry[]> = {
      protocol: [],
      concept: [],
      device: [],
    };
    for (const entry of filteredGlossary) {
      groups[entry.category].push(entry);
    }
    return groups;
  }, [filteredGlossary]);

  const categoryLabels: Record<string, string> = {
    protocol: 'Protocols',
    concept: 'Concepts',
    device: 'Device Types',
  };

  return (
    <div className="space-y-6">
      {filteredGlossary.length === 0 ? (
        <p className="text-sm text-gray-500 py-4 text-center">
          No glossary entries match your search.
        </p>
      ) : (
        Object.entries(groupedEntries).map(([category, entries]) =>
          entries.length > 0 ? (
            <div key={category} className="space-y-2">
              <h3 className="text-sm font-semibold text-white flex items-center gap-2">
                <Network className="w-4 h-4 text-violet-400" />
                {categoryLabels[category]}
              </h3>
              <div className="space-y-1">
                {entries.map((entry) => (
                  <GlossaryItem key={entry.term} entry={entry} />
                ))}
              </div>
            </div>
          ) : null,
        )
      )}
    </div>
  );
}

interface GlossaryItemProps {
  entry: GlossaryEntry;
}

function GlossaryItem({ entry }: GlossaryItemProps): ReactElement {
  return (
    <div className="bg-white/5 rounded-lg p-3">
      <dt className="text-sm font-medium text-white">{entry.term}</dt>
      <dd className="text-xs text-gray-400 mt-0.5">{entry.definition}</dd>
    </div>
  );
}

// =============================================================================
// Shortcuts Section
// =============================================================================

interface Shortcut {
  keys: string[];
  description: string;
  category: 'navigation' | 'actions' | 'general';
}

const SHORTCUTS: Shortcut[] = [
  { keys: ['Esc'], description: 'Close drawer or modal', category: 'general' },
  { keys: ['?'], description: 'Open help', category: 'general' },
  { keys: ['/', ','], description: 'Open settings', category: 'general' },
  { keys: ['g', 'h'], description: 'Go to Command Center', category: 'navigation' },
  { keys: ['g', 'r'], description: 'Go to Runtime Control', category: 'navigation' },
  { keys: ['g', 'd'], description: 'Go to Devices', category: 'navigation' },
  { keys: ['g', 't'], description: 'Go to Topology', category: 'navigation' },
  { keys: ['Ctrl', 'k'], description: 'Quick search', category: 'actions' },
  { keys: ['Ctrl', 's'], description: 'Save current configuration', category: 'actions' },
  { keys: ['Ctrl', 'Enter'], description: 'Run/execute action', category: 'actions' },
];

interface ShortcutsSectionProps {
  searchQuery: string;
}

function ShortcutsSection({ searchQuery }: ShortcutsSectionProps): ReactElement {
  const filteredShortcuts = useMemo(() => {
    if (!searchQuery.trim()) return SHORTCUTS;
    const query = searchQuery.toLowerCase();
    return SHORTCUTS.filter(
      (s) =>
        s.description.toLowerCase().includes(query) ||
        s.keys.some((k) => k.toLowerCase().includes(query)),
    );
  }, [searchQuery]);

  const groupedShortcuts = useMemo(() => {
    const groups: Record<string, Shortcut[]> = {
      general: [],
      navigation: [],
      actions: [],
    };
    for (const shortcut of filteredShortcuts) {
      groups[shortcut.category].push(shortcut);
    }
    return groups;
  }, [filteredShortcuts]);

  const categoryLabels: Record<string, string> = {
    general: 'General',
    navigation: 'Navigation',
    actions: 'Actions',
  };

  return (
    <div className="space-y-6">
      {filteredShortcuts.length === 0 ? (
        <p className="text-sm text-gray-500 py-4 text-center">No shortcuts match your search.</p>
      ) : (
        Object.entries(groupedShortcuts).map(([category, shortcuts]) =>
          shortcuts.length > 0 ? (
            <div key={category} className="space-y-2">
              <h3 className="text-sm font-semibold text-white flex items-center gap-2">
                <Command className="w-4 h-4 text-violet-400" />
                {categoryLabels[category]}
              </h3>
              <div className="space-y-1">
                {shortcuts.map((shortcut) => (
                  <ShortcutItem key={shortcut.description} shortcut={shortcut} />
                ))}
              </div>
            </div>
          ) : null,
        )
      )}
      <p className="text-xs text-gray-500">
        Keyboard shortcuts are contextual and may vary based on the current page.
      </p>
    </div>
  );
}

interface ShortcutItemProps {
  shortcut: Shortcut;
}

function ShortcutItem({ shortcut }: ShortcutItemProps): ReactElement {
  return (
    <div className={cn(layout.flex.between, 'py-2 px-3 bg-white/5 rounded-lg')}>
      <span className="text-sm text-gray-300">{shortcut.description}</span>
      <div className={layout.inline.tight}>
        {shortcut.keys.map((key, idx) => (
          <span key={`${shortcut.description}-${key}`}>
            <kbd
              className={cn(
                'px-2 py-0.5 text-xs font-mono rounded',
                'bg-gray-800 border border-white/20 text-gray-300',
              )}
            >
              {key}
            </kbd>
            {idx < shortcut.keys.length - 1 && <span className="text-gray-600 mx-0.5">+</span>}
          </span>
        ))}
      </div>
    </div>
  );
}

export default HelpDrawer;
