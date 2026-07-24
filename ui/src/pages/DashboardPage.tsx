import {
  Activity,
  Network,
  Play,
  PlugZap,
  SatelliteDish,
  Server,
  Terminal,
  Zap,
} from 'lucide-react';
import { type FC, memo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import type { ErrorType } from '../api/types';
import { iconSizes } from '../constants/sizes';
import { useAppState } from '../contexts/AppContext';
import { useSimulationStatus } from '../hooks/useSimulationStatus';
import { Card, CardContent } from '../ui/Card';
import { Tag } from '../ui/Tag';
import { AccentLink, H2 } from '../ui/Typography';
import { formatNumber, formatTime, formatUptime } from '../utils/format';

/**
 * Dashboard Page - Command Center
 *
 * Live counters, run snapshots, and quick actions for the active NIAC stack.
 */
export const DashboardPage: FC = () => {
  const { t } = useTranslation('pages');
  const { data: stats } = useAppState('stats');
  const { data: history } = useAppState('history');
  const { data: errorInfo } = useAppState('errorTypes');
  const { data: simStatus } = useSimulationStatus();
  const [showErrorCatalog, setShowErrorCatalog] = useState(false);

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

        {/* Honest pair: RX and TX are both real, labeled counters — neither
            is smuggled into the other's helper text. */}
        <Card hover={true} className="group">
          <CardContent className="stack-xs">
            <div className="flex-between">
              <span className="text-sm font-medium text-text-muted">
                {t('dashboard.stats.packetsTitle')}
              </span>
              <Activity
                className={`${iconSizes.lg} text-status-success group-hover:scale-110 transition-transform`}
              />
            </div>
            <div className="flex items-end gap-comfortable">
              <div>
                <p className="text-2xl font-bold text-text-primary">
                  {stats ? formatNumber(stats.stack.packetsReceived) : '—'}
                </p>
                <p className="text-xs text-text-muted">{t('dashboard.stats.rxLabel')}</p>
              </div>
              <div>
                <p className="text-2xl font-bold text-text-primary">
                  {stats ? formatNumber(stats.stack.packetsSent) : '—'}
                </p>
                <p className="text-xs text-text-muted">{t('dashboard.stats.txLabel')}</p>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Honest pair: DNS and DHCP are both real, labeled counters. */}
        <Card hover={true} className="group">
          <CardContent className="stack-xs">
            <div className="flex-between">
              <span className="text-sm font-medium text-text-muted">
                {t('dashboard.stats.queriesTitle')}
              </span>
              <SatelliteDish
                className={`${iconSizes.lg} text-status-warning group-hover:scale-110 transition-transform`}
              />
            </div>
            <div className="flex items-end gap-comfortable">
              <div>
                <p className="text-2xl font-bold text-text-primary">
                  {stats ? formatNumber(stats.stack.dnsQueries) : '—'}
                </p>
                <p className="text-xs text-text-muted">{t('dashboard.stats.dnsLabel')}</p>
              </div>
              <div>
                <p className="text-2xl font-bold text-text-primary">
                  {stats ? formatNumber(stats.stack.dhcpRequests) : '—'}
                </p>
                <p className="text-xs text-text-muted">{t('dashboard.stats.dhcpLabel')}</p>
              </div>
            </div>
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
              <AccentLink to="/runtime" className="no-underline">
                <div className="flex items-center gap-default rounded-lg border border-surface-border bg-surface-hover pad text-left hover:bg-surface-hover hover:border-brand-primary/30 transition-all group">
                  <div className="flex-shrink-0 h-10 w-10 rounded-lg bg-status-success/20 flex-center group-hover:scale-110 transition-transform">
                    <Play className={`${iconSizes.lg} text-status-success`} />
                  </div>
                  <div>
                    <p className="font-medium text-text-primary">
                      {t('dashboard.quickActions.startSimulationLabel')}
                    </p>
                    <p className="text-sm text-text-muted">
                      {t('dashboard.quickActions.startSimulationDescription')}
                    </p>
                  </div>
                </div>
              </AccentLink>

              <button
                type="button"
                onClick={() => setShowErrorCatalog(!showErrorCatalog)}
                aria-expanded={showErrorCatalog}
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

            {showErrorCatalog && errorInfo && (
              <ErrorTypeCatalog errorTypes={errorInfo.availableTypes} info={errorInfo.info} />
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
                      {t('dashboard.recentRuns.deviceCountShort', { value: item.deviceCount })}
                    </Tag>
                  </div>
                  <p className="text-text-primary font-medium text-sm truncate">
                    {item.configName}
                  </p>
                  <div className="flex gap-default mt-tight text-xs text-text-muted">
                    <span>
                      {t('dashboard.recentRuns.rxShort', {
                        value: formatNumber(item.packetsReceived),
                      })}
                    </span>
                    <span>
                      {t('dashboard.recentRuns.txShort', {
                        value: formatNumber(item.packetsSent),
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
              <AccentLink to="/traffic#recent-runs" className="text-sm">
                {t('dashboard.recentRuns.viewAll')}
              </AccentLink>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
};

/**
 * Error Type Catalog - Each entry deep-links to the real fault-injection
 * surface (/traffic) with its error type preselected via query param,
 * rather than duplicating the injection form here. One injection surface,
 * not two.
 */
const ErrorTypeCatalog = memo(({ errorTypes, info }: { errorTypes: ErrorType[]; info: string }) => {
  const { t } = useTranslation('pages');
  return (
    <div className="mt-content rounded-xl border border-status-warning/20 bg-status-warning/10 pad">
      <div className="mb-heading flex items-start gap-compact">
        <PlugZap className={`mt-0.5 ${iconSizes.lg} text-status-warning`} />
        <div>
          <p className="font-semibold text-status-warning">{t('dashboard.errorPanel.title')}</p>
          <p className="text-sm text-status-warning/80">{info}</p>
          <p className="text-xs text-status-warning/70 mt-tight">
            {t('dashboard.errorPanel.clickHint')}
          </p>
        </div>
      </div>
      <div className="grid gap-compact sm:grid-cols-2 lg:grid-cols-3">
        {errorTypes.map((errorType) => (
          <AccentLink
            key={errorType.type}
            to={`/traffic?errorType=${encodeURIComponent(errorType.type)}`}
            className="block no-underline rounded-lg border border-surface-border bg-bg-surface/50 pad-sm hover:border-brand-primary/30 transition-colors"
          >
            <p className="font-semibold text-text-primary">{errorType.type}</p>
            <p className="text-sm text-text-muted">{errorType.description}</p>
          </AccentLink>
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
});

ErrorTypeCatalog.displayName = 'ErrorTypeCatalog';

export default DashboardPage;
