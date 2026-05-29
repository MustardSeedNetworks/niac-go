import {
  Activity,
  Bot,
  Network,
  PlugZap,
  SatelliteDish,
  Server,
  Terminal,
  Zap,
} from 'lucide-react';
import { type FC, memo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { fetchSimulationStatus } from '../api/client';
import type { ErrorType, HistoryRecord } from '../api/types';
import { POLL_INTERVALS } from '../constants/polling';
import { iconSizes } from '../constants/sizes';
import { useAppState } from '../contexts/AppContext';
import { useApiResource } from '../hooks/useApiResource';
import { Button } from '../ui/Button';
import { Card, CardContent } from '../ui/Card';
import { Tag } from '../ui/Tag';
import { AccentLink, H2, SmallText } from '../ui/Typography';
import { formatDuration, formatNumber, formatTime, formatUptime } from '../utils/format';

/**
 * Dashboard Page - Command Center
 *
 * Live counters, run snapshots, and automation status for the active NIAC stack.
 */
export const DashboardPage: FC = () => {
  const { t } = useTranslation('pages');
  const { data: stats } = useAppState('stats');
  const { data: history } = useAppState('history');
  const { data: errorInfo } = useAppState('errorTypes');
  const { data: simStatus } = useApiResource(fetchSimulationStatus, [], {
    intervalMs: POLL_INTERVALS.fast,
  });
  const [showErrors, setShowErrors] = useState(false);

  const isRunning = simStatus?.running ?? false;
  const uptimeSeconds = simStatus?.uptimeSeconds ?? 0;

  return (
    <div className="stack-xl animate-fade-in">
      {/* Status banner */}
      <Card
        className={`border-l-4 ${isRunning ? 'border-l-status-success' : 'border-l-surface-border'}`}
      >
        <CardContent className="py-4">
          <div className="flex flex-wrap items-center justify-between gap-comfortable">
            <div className="flex items-center gap-comfortable">
              <div className="relative">
                <div
                  className={`h-3 w-3 rounded-full ${isRunning ? 'bg-status-success' : 'bg-bg-muted'}`}
                />
                {isRunning && (
                  <div className="absolute inset-0 h-3 w-3 rounded-full bg-status-success animate-ping opacity-75" />
                )}
              </div>
              <div>
                <p className="font-semibold text-text-primary">
                  {isRunning ? t('dashboard.status.running') : t('dashboard.status.stopped')}
                </p>
                <p className="text-sm text-text-muted">
                  {stats?.interface ?? t('dashboard.status.noInterface')} •{' '}
                  {t('dashboard.status.deviceCount', { count: stats?.deviceCount ?? 0 })}
                </p>
              </div>
            </div>
            <div className="flex items-center gap-compact">
              {uptimeSeconds > 0 && (
                <Tag colorScheme="violet">
                  {t('dashboard.status.uptimeLabel', { uptime: formatUptime(uptimeSeconds) })}
                </Tag>
              )}
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Stat cards row */}
      <div className="grid gap-comfortable sm:grid-cols-2 lg:grid-cols-4">
        <Card hover={true} className="group">
          <CardContent className="stack-xs">
            <div className="flex-between">
              <span className="text-sm font-medium text-text-muted">
                {t('dashboard.stats.devicesOnline')}
              </span>
              <Server
                className={`${iconSizes.lg} text-brand-accent group-hover:scale-110 transition-transform`}
              />
            </div>
            <p className="text-3xl font-bold text-text-primary">{stats?.deviceCount ?? '—'}</p>
            <p className="text-xs text-text-muted">{t('dashboard.stats.devicesOnlineHelper')}</p>
          </CardContent>
        </Card>

        <Card hover={true} className="group">
          <CardContent className="stack-xs">
            <div className="flex-between">
              <span className="text-sm font-medium text-text-muted">
                {t('dashboard.stats.packetsRx')}
              </span>
              <Activity
                className={`${iconSizes.lg} text-status-success group-hover:scale-110 transition-transform`}
              />
            </div>
            <p className="text-3xl font-bold text-text-primary">
              {stats ? formatNumber(stats.stack.packetsReceived) : '—'}
            </p>
            <p className="text-xs text-text-muted">
              {stats
                ? t('dashboard.stats.packetsRxHelper', {
                    sent: formatNumber(stats.stack.packetsSent),
                  })
                : t('dashboard.stats.packetsRxAwaiting')}
            </p>
          </CardContent>
        </Card>

        <Card hover={true} className="group">
          <CardContent className="stack-xs">
            <div className="flex-between">
              <span className="text-sm font-medium text-text-muted">
                {t('dashboard.stats.dnsQueries')}
              </span>
              <SatelliteDish
                className={`${iconSizes.lg} text-status-warning group-hover:scale-110 transition-transform`}
              />
            </div>
            <p className="text-3xl font-bold text-text-primary">
              {stats ? formatNumber(stats.stack.dnsQueries) : '—'}
            </p>
            <p className="text-xs text-text-muted">
              {t('dashboard.stats.dnsQueriesHelper', {
                count: stats ? formatNumber(stats.stack.dhcpRequests) : '—',
              })}
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Main content grid */}
      <div className="grid gap-spacious lg:grid-cols-3">
        {/* Quick Actions */}
        <Card className="lg:col-span-2">
          <CardContent className="stack-lg">
            <H2 className="flex items-center gap-compact">
              <Zap className={`${iconSizes.lg} text-brand-accent`} />
              {t('dashboard.quickActions.title')}
            </H2>
            <div className="grid gap-default sm:grid-cols-2">
              <button
                type="button"
                onClick={() => setShowErrors(!showErrors)}
                className="flex items-center gap-default rounded-lg border border-surface-border bg-surface-hover pad text-left hover:bg-surface-hover hover:border-brand-primary/30 transition-all group"
              >
                <div className="flex-shrink-0 h-10 w-10 rounded-lg bg-status-warning/20 flex-center group-hover:scale-110 transition-transform">
                  <PlugZap className={`${iconSizes.lg} text-status-warning`} />
                </div>
                <div>
                  <p className="font-medium text-text-primary">
                    {t('dashboard.quickActions.errorInjectionLabel')}
                  </p>
                  <p className="text-sm text-text-muted">
                    {t('dashboard.quickActions.errorInjectionDescription')}
                  </p>
                </div>
              </button>

              <AccentLink to="/debug" className="no-underline">
                <div className="flex items-center gap-default rounded-lg border border-surface-border bg-surface-hover pad text-left hover:bg-surface-hover hover:border-brand-primary/30 transition-all group">
                  <div className="flex-shrink-0 h-10 w-10 rounded-lg bg-brand-primary/20 flex-center group-hover:scale-110 transition-transform">
                    <Terminal className={`${iconSizes.lg} text-brand-accent`} />
                  </div>
                  <div>
                    <p className="font-medium text-text-primary">
                      {t('dashboard.quickActions.debugConsoleLabel')}
                    </p>
                    <p className="text-sm text-text-muted">
                      {t('dashboard.quickActions.debugConsoleDescription')}
                    </p>
                  </div>
                </div>
              </AccentLink>

              <AccentLink to="/traffic" className="no-underline">
                <div className="flex items-center gap-default rounded-lg border border-surface-border bg-surface-hover pad text-left hover:bg-surface-hover hover:border-brand-primary/30 transition-all group">
                  <div className="flex-shrink-0 h-10 w-10 rounded-lg bg-status-info/20 flex-center group-hover:scale-110 transition-transform">
                    <Activity className={`${iconSizes.lg} text-status-info`} />
                  </div>
                  <div>
                    <p className="font-medium text-text-primary">
                      {t('dashboard.quickActions.trafficInjectionLabel')}
                    </p>
                    <p className="text-sm text-text-muted">
                      {t('dashboard.quickActions.trafficInjectionDescription')}
                    </p>
                  </div>
                </div>
              </AccentLink>

              <AccentLink to="/topology" className="no-underline">
                <div className="flex items-center gap-default rounded-lg border border-surface-border bg-surface-hover pad text-left hover:bg-surface-hover hover:border-brand-primary/30 transition-all group">
                  <div className="flex-shrink-0 h-10 w-10 rounded-lg bg-status-success/20 flex-center group-hover:scale-110 transition-transform">
                    <Network className={`${iconSizes.lg} text-status-success`} />
                  </div>
                  <div>
                    <p className="font-medium text-text-primary">
                      {t('dashboard.quickActions.viewTopologyLabel')}
                    </p>
                    <p className="text-sm text-text-muted">
                      {t('dashboard.quickActions.viewTopologyDescription')}
                    </p>
                  </div>
                </div>
              </AccentLink>
            </div>

            {showErrors && errorInfo && (
              <ErrorInjectionPanel errorTypes={errorInfo.availableTypes} info={errorInfo.info} />
            )}
          </CardContent>
        </Card>

        {/* Recent Runs */}
        <Card>
          <CardContent className="stack-lg">
            <div className="flex-between">
              <H2>{t('dashboard.recentRuns.title')}</H2>
              <Tag colorScheme="gray">{t('dashboard.recentRuns.tag')}</Tag>
            </div>
            <div className="stack">
              {(history ?? []).slice(0, 4).map((item) => (
                <div
                  key={item.id}
                  className="rounded-lg border border-surface-border bg-bg-base/50 pad-sm hover:border-surface-border transition-colors"
                >
                  <div className="flex-between mb-tight">
                    <p className="font-mono text-xs text-brand-accent">
                      {formatTime(item.startedAt)}
                    </p>
                    <Tag colorScheme="gray" className="text-[10px]">
                      {t('dashboard.recentRuns.deviceCountShort', { count: item.deviceCount })}
                    </Tag>
                  </div>
                  <p className="text-text-primary font-medium text-sm truncate">
                    {item.configName}
                  </p>
                  <div className="flex gap-default mt-tight text-xs text-text-muted">
                    <span>
                      {t('dashboard.recentRuns.rxShort', {
                        count: formatNumber(item.packetsReceived),
                      })}
                    </span>
                    <span>
                      {t('dashboard.recentRuns.txShort', {
                        count: formatNumber(item.packetsSent),
                      })}
                    </span>
                  </div>
                </div>
              ))}
              {history?.length === 0 && (
                <div className="text-center py-6 text-text-muted">
                  <Activity className={`${iconSizes['2xl']} mx-auto mb-2 opacity-50`} />
                  <p className="text-sm">{t('dashboard.recentRuns.empty')}</p>
                </div>
              )}
            </div>
            {history && history.length > 0 && (
              <AccentLink to="/traffic" className="text-sm">
                {t('dashboard.recentRuns.viewAll')}
              </AccentLink>
            )}
          </CardContent>
        </Card>
      </div>

      <AutomationTimeline history={history} />
    </div>
  );
};

/**
 * Error Injection Panel - Shows available error types
 */
const ErrorInjectionPanel = memo(
  ({ errorTypes, info }: { errorTypes: ErrorType[]; info: string }) => {
    const { t } = useTranslation('pages');
    return (
      <div className="mt-content rounded-xl border border-status-warning/20 bg-status-warning/10 pad">
        <div className="mb-heading flex items-start gap-compact">
          <PlugZap className={`mt-0.5 ${iconSizes.lg} text-status-warning`} />
          <div>
            <p className="font-semibold text-status-warning">{t('dashboard.errorPanel.title')}</p>
            <SmallText className="text-status-warning/80">{info}</SmallText>
          </div>
        </div>
        <div className="grid gap-compact sm:grid-cols-2 lg:grid-cols-3">
          {errorTypes.map((errorType) => (
            <div
              key={errorType.type}
              className="rounded-lg border border-surface-border bg-bg-surface/50 pad-sm"
            >
              <p className="font-semibold text-text-primary">{errorType.type}</p>
              <SmallText className="text-text-muted">{errorType.description}</SmallText>
            </div>
          ))}
        </div>
        <div className="mt-heading rounded-lg bg-status-info/20 pad-sm text-sm text-status-info">
          <strong>{t('dashboard.errorPanel.tuiModeLabel')}</strong>{' '}
          {t('dashboard.errorPanel.tuiModeDescription', {
            command: 'niac interactive [interface] [config]',
          })}
        </div>
      </div>
    );
  },
);

ErrorInjectionPanel.displayName = 'ErrorInjectionPanel';

/**
 * Automation Timeline - Shows recent run history
 */
const AutomationTimeline = memo(({ history }: { history: HistoryRecord[] | null }) => {
  const { t } = useTranslation('pages');
  const timeline = (history ?? []).slice(0, 4).map((run) => ({
    title: run.configName,
    detail: t('dashboard.automation.eventDetail', {
      count: run.deviceCount,
      duration: formatDuration(run.duration),
    }),
    time: formatTime(run.startedAt),
  }));

  if (timeline.length === 0) {
    return (
      <Card className="border-surface-border bg-bg-surface/70">
        <CardContent>
          <SmallText className="text-text-muted">{t('dashboard.automation.empty')}</SmallText>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className="border-surface-border bg-bg-surface/70">
      <CardContent className="stack-lg">
        <div className="flex-between">
          <H2 className="flex items-center gap-compact">
            <SatelliteDish className={`${iconSizes.lg} text-brand-accent`} />
            {t('dashboard.automation.title')}
          </H2>
          <Tag colorScheme="gray">{t('dashboard.automation.latestEventsTag')}</Tag>
        </div>
        <div className="stack-lg">
          {timeline.map((event) => (
            <div
              key={event.title}
              className="flex flex-col gap-tight rounded-lg border border-surface-border bg-bg-base/50 pad sm:flex-row sm:items-center sm:justify-between"
            >
              <div>
                <SmallText className="text-status-info">{event.time}</SmallText>
                <p className="font-semibold text-text-primary">{event.title}</p>
                <SmallText className="text-text-muted">{event.detail}</SmallText>
              </div>
              <Button
                variant="ghost"
                size="sm"
                className="mt-inline sm:mt-0"
                leftIcon={<Bot className={iconSizes.md} />}
              >
                {t('dashboard.automation.viewDetails')}
              </Button>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  );
});

AutomationTimeline.displayName = 'AutomationTimeline';

export default DashboardPage;
