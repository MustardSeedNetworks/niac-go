// Copyright (c) 2026 Mustard Seed Networks. All rights reserved.

/**
 * FAQSection
 *
 * Collapsible Q&A list, search-filterable.
 */

import { ChevronDown, ChevronRight } from 'lucide-react';
import type { ReactElement } from 'react';
import { useMemo, useState } from 'react';
import { faq } from '../../data/help-content';
import { cn } from '../../styles/theme';
import type { FAQEntry } from './types';

interface FAQSectionProps {
  searchQuery: string;
}

export function FAQSection({ searchQuery }: FAQSectionProps): ReactElement {
  const filtered = useMemo(() => {
    if (!searchQuery.trim()) return faq;
    const query = searchQuery.toLowerCase();
    return faq.filter(
      (q) =>
        q.question.toLowerCase().includes(query) ||
        q.answer.toLowerCase().includes(query) ||
        q.tags.some((t) => t.toLowerCase().includes(query)),
    );
  }, [searchQuery]);

  return (
    <div className="space-y-3">
      <h3 className="text-sm font-semibold text-text-primary">Frequently Asked Questions</h3>
      {filtered.length === 0 ? (
        <p className="text-sm text-text-muted py-4 text-center">No entries match your search.</p>
      ) : (
        <div className="space-y-2">
          {filtered.map((entry) => (
            <FAQItem key={entry.id} entry={entry} />
          ))}
        </div>
      )}
    </div>
  );
}

interface FAQItemProps {
  entry: FAQEntry;
}

function FAQItem({ entry }: FAQItemProps): ReactElement {
  const [open, setOpen] = useState(false);

  return (
    <div className="bg-white/5 rounded-lg overflow-hidden">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className={cn(
          'w-full text-left px-3 py-2.5 flex items-start gap-2',
          'hover:bg-white/5 transition-colors',
        )}
        aria-expanded={open}
      >
        {open ? (
          <ChevronDown className="w-4 h-4 mt-0.5 text-text-muted shrink-0" aria-hidden="true" />
        ) : (
          <ChevronRight className="w-4 h-4 mt-0.5 text-text-muted shrink-0" aria-hidden="true" />
        )}
        <h4 className="text-sm font-medium text-text-primary flex-1 min-w-0">{entry.question}</h4>
      </button>
      {open && (
        <div className="px-3 pb-3 pl-9 space-y-2 border-t border-white/5">
          <p className="text-xs text-text-muted leading-relaxed pt-2">{entry.answer}</p>
          {entry.tags.length > 0 && (
            <div className="flex flex-wrap gap-1">
              {entry.tags.map((tag) => (
                <code
                  key={tag}
                  className="text-xs text-text-muted bg-white/5 rounded px-1.5 py-0.5"
                >
                  {tag}
                </code>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
