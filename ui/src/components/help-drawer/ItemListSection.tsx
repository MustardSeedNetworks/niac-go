/**
 * ItemListSection
 *
 * Generic list of HelpItems filtered by category and search query.
 * Backing component for Devices / Protocols / Commands tabs.
 */

import type { ReactElement } from 'react';
import { useMemo } from 'react';
import { categories, items } from '../../data/help-content';
import { HelpItemCard } from './HelpItemCard';

interface ItemListSectionProps {
  /** Category id from `categories` map. */
  categoryId: keyof typeof categories | string;
  searchQuery: string;
}

export function ItemListSection({ categoryId, searchQuery }: ItemListSectionProps): ReactElement {
  const category = categories[categoryId];

  const filtered = useMemo(() => {
    const inCategory = items.filter((it) => it.category === categoryId);
    if (!searchQuery.trim()) return inCategory;
    const query = searchQuery.toLowerCase();
    return inCategory.filter(
      (it) =>
        it.name.toLowerCase().includes(query) ||
        it.summary.toLowerCase().includes(query) ||
        it.techDesc.toLowerCase().includes(query) ||
        it.laymanDesc.toLowerCase().includes(query) ||
        it.id.toLowerCase().includes(query),
    );
  }, [categoryId, searchQuery]);

  return (
    <div className="stack">
      {category && (
        <section>
          <h3 className="text-sm font-semibold text-text-primary">{category.fullName}</h3>
          <p className="text-xs text-text-muted mt-tight leading-relaxed">{category.description}</p>
        </section>
      )}

      {filtered.length === 0 ? (
        <p className="text-sm text-text-muted py-4 text-center">No entries match your search.</p>
      ) : (
        <div className="stack-sm">
          {filtered.map((item) => (
            <HelpItemCard key={item.id} item={item} />
          ))}
        </div>
      )}
    </div>
  );
}
