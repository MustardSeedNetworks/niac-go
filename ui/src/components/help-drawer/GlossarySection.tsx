/**
 * GlossarySection Component
 *
 * Displays network protocol glossary grouped by category.
 */

import { Network } from 'lucide-react';
import type { ReactElement } from 'react';
import { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { type GlossaryEntry, getGlossary } from '../../data/glossary';

interface GlossarySectionProps {
  searchQuery: string;
}

export function GlossarySection({ searchQuery }: GlossarySectionProps): ReactElement {
  const { t } = useTranslation('help');
  const filteredGlossary = useMemo(() => {
    const glossary = getGlossary(t);
    if (!searchQuery.trim()) return glossary;
    const query = searchQuery.toLowerCase();
    return glossary.filter(
      (entry) =>
        entry.term.toLowerCase().includes(query) || entry.definition.toLowerCase().includes(query),
    );
  }, [searchQuery, t]);

  const groupedEntries = useMemo(() => {
    const groups: Record<GlossaryEntry['category'], GlossaryEntry[]> = {
      protocol: [],
      concept: [],
      device: [],
      niac: [],
      security: [],
    };
    for (const entry of filteredGlossary) {
      groups[entry.category].push(entry);
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
                {t(`glossaryCategories.${category as GlossaryEntry['category']}`)}
              </h3>
              <dl className="stack-xs">
                {entries.map((entry) => (
                  <GlossaryItem key={entry.term} entry={entry} />
                ))}
              </dl>
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
