// Copyright (c) 2025 Mustard Seed Networks. All rights reserved.

/**
 * ShortcutsSection Component
 *
 * Displays keyboard shortcuts grouped by category.
 */

import { Command } from 'lucide-react';
import type { ReactElement } from 'react';
import { useMemo } from 'react';
import { cn, layout } from '../../styles/theme';
import { SHORTCUTS } from './data';
import type { Shortcut } from './types';

interface ShortcutsSectionProps {
  searchQuery: string;
}

const CATEGORY_LABELS: Record<string, string> = {
  general: 'General',
  navigation: 'Navigation',
  actions: 'Actions',
};

export function ShortcutsSection({ searchQuery }: ShortcutsSectionProps): ReactElement {
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

  return (
    <div className="space-y-6">
      {filteredShortcuts.length === 0 ? (
        <p className="text-sm text-text-muted py-4 text-center">No shortcuts match your search.</p>
      ) : (
        Object.entries(groupedShortcuts).map(([category, shortcuts]) =>
          shortcuts.length > 0 ? (
            <div key={category} className="space-y-2">
              <h3 className="text-sm font-semibold text-text-primary flex items-center gap-2">
                <Command className="w-4 h-4 text-brand-400" />
                {CATEGORY_LABELS[category]}
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
      <p className="text-xs text-text-muted">
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
      <span className="text-sm text-text-secondary">{shortcut.description}</span>
      <div className={layout.inline.tight}>
        {shortcut.keys.map((key, idx) => (
          <span key={`${shortcut.description}-${key}`}>
            <kbd
              className={cn(
                'px-2 py-0.5 text-xs font-mono rounded',
                'bg-bg-elevated border border-white/20 text-text-secondary',
              )}
            >
              {key}
            </kbd>
            {idx < shortcut.keys.length - 1 && <span className="text-text-disabled mx-0.5">+</span>}
          </span>
        ))}
      </div>
    </div>
  );
}
