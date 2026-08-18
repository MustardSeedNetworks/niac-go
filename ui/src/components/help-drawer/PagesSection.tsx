/**
 * PagesSection Component
 *
 * The drawer's "Pages" tab: what each screen in the UI is for. This content
 * used to be JSX on the registry, reachable only from the page it described;
 * as a tab it is browsable and searchable next to the protocol and device
 * reference, and the page header's (?) opens the drawer straight to it.
 *
 * Page titles come from the `pages` namespace, so this list is translated
 * even though the help bodies are English.
 */

import type { ReactElement } from 'react';
import { useTranslation } from 'react-i18next';
import { pageHelp } from '../../data/page-help';
import { usePages } from '../../pageRegistry';
import { cn } from '../../styles/theme';
import { PageHelpBody } from './PageHelpBody';

interface PagesSectionProps {
  searchQuery: string;
  /** Route path whose help is showing. */
  activePath: string;
  onSelectPath: (path: string) => void;
}

/** Flattened block text, for matching a search query against page content. */
function bodyText(path: string): string {
  const blocks = pageHelp[path] ?? [];
  const parts: string[] = [];
  for (const block of blocks) {
    if (block.kind === 'paragraph' || block.kind === 'heading') {
      parts.push(block.text);
    } else if (block.kind === 'terms') {
      for (const item of block.items) {
        parts.push(item.term, item.description);
      }
    } else {
      parts.push(...block.items);
    }
  }
  return parts.join(' ').toLowerCase();
}

export function PagesSection({
  searchQuery,
  activePath,
  onSelectPath,
}: PagesSectionProps): ReactElement {
  const { t } = useTranslation('help');
  const documented = usePages().filter((page) => pageHelp[page.path]);

  const query = searchQuery.trim().toLowerCase();
  const matches = query
    ? documented.filter(
        (page) => page.title.toLowerCase().includes(query) || bodyText(page.path).includes(query),
      )
    : documented;

  const current = matches.find((page) => page.path === activePath) ?? matches[0];

  if (!current) {
    return <p className="text-sm text-text-muted py-row">{t('pages.empty')}</p>;
  }

  return (
    <div className="stack-lg">
      <nav className="flex flex-wrap gap-tight" aria-label={t('pages.contents')}>
        {matches.map((page) => (
          <button
            key={page.path}
            type="button"
            onClick={() => onSelectPath(page.path)}
            aria-current={page.path === current.path ? 'true' : undefined}
            className={cn(
              'px-3 py-1.5 rounded-lg text-xs font-medium transition-colors',
              page.path === current.path
                ? 'bg-brand-primary/10 text-brand-primary'
                : 'text-text-muted hover:bg-surface-hover hover:text-text-primary',
            )}
          >
            {page.title}
          </button>
        ))}
      </nav>

      <section>
        <h3 className="text-sm font-semibold text-text-primary mb-2">{current.title}</h3>
        <PageHelpBody blocks={pageHelp[current.path] ?? []} />
      </section>
    </div>
  );
}

export default PagesSection;
