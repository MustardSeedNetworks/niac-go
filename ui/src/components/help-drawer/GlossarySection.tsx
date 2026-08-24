/**
 * GlossarySection Component
 *
 * Displays network protocol glossary grouped by category.
 */

import { Network } from 'lucide-react';
import type { ReactElement } from 'react';
import { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { GLOSSARY } from '../../data/help-content';
import type { GlossaryEntry } from './types';

interface GlossarySectionProps {
  searchQuery: string;
}

const CATEGORY_LABELS: Record<string, string> = {
  protocol: 'Protocols',
  concept: 'Concepts',
  device: 'Device Types',
  niac: 'NIAC-Specific',
  security: 'Security',
};

export function GlossarySection({ searchQuery }: GlossarySectionProps): ReactElement {
  const { t } = useTranslation('help');
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
      niac: [],
      security: [],
    };
    for (const entry of filteredGlossary) {
      const bucket = groups[entry.category];
      if (bucket) {
        bucket.push(entry);
      }
    }
    return groups;
  }, [filteredGlossary]);

  return (
    <div className="stack-xl">
      {filteredGlossary.length === 0 ? (
        <p className="text-sm text-text-muted py-4 text-center">{t('search.noGlossary')}</p>
      ) : (
        Object.entries(groupedEntries).map(([category, entries]) =>
          entries.length > 0 ? (
            <div key={category} className="stack-sm">
              <h3 className="text-sm font-semibold text-text-primary flex items-center gap-compact">
                <Network className="w-4 h-4 text-brand-accent" />
                {CATEGORY_LABELS[category]}
              </h3>
              <div className="stack-xs">
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
    <div className="bg-surface-hover rounded-lg pad-sm">
      <dt className="label">{entry.term}</dt>
      <dd className="text-xs text-text-muted mt-0.5">{entry.definition}</dd>
    </div>
  );
}
