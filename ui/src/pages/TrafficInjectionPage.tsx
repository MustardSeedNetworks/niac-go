import { History } from 'lucide-react';
import type { FC } from 'react';
import { fetchHistory } from '../api/client';
import { ErrorInjectionPanel } from '../components/ErrorInjectionPanel';
import { ReplayControlPanel } from '../components/ReplayControlPanel';
import { POLL_INTERVALS } from '../constants/polling';
import { iconSizes } from '../constants/sizes';
import { useApiResource } from '../hooks/useApiResource';
import { Card, CardContent } from '../ui/Card';
import { Tag } from '../ui/Tag';
import { H2, P, SmallText } from '../ui/Typography';
import { formatDuration, formatNumber, formatTime } from '../utils/format';

export const TrafficInjectionPage: FC = () => {
  const { data: history } = useApiResource(fetchHistory, [], {
    intervalMs: POLL_INTERVALS.slow,
  });

  return (
    <div className="space-y-8">
      {/* Error Injection */}
      <div className="space-y-4">
        <div>
          <H2>Error Injection</H2>
          <P className="text-text-muted">
            Inject network errors on device interfaces for testing and simulation scenarios.
          </P>
        </div>
        <ErrorInjectionPanel />
      </div>

      {/* PCAP Replay */}
      <div className="space-y-4">
        <div>
          <H2>PCAP Replay</H2>
          <P className="text-text-muted">
            Replay captured packet traffic with loop and timing controls for testing.
          </P>
        </div>
        <ReplayControlPanel />
      </div>

      {/* Recent runs — previously lived on the standalone /analysis page */}
      <Card className="border-white/5 bg-bg-surface/70">
        <CardContent className="space-y-3">
          <div className="flex items-center justify-between">
            <H2 className="flex items-center gap-2 text-lg">
              <History className={`${iconSizes.lg} text-text-muted`} />
              Recent runs
            </H2>
            <Tag colorScheme="gray">History</Tag>
          </div>
          <SmallText className="text-text-muted">
            Past simulation runs the daemon recorded.
          </SmallText>
          <div className="space-y-2 text-sm text-text-secondary">
            {(history ?? []).slice(0, 5).map((item) => (
              <div key={item.id} className="rounded-lg border border-white/5 bg-bg-base/50 p-3">
                <p className="text-text-primary font-semibold">{item.configName}</p>
                <SmallText className="text-text-muted">
                  {formatTime(item.startedAt)} · duration {formatDuration(item.duration)} · RX{' '}
                  {formatNumber(item.packetsReceived)} · TX {formatNumber(item.packetsSent)}
                </SmallText>
              </div>
            ))}
            {(!history || history.length === 0) && (
              <SmallText className="text-text-muted italic">No captured runs yet.</SmallText>
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  );
};
