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
import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useFocusTrap } from '../hooks/useFocusTrap';
import { cn, drawer, layout, spacing } from '../styles/theme';
import { FAQSection } from './help-drawer/FAQSection';
import { GlossarySection } from './help-drawer/GlossarySection';
import { ItemListSection } from './help-drawer/ItemListSection';
import { OverviewSection } from './help-drawer/OverviewSection';
import { ShortcutsSection } from './help-drawer/ShortcutsSection';

interface HelpDrawerProps {
  isOpen: boolean;
  onClose: () => void;
}

type HelpTab = 'overview' | 'devices' | 'protocols' | 'commands' | 'glossary' | 'shortcuts' | 'faq';

interface TabConfig {
  id: HelpTab;
  /** English fallback label — also passed as the i18next defaultValue. */
  label: string;
  /** i18next key (in `help` namespace). */
  i18nKey: string;
  icon: ReactNode;
}

// "Protocols", "Commands", "Glossary", "Shortcuts", "FAQ" still describe
// generic UI affordances — they get translated. The values inside each
// list (industry protocol names, CLI command names) stay verbatim.
const TABS: TabConfig[] = [
  {
    id: 'overview',
    label: 'Overview',
    i18nKey: 'help:tabs.overview',
    icon: <LayoutGrid className="w-4 h-4" />,
  },
  {
    id: 'devices',
    label: 'Devices',
    i18nKey: 'help:tabs.devices',
    icon: <Boxes className="w-4 h-4" />,
  },
  {
    id: 'protocols',
    label: 'Protocols',
    i18nKey: 'help:tabs.protocols',
    icon: <Network className="w-4 h-4" />,
  },
  {
    id: 'commands',
    label: 'Commands',
    i18nKey: 'help:tabs.commands',
    icon: <Terminal className="w-4 h-4" />,
  },
  {
    id: 'glossary',
    label: 'Glossary',
    i18nKey: 'help:tabs.glossary',
    icon: <Book className="w-4 h-4" />,
  },
  {
    id: 'shortcuts',
    label: 'Shortcuts',
    i18nKey: 'help:tabs.shortcuts',
    icon: <Keyboard className="w-4 h-4" />,
  },
  // "FAQ" is widely recognised as an English abbreviation but is still
  // a user-facing label, so we translate it. Locale files can override.
  {
    id: 'faq',
    label: 'FAQ',
    i18nKey: 'help:tabs.faq',
    icon: <MessageCircleQuestion className="w-4 h-4" />,
  },
];

export function HelpDrawer({ isOpen, onClose }: HelpDrawerProps): ReactElement | null {
  const { t } = useTranslation();
  const [activeTab, setActiveTab] = useState<HelpTab>('overview');
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
          aria-label={t('help:drawer.backdropAriaLabel', 'Close help drawer')}
        />

        {/* Drawer */}
        <div
          ref={drawerRef}
          role="dialog"
          aria-modal="true"
          aria-label={t('help:drawer.drawerAriaLabel', 'Help')}
          className={cn(drawer.content, drawer.size.lg, 'animate-slide-in-right')}
        >
          {/* Header */}
          <div className="sticky top-0 bg-bg-surface border-b border-surface-border z-10">
            <div className="px-4 py-3 flex items-center justify-between">
              <div className={layout.inline.default}>
                <HelpCircle className="w-5 h-5 text-brand-accent" aria-hidden="true" />
                <h2 className="text-lg font-semibold text-text-primary">
                  {t('help:drawer.title', 'Help')}
                </h2>
              </div>
              <button
                type="button"
                onClick={onClose}
                className={cn(
                  'p-2 hover:bg-surface-hover rounded-lg transition-colors',
                  'text-text-muted hover:text-text-primary',
                )}
                aria-label={t('help:drawer.closeAriaLabel', 'Close help')}
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
                  placeholder={t('help:drawer.searchPlaceholder', 'Search help...')}
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className={cn(
                    'w-full pl-10 pr-4 py-2 bg-surface-hover border border-surface-border rounded-lg',
                    'text-sm text-text-primary placeholder:text-text-muted',
                    'focus:outline-none focus:ring-2 focus:ring-brand-primary/50',
                  )}
                />
              </div>
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
                    <span>{t(tab.i18nKey, tab.label)}</span>
                  </button>
                ))}
              </nav>
            </div>
          </div>

          {/* Content */}
          <div className={cn(spacing.drawer, 'space-y-6')}>
            {activeTab === 'overview' && <OverviewSection searchQuery={searchQuery} />}
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
