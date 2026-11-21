import { useTranslation } from 'react-i18next';
import { Link } from 'react-router';

/**
 * Rendered for any route the registry does not know. Says what happened and
 * offers the way out; a silent redirect to the dashboard hid typos and stale
 * links from the operator.
 */
export function NotFound() {
  const { t } = useTranslation('common');
  return (
    <section className="flex flex-1 items-center justify-center p-8" data-testid="not-found">
      <div className="max-w-md rounded-xl border border-surface-border bg-surface-raised p-6 text-center">
        <h1 className="font-display text-xl font-semibold text-text-primary">
          {t('emptyState.notFoundTitle')}
        </h1>
        <p className="mt-2 text-sm text-text-secondary">{t('emptyState.notFoundBody')}</p>
        <Link
          to="/"
          className="mt-4 inline-block rounded-lg border border-surface-border px-3 py-2 text-sm text-text-primary hover:bg-surface-hover"
        >
          {t('emptyState.notFoundHome')}
        </Link>
      </div>
    </section>
  );
}
