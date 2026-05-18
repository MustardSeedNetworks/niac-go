/**
 * ShortcutsSection Component
 *
 * Displays keyboard shortcuts grouped by category.
 */

import { Command } from 'lucide-react';
import type { ReactElement } from 'react';
import { useMemo } from 'react';
import { SHORTCUTS } from '../../data/help-content';
import { cn, layout } from '../../styles/theme';
import type { Shortcut } from './types';

interface ShortcutsSectionProps {
  searchQuery: string;
}

const CATEGORY_LABELS: Record<string, string> = {
  general: 'General',
  navigation: 'Navigation',
  actions: 'Actions',
  tui: 'Terminal UI',
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
      tui: [],
    };
    for (const shortcut of filteredShortcuts) {
      const bucket = groups[shortcut.category];
      if (bucket) {
        bucket.push(shortcut);
      }
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
                <Command className="w-4 h-4 text-brand-accent" />
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
    <div className={cn(layout.flex.between, 'py-2 px-3 bg-surface-hover rounded-lg')}>
      <span className="text-sm text-text-secondary">{shortcut.description}</span>
      <div className={layout.inline.tight}>
        {shortcut.keys.map((key, idx) => (
          <span key={`${shortcut.description}-${key}`}>
            <kbd
              className={cn(
                'px-2 py-0.5 text-xs font-mono rounded',
                'bg-bg-elevated border border-surface-border text-text-secondary',
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
