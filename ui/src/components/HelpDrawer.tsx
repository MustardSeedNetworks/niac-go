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
 * The frame is the shared RightDrawer and the sections are SettingsTabs.
 */

import {
  Book,
  Boxes,
  FileText,
  Keyboard,
  LayoutGrid,
  MessageCircleQuestion,
  Network,
  Terminal,
} from 'lucide-react';
import type { ReactElement, ReactNode } from 'react';
import { useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useBuildVersion } from '../hooks/useBuildVersion';
import { RightDrawer } from '../shared/RightDrawer';
import { SearchInput } from '../shared/SearchInput';
import { FAQSection } from './help-drawer/FAQSection';
import { GlossarySection } from './help-drawer/GlossarySection';
import { HelpTabs } from './help-drawer/HelpTabs';
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

export function HelpDrawer({ isOpen, onClose, section }: HelpDrawerProps): ReactElement {
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

  const tc = useTranslation('common').t;

  return (
    <RightDrawer
      open={isOpen}
      onClose={onClose}
      title={t('drawer.title')}
      closeLabel={t('drawer.closeAriaLabel')}
      data-testid="help-drawer"
      footer={
        <span className="mr-auto text-xs text-text-muted" data-testid="help-drawer-version">
          NIAC v{buildVersion.version.replace(/^v/, '')} · {buildVersion.commit}
        </span>
      }
    >
      <div className="stack">
        <SearchInput
          value={searchQuery}
          onValueChange={setSearchQuery}
          label={t('drawer.searchLabel')}
          clearLabel={tc('accessibility.clearSearch')}
          placeholder={t('drawer.searchPlaceholder')}
        />
        <HelpTabs
          tabs={TABS}
          value={activeTab}
          onChange={(id) => {
            const tab = TABS.find((candidate) => candidate.id === id);
            if (tab) setActiveTab(tab.id);
          }}
          label={t('drawer.tabsLabel')}
        >
          <div className="stack-xl">
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
        </HelpTabs>
      </div>
    </RightDrawer>
  );
}

export default HelpDrawer;
