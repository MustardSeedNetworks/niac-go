import { SearchX } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { ButtonLink } from '../shared/Button';
import { Card } from '../shared/Card';
import { EmptyState } from '../shared/EmptyState';
import { linkTo } from '../ui/linkTo';
import { AppPageHeader } from './AppPageHeader';

/**
 * Rendered for any route the registry does not know. Says what happened and
 * offers the way out; a silent redirect to the dashboard hid typos and stale
 * links from the operator.
 */
export function NotFound() {
  const { t } = useTranslation('common');
  return (
    <section className="stack-xl" data-testid="not-found">
      <AppPageHeader
        kicker={t('app.name')}
        title={t('emptyState.notFoundTitle')}
        currentLabel={t('emptyState.notFoundTitle')}
      />
      <Card>
        <EmptyState
          icon={<SearchX className="h-6 w-6" />}
          title={t('emptyState.notFoundBody')}
          action={<ButtonLink renderLink={linkTo('/')}>{t('emptyState.notFoundHome')}</ButtonLink>}
        />
      </Card>
    </section>
  );
}
