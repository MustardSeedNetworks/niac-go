import type { FC } from 'react';
import { useTranslation } from 'react-i18next';
import { fetchSessionClients } from '../../api/client';
import type { ObservedClient } from '../../api/types';
import { POLL_INTERVALS } from '../../constants/polling';
import { useApiResource } from '../../hooks/useApiResource';
import { Card, CardContent } from '../../ui/Card';
import { DataTable, type DataTableColumn } from '../../ui/DataTable';
import { H2, SmallText } from '../../ui/Typography';
import { useFormatRelativeTime } from '../../utils/format';

interface AttachedClientsCardProps {
  sessionId: string;
}

/**
 * Who is plugged into the running scenario, and where. Each row is a MAC the
 * session saw on the wire, placed on the pool port the runtime assigned it,
 * so two testers on one scenario read as two cables into two ports.
 */
export const AttachedClientsCard: FC<AttachedClientsCardProps> = ({ sessionId }) => {
  const { t } = useTranslation('pages');
  const { t: tCommon } = useTranslation('common');
  const formatRelative = useFormatRelativeTime();
  const { data: clients, error } = useApiResource(
    () => fetchSessionClients(sessionId),
    ['clients', sessionId],
    { intervalMs: POLL_INTERVALS.medium },
  );

  const notPlaced = t('runtime.clients.notPlaced');
  const columns: DataTableColumn<ObservedClient>[] = [
    {
      key: 'mac',
      header: t('runtime.clients.mac'),
      cellClassName: 'font-mono text-xs text-text-primary',
      cell: (client) => client.mac,
    },
    {
      key: 'ip',
      header: t('runtime.clients.ip'),
      cellClassName: 'font-mono text-xs text-text-muted',
      cell: (client) => client.ip || tCommon('format.dash'),
    },
    {
      key: 'device',
      header: t('runtime.clients.device'),
      'data-testid': 'attached-client-device',
      cell: (client) => client.device || notPlaced,
    },
    {
      key: 'port',
      header: t('runtime.clients.port'),
      'data-testid': 'attached-client-port',
      cell: (client) => client.interface || notPlaced,
    },
    {
      key: 'vlan',
      header: t('runtime.fabric.physicalVlan'),
      cellClassName: 'text-text-muted',
      cell: (client) => client.vlan ?? t('runtime.fabric.untagged'),
    },
    {
      key: 'lastSeen',
      header: t('runtime.clients.lastSeen'),
      cellClassName: 'text-text-muted',
      cell: (client) => formatRelative(client.lastSeen),
    },
  ];

  return (
    <Card data-testid="attached-clients">
      <CardContent className="stack-lg">
        <div>
          <H2>{t('runtime.clients.title')}</H2>
          <SmallText>{t('runtime.clients.help')}</SmallText>
        </div>
        {error ? (
          <SmallText className="text-status-error" role="alert">
            {t('runtime.clients.loadError', { error: error.message })}
          </SmallText>
        ) : (
          <DataTable
            rows={clients ?? []}
            columns={columns}
            getRowKey={(client) => client.mac}
            rowTestId={(client) => `attached-client-${client.mac}`}
            loading={clients === null}
            loadingMessage={t('runtime.clients.loading')}
            emptyMessage={t('runtime.clients.empty')}
          />
        )}
      </CardContent>
    </Card>
  );
};
