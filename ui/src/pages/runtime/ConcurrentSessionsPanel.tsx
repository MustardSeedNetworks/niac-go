import { useTranslation } from 'react-i18next';
import type { SimulationStatus } from '../../api/types';
import { TableSection } from '../../components/TableSection';
import { Button } from '../../shared/Button';
import { DataTable, type DataTableColumn } from '../../shared/DataTable';
import { StatusPill } from '../../shared/StatusPill';
import { ActionButton } from '../../ui/ActionButton';
import { ContainedTable } from '../../ui/ContainedTable';
import { SmallText } from '../../ui/Typography';
import { useTableLabels } from '../../ui/tableLabels';
import { formatNumber } from '../../utils/format';

interface ConcurrentSessionsPanelProps {
  sessions: SimulationStatus[];
  /** The scenario this browser is reading; the daemon has no say in it. */
  selectedSessionId: string | null;
  stoppingSessionId: string | null;
  onSelect: (session: SimulationStatus) => void;
  onStop: (session: SimulationStatus) => void;
}

export function ConcurrentSessionsPanel({
  sessions,
  selectedSessionId,
  stoppingSessionId,
  onSelect,
  onStop,
}: ConcurrentSessionsPanelProps) {
  const { t } = useTranslation('pages');
  const labels = useTableLabels<SimulationStatus>(t('runtime.sessions.title').toLowerCase());

  const columns: DataTableColumn<SimulationStatus>[] = [
    {
      key: 'scenario',
      header: t('runtime.sessions.scenario'),
      sortValue: (session) => session.sessionId ?? '',
      cell: (session) => (
        <span
          className="font-medium text-text-primary"
          data-testid={`session-row-${session.sessionId}`}
        >
          {session.sessionId}
        </span>
      ),
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
            <StatusPill tone="error" label={t('runtime.sessions.degraded')} />
            <SmallText>{session.degradedReason || t('runtime.sessions.degradedHelp')}</SmallText>
          </div>
        ) : (
          <StatusPill
            tone="success"
            label={
              session.sessionId === selectedSessionId
                ? t('runtime.sessions.selected')
                : t('runtime.running.active')
            }
          />
        ),
    },
  ];

  return (
    <TableSection title={t('runtime.sessions.title')} description={t('runtime.sessions.help')}>
      <ContainedTable>
        <DataTable
          rows={sessions}
          columns={columns}
          getRowKey={(session) => String(session.sessionId)}
          caption={t('runtime.sessions.title')}
          labels={labels}
          empty={null}
          rowActions={(session) => (
            <>
              {session.sessionId !== selectedSessionId && (
                <Button
                  variant="secondary"
                  size="sm"
                  data-testid={`session-select-${session.sessionId}`}
                  onClick={() => onSelect(session)}
                >
                  {t('runtime.sessions.select')}
                </Button>
              )}
              <ActionButton
                action="stop"
                variant="destructive"
                size="sm"
                data-testid={`session-stop-${session.sessionId}`}
                loading={stoppingSessionId === session.sessionId}
                onClick={() => onStop(session)}
              >
                {t('runtime.running.stopButton')}
              </ActionButton>
            </>
          )}
        />
      </ContainedTable>
    </TableSection>
  );
}
