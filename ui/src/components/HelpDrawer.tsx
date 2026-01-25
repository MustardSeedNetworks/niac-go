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

import { Book, HelpCircle, Keyboard, Search, X, Zap } from 'lucide-react';
import type { ReactElement } from 'react';
import { useState } from 'react';
import { useFocusTrap } from '../hooks/useFocusTrap';
import { cn, drawer, layout, spacing } from '../styles/theme';
import { FeaturesSection } from './help/HelpFeaturesSection';
import { GlossarySection } from './help/HelpGlossarySection';
import { ShortcutsSection } from './help/HelpShortcutsSection';
import type { HelpTab, TabConfig } from './help/helpData';

interface HelpDrawerProps {
  isOpen: boolean;
  onClose: () => void;
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

export default HelpDrawer;
