import {
  Activity,
  ChevronDown,
  ChevronRight,
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
import { LinkButton } from '../ui/Button';
import { Card, CardContent } from '../ui/Card';
import { type RollupState, StatusRollup } from '../ui/StatusRollup';
import { Tag } from '../ui/Tag';
import { AccentLink, H2 } from '../ui/Typography';
import { formatNumber, formatTime, formatUptime } from '../utils/format';

/**
 * quickActions is the destination list, not the copy: each tile is named by
 * the title of the page it opens, read from the same locale key the page
 * header renders, so the dashboard cannot call a page something the page does
 * not call itself (#2186). Only the one-line description is written here.
 * DashboardPage.test.tsx resolves each tile's expected name from its `path`,
 * so a `titleKey` naming a different page fails there, and it holds the page's
 * own rule that one label may not appear twice (see the rollup action below).
 */
const quickActions = [
  {
    path: '/traffic',
    titleKey: 'traffic.title',
    icon: PlugZap,
    iconClass: 'text-status-warning',
    tileClass: 'bg-status-warning/20',
    descriptionKey: 'dashboard.quickActions.faultInjectionDescription',
  },
  {
    path: '/debug',
    titleKey: 'debug.title',
    icon: Terminal,
    iconClass: 'text-brand-accent',
    tileClass: 'bg-brand-primary/20',
    descriptionKey: 'dashboard.quickActions.debugConsoleDescription',
  },
  {
    path: '/topology',
    titleKey: 'topology.title',
    icon: Network,
    iconClass: 'text-status-success',
    tileClass: 'bg-status-success/20',
    descriptionKey: 'dashboard.quickActions.viewTopologyDescription',
  },
] as const;

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
  const { data: simStatus, loading: simLoading } = useSimulationStatus();
  const [showErrorCatalog, setShowErrorCatalog] = useState(false);

  const isRunning = simStatus?.running ?? false;
  const uptimeSeconds = simStatus?.uptimeSeconds ?? 0;

  const rollupState: RollupState =
    simLoading || !simStatus ? 'unknown' : simStatus.degraded ? 'warn' : 'ok';

  const rollupHeadline = simLoading
    ? t('dashboard.rollup.checking')
    : !simStatus
      ? t('dashboard.rollup.noDaemon')
      : simStatus.degraded
        ? (simStatus.degradedReason ?? t('dashboard.rollup.degraded'))
        : isRunning
          ? t('dashboard.rollup.running', {
              count: simStatus.deviceCount,
              iface: simStatus.interface ?? t('dashboard.status.noInterface'),
            })
          : t('dashboard.rollup.idle');

  const rollupBody = simLoading
    ? undefined
    : !simStatus
      ? t('dashboard.rollup.noDaemonBody')
      : simStatus.degraded
        ? t('dashboard.rollup.degradedBody')
        : isRunning
          ? undefined
          : t('dashboard.rollup.idleBody');

  return (
    <div className="stack-xl animate-fade-in">
      {/* Overview opens with the rollup rather than the counters: the first
          question here is whether the stack is healthy, and a row of numbers
          does not answer it.

          Until the daemon answers, its counters are not zero — they are
          unmeasurable, so the rollup prints em dashes instead. A daemon that
          never answers is the same case: NIAC with no daemon has nothing to
          report, which is not the same as a simulation reporting nothing.

          A running-but-degraded stack is the one state the daemon names
          itself, so its own reason becomes the headline. Idle is calm rather
          than green — no simulation running is the normal resting state, not
          a success. */}
      <StatusRollup
        state={rollupState}
        headline={rollupHeadline}
        body={rollupBody}
        figures={[
          {
            label: t('dashboard.rollup.devicesLabel'),
            value: simStatus ? String(simStatus.deviceCount) : '—',
          },
          {
            label: t('dashboard.rollup.uptimeLabel'),
            value: uptimeSeconds > 0 ? formatUptime(uptimeSeconds) : '—',
          },
        ]}
        /* The overview's one primary action. It sits on the rollup because
           that is where the page answers "is anything running". It replaces
           the quick-action tile of the same name: the same label twice on
           one page, once as a shortcut among equals, is not a primary. */
        actions={
          <LinkButton to="/runtime" leftIcon={<Play className={iconSizes.md} />}>
            {t('dashboard.quickActions.startSimulationLabel')}
          </LinkButton>
        }
      />

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
            <div className="grid gap-default sm:grid-cols-2" data-testid="quick-actions">
              {quickActions.map((action) => {
                const Icon = action.icon;
                return (
                  <AccentLink key={action.path} to={action.path} className="no-underline">
                    <div className="flex items-center gap-default rounded-lg border border-surface-border bg-surface-hover pad text-left hover:bg-surface-hover hover:border-brand-primary/30 transition-all group">
                      <div
                        className={`flex-shrink-0 h-10 w-10 rounded-lg ${action.tileClass} flex-center group-hover:scale-110 transition-transform`}
                      >
                        <Icon className={`${iconSizes.lg} ${action.iconClass}`} />
                      </div>
                      <div>
                        <p className="font-medium text-text-primary">{t(action.titleKey)}</p>
                        <p className="text-sm text-text-muted">{t(action.descriptionKey)}</p>
                      </div>
                    </div>
                  </AccentLink>
                );
              })}
            </div>

            {errorInfo && (
              <ErrorTypeCatalog
                errorTypes={errorInfo.availableTypes}
                info={errorInfo.info}
                expanded={showErrorCatalog}
                onToggle={() => setShowErrorCatalog(!showErrorCatalog)}
              />
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
                  <p className="text-text-primary font-medium text-sm wrap-anywhere">
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
              <AccentLink to="/runtime#recent-runs" className="text-sm">
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
const ErrorTypeCatalog = memo(
  ({
    errorTypes,
    info,
    expanded,
    onToggle,
  }: {
    errorTypes: ErrorType[];
    info: string;
    expanded: boolean;
    onToggle: () => void;
  }) => {
    const { t } = useTranslation('pages');
    return (
      <div className="mt-content rounded-xl border border-status-warning/20 bg-status-warning/10 pad">
        <button
          type="button"
          onClick={onToggle}
          aria-expanded={expanded}
          className="flex w-full items-start gap-compact text-left focus:outline-none focus:ring-2 focus:ring-brand-primary rounded-lg"
        >
          {expanded ? (
            <ChevronDown className={`mt-0.5 ${iconSizes.lg} text-status-warning`} />
          ) : (
            <ChevronRight className={`mt-0.5 ${iconSizes.lg} text-status-warning`} />
          )}
          <div>
            <p className="font-semibold text-status-warning-strong">
              {t('dashboard.errorPanel.title')}
            </p>
            <p className="text-sm text-status-warning-strong">{info}</p>
            <p className="text-xs text-status-warning-strong mt-tight">
              {t('dashboard.errorPanel.clickHint')}
            </p>
          </div>
        </button>
        {expanded && (
          <div className="mt-heading grid gap-compact sm:grid-cols-2 lg:grid-cols-3">
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
        )}
      </div>
    );
  },
);

ErrorTypeCatalog.displayName = 'ErrorTypeCatalog';

export default DashboardPage;
