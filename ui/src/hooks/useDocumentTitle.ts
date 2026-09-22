import { useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { useLocation } from 'react-router';
import { usePages } from '../pageRegistry';

/**
 * useDocumentTitle keeps the browser tab naming the route you are on.
 *
 * index.html sets the title once and nothing ever changed it, so every route
 * read "NIAC" — a history entry, a bookmark and a second window were all
 * indistinguishable (#2189). The name comes from the page registry, which is
 * where a route's name is already declared for the rail and the page header.
 *
 * A route the registry does not know — the per-device editor, whose segment
 * is a hostname the operator typed — keeps the product name rather than
 * having a tab name guessed from its URL.
 */
export function useDocumentTitle(): void {
  const { pathname } = useLocation();
  const { t } = useTranslation('common');
  const pages = usePages();

  useEffect(() => {
    const product = t('app.name');
    const page = pages.find((candidate) => candidate.path === pathname);
    // The dashboard is the product's front page, so "Dashboard | NIAC" would
    // only be a longer way of saying NIAC.
    document.title = page && page.path !== '/' ? `${page.label} | ${product}` : product;
  }, [pathname, pages, t]);
}
