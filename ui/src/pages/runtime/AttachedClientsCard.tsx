import { useTranslation } from 'react-i18next';
import { fetchSessionClients } from '../../api/client';
import type { ObservedClient, SimulationStatus } from '../../api/types';
import { TableSection } from '../../components/TableSection';
import { POLL_INTERVALS } from '../../constants/polling';
import { useApiResource } from '../../hooks/useApiResource';
import { Alert } from '../../shared/Alert';
import { DataTable, type DataTableColumn } from '../../shared/DataTable';
import { EmptyState } from '../../shared/EmptyState';
import { useTableLabels } from '../../ui/tableLabels';
import { useFormatRelativeTime } from '../../utils/format';
import { AttachedClientMove } from './AttachedClientMove';
import { sessionPool } from './sessionPool';

interface AttachedClientsCardProps {
  sessionId: string;
  fabric?: SimulationStatus['fabric'];
}

/**
 * Who is plugged into the running scenario, and where. Each row is a MAC the
 * session saw on the wire, placed on the pool port the runtime assigned it,
 * so two testers on one scenario read as two cables into two ports.
 */
export function AttachedClientsCard({ sessionId, fabric }: AttachedClientsCardProps) {
  const { t } = useTranslation('pages');
  const { t: tCommon } = useTranslation('common');
  const formatRelative = useFormatRelativeTime();
  const {
    data: clients,
    error,
    refetch,
  } = useApiResource(() => fetchSessionClients(sessionId), ['clients', sessionId], {
    intervalMs: POLL_INTERVALS.medium,
  });

  const pool = sessionPool(fabric);
  const notPlaced = t('runtime.clients.notPlaced');
  const labels = useTableLabels<ObservedClient>(t('runtime.clients.title').toLowerCase());
  const columns: DataTableColumn<ObservedClient>[] = [
    {
      key: 'mac',
      header: t('runtime.clients.mac'),
      sortValue: (client) => client.mac,
      cell: (client) => <span className="font-mono text-xs text-text-primary">{client.mac}</span>,
    },
    {
      key: 'ip',
      header: t('runtime.clients.ip'),
      cell: (client) => (
        <span className="font-mono text-xs text-text-muted">
          {client.ip || tCommon('format.dash')}
        </span>
      ),
    },
    {
      key: 'device',
      header: t('runtime.clients.device'),
      sortValue: (client) => client.device ?? '',
      cell: (client) => (
        <span data-testid="attached-client-device">{client.device || notPlaced}</span>
      ),
    },
    {
      key: 'port',
      header: t('runtime.clients.port'),
      cell: (client) => (
        <span data-testid="attached-client-port">{client.interface || notPlaced}</span>
      ),
    },
    {
      key: 'vlan',
      header: t('runtime.fabric.physicalVlan'),
      cell: (client) => (
        <span className="text-text-muted">{client.vlan ?? t('runtime.fabric.untagged')}</span>
      ),
    },
    {
      key: 'lastSeen',
      header: t('runtime.clients.lastSeen'),
      cell: (client) => <span className="text-text-muted">{formatRelative(client.lastSeen)}</span>,
    },
  ];

  return (
    <TableSection
      title={t('runtime.clients.title')}
      description={t('runtime.clients.help')}
      data-testid="attached-clients"
    >
      {error ? (
        <Alert tone="error">{t('runtime.clients.loadError', { error: error.message })}</Alert>
      ) : (
        <>
          <DataTable
            rows={clients ?? []}
            columns={columns}
            getRowKey={(client) => client.mac}
            caption={t('runtime.clients.title')}
            labels={labels}
            loading={clients === null}
            empty={<EmptyState title={t('runtime.clients.empty')} />}
          />
          {pool && clients && clients.length > 0 && (
            <AttachedClientMove
              sessionId={sessionId}
              pool={pool}
              clients={clients}
              onMoved={refetch}
            />
          )}
        </>
      )}
    </TableSection>
  );
}
