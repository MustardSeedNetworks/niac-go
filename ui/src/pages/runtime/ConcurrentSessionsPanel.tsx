import type { FC } from 'react';
import { useTranslation } from 'react-i18next';
import type { SimulationStatus } from '../../api/types';
import { Button } from '../../ui/Button';
import { Card, CardContent } from '../../ui/Card';
import { DataTable, type DataTableColumn } from '../../ui/DataTable';
import { Tag } from '../../ui/Tag';
import { H2, SmallText } from '../../ui/Typography';
import { formatNumber } from '../../utils/format';

interface ConcurrentSessionsPanelProps {
  sessions: SimulationStatus[];
  /** The scenario this browser is reading; the daemon has no say in it. */
  selectedSessionId: string | null;
  stoppingSessionId: string | null;
  onSelect: (session: SimulationStatus) => void;
  onStop: (session: SimulationStatus) => void;
}

export const ConcurrentSessionsPanel: FC<ConcurrentSessionsPanelProps> = ({
  sessions,
  selectedSessionId,
  stoppingSessionId,
  onSelect,
  onStop,
}) => {
  const { t } = useTranslation('pages');

  const columns: DataTableColumn<SimulationStatus>[] = [
    {
      key: 'scenario',
      header: t('runtime.sessions.scenario'),
      cellClassName: 'font-medium text-text-primary',
      cell: (session) => session.sessionId,
    },
    {
      key: 'vlan',
      header: t('runtime.sessions.vlan'),
      cell: (session) => session.physicalVlan ?? t('runtime.fabric.untagged'),
    },
    {
      key: 'devices',
      header: t('runtime.sessions.devices'),
      cell: (session) => session.deviceCount,
    },
    {
      key: 'started',
      header: t('runtime.sessions.started'),
      cell: (session) =>
        session.startedAt ? new Date(session.startedAt).toLocaleString() : '\u2014',
    },
    {
      key: 'packets',
      header: t('runtime.sessions.packets'),
      cell: (session) =>
        t('runtime.sessions.packetCounts', {
          received: formatNumber(session.fabric?.received ?? 0),
          transmitted: formatNumber(session.fabric?.transmitted ?? 0),
        }),
    },
    {
      key: 'state',
      header: t('runtime.sessions.state'),
      cell: (session) =>
        session.degraded ? (
          <div className="stack-compact">
            <Tag colorScheme="red">{t('runtime.sessions.degraded')}</Tag>
            <SmallText>{session.degradedReason || t('runtime.sessions.degradedHelp')}</SmallText>
          </div>
        ) : (
          <Tag colorScheme="green">
            {session.sessionId === selectedSessionId
              ? t('runtime.sessions.selected')
              : t('runtime.running.active')}
          </Tag>
        ),
    },
    {
      key: 'actions',
      header: t('runtime.sessions.actions'),
      cellClassName: 'flex gap-compact',
      cell: (session) => (
        <>
          {session.sessionId !== selectedSessionId && (
            <Button
              size="xs"
              variant="outline"
              data-testid={`session-select-${session.sessionId}`}
              onClick={() => onSelect(session)}
            >
              {t('runtime.sessions.select')}
            </Button>
          )}
          <Button
            size="xs"
            variant="outline"
            tone="red"
            data-testid={`session-stop-${session.sessionId}`}
            loading={stoppingSessionId === session.sessionId}
            onClick={() => onStop(session)}
            action="stop"
          >
            {t('runtime.running.stopButton')}
          </Button>
        </>
      ),
    },
  ];

  return (
    <Card>
      <CardContent className="stack-lg">
        <div>
          <H2>{t('runtime.sessions.title')}</H2>
          <SmallText>{t('runtime.sessions.help')}</SmallText>
        </div>
        <DataTable
          rows={sessions}
          columns={columns}
          getRowKey={(session) => String(session.sessionId)}
          rowTestId={(session) => `session-row-${session.sessionId}`}
          emptyMessage={null}
        />
      </CardContent>
    </Card>
  );
};
