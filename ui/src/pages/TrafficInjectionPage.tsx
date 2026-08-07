import { History } from 'lucide-react';
import { type FC, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { useLocation } from 'react-router';
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
  const { t } = useTranslation('pages');
  const { data: history } = useApiResource(fetchHistory, [], {
    intervalMs: POLL_INTERVALS.slow,
  });
  const location = useLocation();

  // Dashboard's "View all history" link points here with #recent-runs so
  // it lands on the actual expanded run history instead of the top of a
  // page whose primary content is fault injection.
  useEffect(() => {
    if (location.hash === '#recent-runs') {
      document.getElementById('recent-runs')?.scrollIntoView({ behavior: 'smooth' });
    }
  }, [location.hash]);

  return (
    <div className="space-y-8">
      {/* Error Injection */}
      <div className="stack-lg">
        <div>
          <H2>{t('traffic.page.errorInjectionTitle')}</H2>
          <P className="text-text-muted">{t('traffic.page.errorInjectionDescription')}</P>
        </div>
        <ErrorInjectionPanel />
      </div>

      {/* PCAP Replay */}
      <div className="stack-lg">
        <div>
          <H2>{t('traffic.page.pcapReplayTitle')}</H2>
          <P className="text-text-muted">{t('traffic.page.pcapReplayDescription')}</P>
        </div>
        <ReplayControlPanel />
      </div>

      {/* Recent runs — previously lived on the standalone /analysis page */}
      <Card id="recent-runs" className="border-surface-border bg-bg-surface/70">
        <CardContent className="stack">
          <div className="flex-between">
            <H2 className="flex items-center gap-compact text-lg">
              <History className={`${iconSizes.lg} text-text-muted`} />
              {t('traffic.page.recentRunsTitle')}
            </H2>
            <Tag colorScheme="gray">{t('traffic.page.recentRunsTag')}</Tag>
          </div>
          <SmallText className="text-text-muted">
            {t('traffic.page.recentRunsDescription')}
          </SmallText>
          <div className="stack-sm text-sm text-text-secondary">
            {(history ?? []).slice(0, 5).map((item) => (
              <div
                key={item.id}
                className="rounded-lg border border-surface-border bg-bg-base/50 pad-sm"
              >
                <p className="text-text-primary font-semibold">{item.configName}</p>
                <SmallText className="text-text-muted">
                  {t('traffic.page.recentRunStats', {
                    time: formatTime(item.startedAt),
                    duration: formatDuration(item.duration),
                    rx: formatNumber(item.packetsReceived),
                    tx: formatNumber(item.packetsSent),
                  })}
                </SmallText>
              </div>
            ))}
            {(!history || history.length === 0) && (
              <SmallText className="text-text-muted italic">
                {t('traffic.page.noCapturedRuns')}
              </SmallText>
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  );
};
