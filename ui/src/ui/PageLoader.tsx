import type { FC } from 'react';
import { useTranslation } from 'react-i18next';

/**
 * PageLoader is the Suspense fallback for lazy-loaded route components.
 * Sized to roughly match a typical page header so the layout doesn't
 * jump when a chunk finishes loading.
 */
export const PageLoader: FC = () => {
  const { t } = useTranslation('common');

  return (
    <div className="flex-center min-h-[400px]">
      <div className="flex flex-col items-center gap-default">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-brand-primary border-t-transparent" />
        <p className="text-sm text-text-muted">{t('status.loading')}</p>
      </div>
    </div>
  );
};
