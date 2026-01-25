// Copyright (c) 2025 Mustard Seed Networks. All rights reserved.

/**
 * GlossarySection Component
 *
 * Displays network protocol glossary grouped by category.
 */

import { Network } from 'lucide-react';
import type { ReactElement } from 'react';
import { useMemo } from 'react';
import { GLOSSARY } from './data';
import type { GlossaryEntry } from './types';

interface GlossarySectionProps {
  searchQuery: string;
}

const CATEGORY_LABELS: Record<string, string> = {
  protocol: 'Protocols',
  concept: 'Concepts',
  device: 'Device Types',
};

export function GlossarySection({ searchQuery }: GlossarySectionProps): ReactElement {
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
                {CATEGORY_LABELS[category]}
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
