import type { FC } from 'react';
import { useTranslation } from 'react-i18next';
import type { SimulationStatus } from '../../api/types';
import { Button } from '../../ui/Button';
import { Card, CardContent } from '../../ui/Card';
import { Tag } from '../../ui/Tag';
import { H2, SmallText } from '../../ui/Typography';
import { formatNumber } from '../../utils/format';

interface ConcurrentSessionsPanelProps {
  sessions: SimulationStatus[];
  stoppingSessionId: string | null;
  selectingSessionId: string | null;
  onSelect: (session: SimulationStatus) => void;
  onStop: (session: SimulationStatus) => void;
}

export const ConcurrentSessionsPanel: FC<ConcurrentSessionsPanelProps> = ({
  sessions,
  stoppingSessionId,
  selectingSessionId,
  onSelect,
  onStop,
}) => {
  const { t } = useTranslation('pages');
  return (
    <Card>
      <CardContent className="stack-lg">
        <div>
          <H2>{t('runtime.sessions.title')}</H2>
          <SmallText>{t('runtime.sessions.help')}</SmallText>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead className="text-xs uppercase text-text-muted">
              <tr>
                <th className="px-cell py-row">{t('runtime.sessions.scenario')}</th>
                <th className="px-cell py-row">{t('runtime.sessions.vlan')}</th>
                <th className="px-cell py-row">{t('runtime.sessions.devices')}</th>
                <th className="px-cell py-row">{t('runtime.sessions.started')}</th>
                <th className="px-cell py-row">{t('runtime.sessions.packets')}</th>
                <th className="px-cell py-row">{t('runtime.sessions.state')}</th>
                <th className="px-cell py-row">{t('runtime.sessions.actions')}</th>
              </tr>
            </thead>
            <tbody>
              {sessions.map((session) => (
                <tr
                  key={session.sessionId}
                  data-testid={`session-row-${session.sessionId}`}
                  className="border-t border-surface-border"
                >
                  <td className="px-cell py-row font-medium text-text-primary">
                    {session.sessionId}
                  </td>
                  <td className="px-cell py-row text-text-secondary">
                    {session.physicalVlan ?? t('runtime.fabric.untagged')}
                  </td>
                  <td className="px-cell py-row text-text-secondary">{session.deviceCount}</td>
                  <td className="px-cell py-row text-text-secondary">
                    {session.startedAt ? new Date(session.startedAt).toLocaleString() : '—'}
                  </td>
                  <td className="px-cell py-row text-text-secondary">
                    {t('runtime.sessions.packetCounts', {
                      received: formatNumber(session.fabric?.received ?? 0),
                      transmitted: formatNumber(session.fabric?.transmitted ?? 0),
                    })}
                  </td>
                  <td className="px-cell py-row">
                    {session.degraded ? (
                      <div className="stack-compact">
                        <Tag colorScheme="red">{t('runtime.sessions.degraded')}</Tag>
                        <SmallText>
                          {session.degradedReason || t('runtime.sessions.degradedHelp')}
                        </SmallText>
                      </div>
                    ) : (
                      <Tag colorScheme="green">
                        {session.selected
                          ? t('runtime.sessions.selected')
                          : t('runtime.running.active')}
                      </Tag>
                    )}
                  </td>
                  <td className="px-cell py-row flex gap-compact">
                    {!session.selected && (
                      <Button
                        size="xs"
                        variant="outline"
                        data-testid={`session-select-${session.sessionId}`}
                        loading={selectingSessionId === session.sessionId}
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
                    >
                      {t('runtime.running.stopButton')}
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </CardContent>
    </Card>
  );
};
