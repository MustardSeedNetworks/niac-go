import { AlertCircle, FileCode, Search, Upload } from 'lucide-react';
import type { FC } from 'react';
import { useTranslation } from 'react-i18next';
import { iconSizes } from '../../constants/sizes';
import { Button } from '../../ui/Button';
import { Card, CardContent } from '../../ui/Card';
import { H2, P, SmallText } from '../../ui/Typography';

interface LoadingStateProps {
  show: boolean;
}

export const TemplatesLoadingState: FC<LoadingStateProps> = ({ show }) => {
  const { t } = useTranslation('pages');
  if (!show) {
    return null;
  }

  return (
    <Card className="border-surface-border bg-bg-surface/70">
      <CardContent className="flex items-center justify-center py-12">
        <div className="flex items-center gap-3 text-text-muted">
          <div
            className={`${iconSizes.lg} animate-spin rounded-full border-2 border-brand-primary border-t-transparent`}
          />
          <span>{t('templates.states.loading')}</span>
        </div>
      </CardContent>
    </Card>
  );
};

interface ErrorStateProps {
  error: Error | null;
  onRetry: () => void;
}

export const TemplatesErrorState: FC<ErrorStateProps> = ({ error, onRetry }) => {
  const { t } = useTranslation('pages');
  if (!error) {
    return null;
  }

  return (
    <Card className="border-status-error/30 bg-status-error/20">
      <CardContent className="space-y-3">
        <div className="flex items-start gap-3">
          <AlertCircle className={`mt-1 ${iconSizes.lg} text-status-error`} />
          <div>
            <p className="font-semibold text-status-error">
              {t('templates.states.loadErrorTitle')}
            </p>
            <SmallText className="text-status-error/90">{error.message}</SmallText>
            <Button variant="outline" size="sm" className="mt-3" onClick={onRetry}>
              {t('templates.states.retry')}
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  );
};

interface EmptyStateProps {
  show: boolean;
  onUploadClick: () => void;
}

export const TemplatesEmptyState: FC<EmptyStateProps> = ({ show, onUploadClick }) => {
  const { t } = useTranslation('pages');
  if (!show) {
    return null;
  }

  return (
    <Card className="border-surface-border bg-bg-surface/70">
      <CardContent className="py-12 text-center">
        <FileCode className={`mx-auto ${iconSizes['3xl']} text-text-disabled`} />
        <H2 className="mt-4 mb-2">{t('templates.states.emptyTitle')}</H2>
        <P className="text-text-muted">{t('templates.states.emptyDescription')}</P>
        <Button
          tone="violet"
          className="mt-4"
          leftIcon={<Upload className={iconSizes.md} />}
          onClick={onUploadClick}
        >
          {t('templates.states.uploadTemplate')}
        </Button>
      </CardContent>
    </Card>
  );
};

interface NoResultsStateProps {
  show: boolean;
  searchQuery: string;
  onClearSearch: () => void;
}

export const TemplatesNoResultsState: FC<NoResultsStateProps> = ({
  show,
  searchQuery,
  onClearSearch,
}) => {
  const { t } = useTranslation('pages');
  if (!show) {
    return null;
  }

  return (
    <Card className="border-surface-border bg-bg-surface/70">
      <CardContent className="py-12 text-center">
        <Search className={`mx-auto ${iconSizes['3xl']} text-text-disabled`} />
        <H2 className="mt-4 mb-2">{t('templates.states.noMatchTitle')}</H2>
        <P className="text-text-muted">
          {t('templates.states.noMatchDescription', { query: searchQuery })}
        </P>
        <Button variant="outline" className="mt-4" onClick={onClearSearch}>
          {t('templates.states.clearSearch')}
        </Button>
      </CardContent>
    </Card>
  );
};
