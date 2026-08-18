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

import {
  Book,
  Boxes,
  FileText,
  HelpCircle,
  Keyboard,
  LayoutGrid,
  MessageCircleQuestion,
  Network,
  Search,
  Terminal,
  X,
} from 'lucide-react';
import type { ReactElement, ReactNode } from 'react';
import { useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useBuildVersion } from '../hooks/useBuildVersion';
import { useFocusTrap } from '../hooks/useFocusTrap';
import { cn, drawer, layout, spacing } from '../styles/theme';
import { FAQSection } from './help-drawer/FAQSection';
import { GlossarySection } from './help-drawer/GlossarySection';
import { ItemListSection } from './help-drawer/ItemListSection';
import { OverviewSection } from './help-drawer/OverviewSection';
import { PagesSection } from './help-drawer/PagesSection';
import { ShortcutsSection } from './help-drawer/ShortcutsSection';

interface HelpDrawerProps {
  isOpen: boolean;
  onClose: () => void;
  /**
   * Route path this open was requested on — supplied by a page header's (?).
   * App clears it on close, so clicking (?) always lands on that page's help
   * even if the reader browsed elsewhere during the previous open.
   */
  section?: string;
}

type HelpTab =
  | 'overview'
  | 'pages'
  | 'devices'
  | 'protocols'
  | 'commands'
  | 'glossary'
  | 'shortcuts'
  | 'faq';

interface TabConfig {
  id: HelpTab;
  label: string;
  icon: ReactNode;
}

export function HelpDrawer({ isOpen, onClose, section }: HelpDrawerProps): ReactElement | null {
  const { t } = useTranslation('help');
  const [activeTab, setActiveTab] = useState<HelpTab>(section ? 'pages' : 'overview');
  const [activePath, setActivePath] = useState(section ?? '/');
  const [requestedSection, setRequestedSection] = useState(section);
  const [searchQuery, setSearchQuery] = useState('');
  const buildVersion = useBuildVersion();

  if (section !== requestedSection) {
    setRequestedSection(section);
    if (section) {
      setActiveTab('pages');
      setActivePath(section);
    }
  }

  const TABS: TabConfig[] = useMemo(
    () => [
      { id: 'overview', label: t('tabs.overview'), icon: <LayoutGrid className="w-4 h-4" /> },
      { id: 'pages', label: t('tabs.pages'), icon: <FileText className="w-4 h-4" /> },
      { id: 'devices', label: t('tabs.devices'), icon: <Boxes className="w-4 h-4" /> },
      { id: 'protocols', label: t('tabs.protocols'), icon: <Network className="w-4 h-4" /> },
      { id: 'commands', label: t('tabs.commands'), icon: <Terminal className="w-4 h-4" /> },
      { id: 'glossary', label: t('tabs.glossary'), icon: <Book className="w-4 h-4" /> },
      { id: 'shortcuts', label: t('tabs.shortcuts'), icon: <Keyboard className="w-4 h-4" /> },
      {
        id: 'faq',
        label: t('tabs.faq'),
        icon: <MessageCircleQuestion className="w-4 h-4" />,
      },
    ],
    [t],
  );

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
          data-testid="help-drawer"
          role="dialog"
          aria-modal="true"
          aria-label={t('drawer.drawerAriaLabel')}
          className={cn(drawer.content, drawer.size.lg, 'animate-slide-in-right')}
        >
          {/* Header */}
          <div className="sticky top-0 bg-bg-surface border-b border-surface-border z-10">
            <div className="px-4 py-row-lg flex-between">
              <div className={layout.inline.default}>
                <HelpCircle className="w-5 h-5 text-brand-accent" aria-hidden="true" />
                <div>
                  <h2 className="heading-3 text-text-primary">{t('drawer.title')}</h2>
                  <p
                    className="caption text-text-muted"
                    data-testid="help-drawer-version"
                    title={`commit ${buildVersion.commit} · built ${buildVersion.buildTime}`}
                  >
                    NIAC v{buildVersion.version}
                  </p>
                </div>
              </div>
              <button
                type="button"
                data-testid="help-drawer-close"
                onClick={onClose}
                className={cn(
                  'pad-xs hover:bg-surface-hover rounded-lg transition-colors',
                  'text-text-muted hover:text-text-primary',
                )}
                aria-label={t('drawer.closeAriaLabel')}
              >
                <X className="w-5 h-5" aria-hidden="true" />
              </button>
            </div>

            {/* Search */}
            <div className="px-4 pb-3">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-text-muted" />
                <input
                  type="text"
                  placeholder={t('drawer.searchPlaceholder')}
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className={cn(
                    'w-full pl-10 pr-4 py-row bg-surface-hover border border-surface-border rounded-lg',
                    'text-sm text-text-primary placeholder:text-text-muted',
                    'focus:outline-none focus:ring-2 focus:ring-brand-primary/50',
                  )}
                />
              </div>
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
                    <span>{tab.label}</span>
                  </button>
                ))}
              </nav>
            </div>
          </div>

          {/* Content */}
          <div className={cn(spacing.drawer, 'stack-xl')}>
            {activeTab === 'overview' && <OverviewSection searchQuery={searchQuery} />}
            {activeTab === 'pages' && (
              <PagesSection
                searchQuery={searchQuery}
                activePath={activePath}
                onSelectPath={setActivePath}
              />
            )}
            {activeTab === 'devices' && (
              <ItemListSection categoryId="devices" searchQuery={searchQuery} />
            )}
            {activeTab === 'protocols' && (
              <ItemListSection categoryId="protocols" searchQuery={searchQuery} />
            )}
            {activeTab === 'commands' && (
              <ItemListSection categoryId="commands" searchQuery={searchQuery} />
            )}
            {activeTab === 'glossary' && <GlossarySection searchQuery={searchQuery} />}
            {activeTab === 'shortcuts' && <ShortcutsSection searchQuery={searchQuery} />}
            {activeTab === 'faq' && <FAQSection searchQuery={searchQuery} />}
          </div>
        </div>
      </div>
    </>
  );
}

export default HelpDrawer;
