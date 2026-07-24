import { AlertCircle, Plus, Search, Server } from 'lucide-react';
import type { FC } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router';
import { iconSizes } from '../../constants/sizes';
import { Button } from '../../ui/Button';
import { Card, CardContent } from '../../ui/Card';
import { DeviceCardGridSkeleton, DeviceTableSkeleton } from '../../ui/Skeleton';
import { H2, P, SmallText } from '../../ui/Typography';

interface LoadingStateProps {
  viewMode: 'cards' | 'table';
}

export const DeviceListLoadingState: FC<LoadingStateProps> = ({ viewMode }) => {
  const { t } = useTranslation('devices');

  if (viewMode === 'table') {
    return (
      <Card className="border-surface-border bg-bg-surface/70">
        <CardContent className="p-0">
          {/* Table header skeleton */}
          <div className="flex items-center gap-comfortable border-b border-surface-border px-4 py-row-lg bg-bg-base/40">
            <div className="h-4 w-4 rounded bg-bg-elevated/50" />
            <div className="flex-1 grid grid-cols-12 gap-comfortable text-sm font-medium text-text-muted">
              <div className="col-span-3">{t('list.tableHeaderHostname')}</div>
              <div className="col-span-2">{t('list.tableHeaderType')}</div>
              <div className="col-span-2">{t('list.tableHeaderIpAddress')}</div>
              <div className="col-span-3">{t('list.tableHeaderProtocols')}</div>
              <div className="col-span-2 text-right">{t('list.tableHeaderActions')}</div>
            </div>
          </div>
          <DeviceTableSkeleton rows={8} />
        </CardContent>
      </Card>
    );
  }

  return <DeviceCardGridSkeleton count={8} />;
};

interface ErrorStateProps {
  error: Error;
  onRetry: () => void;
}

export const DeviceListErrorState: FC<ErrorStateProps> = ({ error, onRetry }) => {
  const { t } = useTranslation('devices');
  return (
    <Card className="border-status-error/30 bg-status-error/20" role="alert" aria-live="assertive">
      <CardContent className="stack">
        <div className="flex items-start gap-default">
          <AlertCircle className="mt-tight h-5 w-5 text-status-error" />
          <div>
            <p className="font-semibold text-status-error">{t('list.states.loadErrorTitle')}</p>
            <SmallText className="text-status-error/90">{error.message}</SmallText>
            <Button variant="outline" size="sm" className="mt-heading" onClick={onRetry}>
              {t('list.states.retry')}
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  );
};

export const DeviceListEmptyState: FC = () => {
  const { t } = useTranslation('devices');
  const navigate = useNavigate();

  return (
    <Card className="border-surface-border bg-bg-surface/70">
      <CardContent className="py-centered text-center">
        <Server className="mx-auto h-12 w-12 text-text-disabled" />
        <H2 className="mt-content mb-2">{t('list.states.emptyTitle')}</H2>
        <P className="text-text-muted">{t('list.states.emptyDescription')}</P>
        <Button
          tone="violet"
          className="mt-content"
          leftIcon={<Plus className={iconSizes.md} />}
          onClick={() => navigate('/device-config/new')}
        >
          {t('list.states.addDevice')}
        </Button>
      </CardContent>
    </Card>
  );
};

interface NoResultsStateProps {
  onClearFilters: () => void;
}

export const DeviceListNoResultsState: FC<NoResultsStateProps> = ({ onClearFilters }) => {
  const { t } = useTranslation('devices');
  return (
    <Card className="border-surface-border bg-bg-surface/70">
      <CardContent className="py-centered text-center">
        <Search className="mx-auto h-12 w-12 text-text-disabled" />
        <H2 className="mt-content mb-2">{t('list.states.noMatchTitle')}</H2>
        <P className="text-text-muted">{t('list.states.noMatchDescription')}</P>
        <Button variant="outline" className="mt-content" onClick={onClearFilters}>
          {t('list.states.clearFilters')}
        </Button>
      </CardContent>
    </Card>
  );
};
