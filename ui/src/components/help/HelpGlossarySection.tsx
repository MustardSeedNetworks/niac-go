import { Network } from 'lucide-react';
import type { ReactElement } from 'react';
import { useMemo } from 'react';
import type { GlossaryEntry } from './helpData';
import { CATEGORY_LABELS, GLOSSARY } from './helpData';

interface GlossarySectionProps {
  searchQuery: string;
}

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
