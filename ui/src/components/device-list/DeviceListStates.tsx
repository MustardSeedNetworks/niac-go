import { Plus, Search, Server } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router';
import { Alert } from '../../shared/Alert';
import { Button } from '../../shared/Button';
import { Card } from '../../shared/Card';
import { DataTable } from '../../shared/DataTable';
import { EmptyState } from '../../shared/EmptyState';
import { Skeleton } from '../../shared/Skeleton';
import { useTableLabels } from '../../ui/tableLabels';

interface LoadingStateProps {
  viewMode: 'cards' | 'table';
}

const CARD_SKELETONS = ['a', 'b', 'c', 'd', 'e', 'f', 'g', 'h'] as const;

export function DeviceListLoadingState({ viewMode }: LoadingStateProps) {
  const { t } = useTranslation('devices');
  const { t: tCommon } = useTranslation('common');
  const labels = useTableLabels<never>(tCommon('labels.devices').toLowerCase());

  if (viewMode === 'table') {
    return (
      <DataTable
        rows={[]}
        loading
        columns={[
          { key: 'hostname', header: t('list.tableHeaderHostname'), cell: () => null },
          { key: 'type', header: t('list.tableHeaderType'), cell: () => null },
          { key: 'ip', header: t('list.tableHeaderIpAddress'), cell: () => null },
          { key: 'protocols', header: t('list.tableHeaderProtocols'), cell: () => null },
        ]}
        getRowKey={() => ''}
        caption={t('list.listAria')}
        labels={labels}
        empty={null}
      />
    );
  }

  return (
    <div
      role="status"
      aria-label={t('list.listAria')}
      className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4"
    >
      {CARD_SKELETONS.map((key) => (
        <Skeleton key={key} className="h-40 w-full" />
      ))}
    </div>
  );
}

interface ErrorStateProps {
  error: Error;
  onRetry: () => void;
}

export function DeviceListErrorState({ error, onRetry }: ErrorStateProps) {
  const { t } = useTranslation('devices');
  return (
    <Alert tone="error" title={t('list.states.loadErrorTitle')}>
      <p>{error.message}</p>
      <Button variant="secondary" size="sm" className="mt-2" onClick={onRetry}>
        {t('list.states.retry')}
      </Button>
    </Alert>
  );
}

export function DeviceListEmptyState() {
  const { t } = useTranslation('devices');
  const navigate = useNavigate();

  return (
    <Card>
      <EmptyState
        icon={<Server className="h-6 w-6" />}
        title={t('list.states.emptyTitle')}
        hint={t('list.states.emptyDescription')}
        action={
          <Button
            variant="primary"
            icon={<Plus className="h-4 w-4" />}
            onClick={() => navigate('/device-config/new')}
          >
            {t('list.states.addDevice')}
          </Button>
        }
      />
    </Card>
  );
}

interface NoResultsStateProps {
  onClearFilters: () => void;
}

export function DeviceListNoResultsState({ onClearFilters }: NoResultsStateProps) {
  const { t } = useTranslation('devices');
  return (
    <Card>
      <EmptyState
        icon={<Search className="h-6 w-6" />}
        title={t('list.states.noMatchTitle')}
        hint={t('list.states.noMatchDescription')}
        action={
          <Button variant="secondary" onClick={onClearFilters}>
            {t('list.states.clearFilters')}
          </Button>
        }
      />
    </Card>
  );
}
