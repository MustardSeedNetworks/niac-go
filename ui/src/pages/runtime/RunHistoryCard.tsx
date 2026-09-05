import { History } from 'lucide-react';
import { type FC, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { useLocation } from 'react-router';
import { fetchHistory } from '../../api/client';
import { POLL_INTERVALS } from '../../constants/polling';
import { iconSizes } from '../../constants/sizes';
import { useApiResource } from '../../hooks/useApiResource';
import { Card, CardContent } from '../../ui/Card';
import { Tag } from '../../ui/Tag';
import { H2, SmallText } from '../../ui/Typography';
import { formatDuration, formatNumber, formatTime } from '../../utils/format';

const RECENT_RUN_LIMIT = 5;

/**
 * Run history for the daemon, alongside the session controls that produce it.
 * The TUI's history viewer was the only other place this existed; the web UI is
 * now the single home for it.
 */
export const RunHistoryCard: FC = () => {
  const { t } = useTranslation('pages');
  const { data: history, error } = useApiResource(fetchHistory, [], {
    intervalMs: POLL_INTERVALS.slow,
  });
  const location = useLocation();

  // The dashboard's "view all" link carries #recent-runs so it lands on the
  // history itself rather than the top of the session controls.
  useEffect(() => {
    if (location.hash === '#recent-runs') {
      document.getElementById('recent-runs')?.scrollIntoView({ behavior: 'smooth' });
    }
  }, [location.hash]);

  return (
    <Card id="recent-runs" className="border-surface-border bg-bg-surface/70">
      <CardContent className="stack">
        <div className="flex-between">
          <H2 className="flex items-center gap-compact text-lg">
            <History className={`${iconSizes.lg} text-text-muted`} />
            {t('runtime.recentRunsTitle')}
          </H2>
          <Tag colorScheme="gray">{t('runtime.recentRunsTag')}</Tag>
        </div>
        <SmallText className="text-text-muted">{t('runtime.recentRunsDescription')}</SmallText>
        <div className="stack-sm text-sm text-text-secondary">
          {(history ?? []).slice(0, RECENT_RUN_LIMIT).map((item) => (
            <div
              key={item.id}
              className="rounded-lg border border-surface-border bg-bg-base/50 pad-sm"
            >
              <p className="text-text-primary font-semibold">{item.configName}</p>
              <SmallText className="text-text-muted">
                {t('runtime.recentRunStats', {
                  time: formatTime(item.startedAt),
                  duration: formatDuration(item.duration),
                  rx: formatNumber(item.packetsReceived),
                  tx: formatNumber(item.packetsSent),
                })}
              </SmallText>
            </div>
          ))}
          {error ? (
            // "No runs recorded" and "the history could not be read" are
            // different facts, and showing the first for the second tells an
            // operator their runs were not saved.
            <SmallText role="alert" className="text-status-error">
              {t('runtime.recentRunsFailed', { error: error.message })}
            </SmallText>
          ) : (
            (!history || history.length === 0) && (
              <SmallText className="text-text-muted italic">
                {t('runtime.noCapturedRuns')}
              </SmallText>
            )
          )}
        </div>
      </CardContent>
    </Card>
  );
};
