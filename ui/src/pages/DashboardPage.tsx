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
import { fetchErrorTypes, fetchHistory, fetchSimulationStatus, fetchStats } from '../api/client';
import type { ErrorType, HistoryRecord } from '../api/types';
import { POLL_INTERVALS } from '../constants/polling';
import { useApiResource } from '../hooks/useApiResource';
import { Button } from '../ui/Button';
import { Card, CardContent } from '../ui/Card';
import { Tag } from '../ui/Tag';
import { AccentLink, H2, SmallText } from '../ui/Typography';
import { formatNumber, formatTime, formatUptime } from '../utils/format';

/**
 * Format duration string with fallback
 */
function formatDuration(value: string): string {
  return value || '—';
}

/**
 * Dashboard Page - Command Center
 *
 * Live counters, run snapshots, and automation status for the active NIAC stack.
 */
export const DashboardPage: FC = () => {
  const { data: stats } = useApiResource(fetchStats, [], {
    intervalMs: POLL_INTERVALS.medium,
  });
  const { data: history } = useApiResource(fetchHistory, [], {
    intervalMs: POLL_INTERVALS.slow,
  });
  const { data: errorInfo } = useApiResource(fetchErrorTypes, []);
  const { data: simStatus } = useApiResource(fetchSimulationStatus, [], {
    intervalMs: POLL_INTERVALS.fast,
  });
  const [showErrors, setShowErrors] = useState(false);

  const isRunning = simStatus?.running ?? false;
  const uptimeSeconds = simStatus?.uptimeSeconds ?? 0;

  return (
    <div className="space-y-6 animate-fade-in">
      {/* Status banner */}
      <Card className={`border-l-4 ${isRunning ? 'border-l-emerald-500' : 'border-l-gray-500'}`}>
        <CardContent className="py-4">
          <div className="flex flex-wrap items-center justify-between gap-4">
            <div className="flex items-center gap-4">
              <div className="relative">
                <div
                  className={`h-3 w-3 rounded-full ${isRunning ? 'bg-emerald-500' : 'bg-gray-500'}`}
                />
                {isRunning && (
                  <div className="absolute inset-0 h-3 w-3 rounded-full bg-emerald-500 animate-ping opacity-75" />
                )}
              </div>
              <div>
                <p className="font-semibold text-white">
                  {isRunning ? 'Simulation Running' : 'Simulation Stopped'}
                </p>
                <p className="text-sm text-gray-400">
                  {stats?.interface ?? 'No interface'} • {stats?.deviceCount ?? 0} devices
                </p>
              </div>
            </div>
            <div className="flex items-center gap-2">
              {uptimeSeconds > 0 && (
                <Tag colorScheme="violet">Uptime: {formatUptime(uptimeSeconds)}</Tag>
              )}
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Stat cards row */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <Card hover={true} className="group">
          <CardContent className="space-y-1">
            <div className="flex items-center justify-between">
              <span className="text-sm font-medium text-gray-400">Devices Online</span>
              <Server className="h-5 w-5 text-violet-400 group-hover:scale-110 transition-transform" />
            </div>
            <p className="text-3xl font-bold text-white">{stats?.deviceCount ?? '—'}</p>
            <p className="text-xs text-gray-500">Active network devices</p>
          </CardContent>
        </Card>

        <Card hover={true} className="group">
          <CardContent className="space-y-1">
            <div className="flex items-center justify-between">
              <span className="text-sm font-medium text-gray-400">Packets RX</span>
              <Activity className="h-5 w-5 text-emerald-400 group-hover:scale-110 transition-transform" />
            </div>
            <p className="text-3xl font-bold text-white">
              {stats ? formatNumber(stats.stack.packetsReceived) : '—'}
            </p>
            <p className="text-xs text-gray-500">
              {stats ? `${formatNumber(stats.stack.packetsSent)} sent` : 'Awaiting data'}
            </p>
          </CardContent>
        </Card>

        <Card hover={true} className="group">
          <CardContent className="space-y-1">
            <div className="flex items-center justify-between">
              <span className="text-sm font-medium text-gray-400">DNS Queries</span>
              <SatelliteDish className="h-5 w-5 text-amber-400 group-hover:scale-110 transition-transform" />
            </div>
            <p className="text-3xl font-bold text-white">
              {stats ? formatNumber(stats.stack.dnsQueries) : '—'}
            </p>
            <p className="text-xs text-gray-500">
              DHCP: {stats ? formatNumber(stats.stack.dhcpRequests) : '—'}
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Main content grid */}
      <div className="grid gap-6 lg:grid-cols-3">
        {/* Quick Actions */}
        <Card className="lg:col-span-2">
          <CardContent className="space-y-4">
            <H2 className="mb-0 flex items-center gap-2">
              <Zap className="h-5 w-5 text-violet-400" />
              Quick Actions
            </H2>
            <div className="grid gap-3 sm:grid-cols-2">
              <button
                type="button"
                onClick={() => setShowErrors(!showErrors)}
                className="flex items-center gap-3 rounded-lg border border-white/10 bg-white/5 p-4 text-left hover:bg-white/10 hover:border-violet-500/30 transition-all group"
              >
                <div className="flex-shrink-0 h-10 w-10 rounded-lg bg-yellow-500/20 flex items-center justify-center group-hover:scale-110 transition-transform">
                  <PlugZap className="h-5 w-5 text-yellow-400" />
                </div>
                <div>
                  <p className="font-medium text-white">Error Injection</p>
                  <p className="text-sm text-gray-400">Inject network errors</p>
                </div>
              </button>

              <AccentLink to="/debug" className="no-underline">
                <div className="flex items-center gap-3 rounded-lg border border-white/10 bg-white/5 p-4 text-left hover:bg-white/10 hover:border-violet-500/30 transition-all group">
                  <div className="flex-shrink-0 h-10 w-10 rounded-lg bg-violet-500/20 flex items-center justify-center group-hover:scale-110 transition-transform">
                    <Terminal className="h-5 w-5 text-violet-400" />
                  </div>
                  <div>
                    <p className="font-medium text-white">Debug Console</p>
                    <p className="text-sm text-gray-400">View live logs</p>
                  </div>
                </div>
              </AccentLink>

              <AccentLink to="/traffic" className="no-underline">
                <div className="flex items-center gap-3 rounded-lg border border-white/10 bg-white/5 p-4 text-left hover:bg-white/10 hover:border-violet-500/30 transition-all group">
                  <div className="flex-shrink-0 h-10 w-10 rounded-lg bg-blue-500/20 flex items-center justify-center group-hover:scale-110 transition-transform">
                    <Activity className="h-5 w-5 text-blue-400" />
                  </div>
                  <div>
                    <p className="font-medium text-white">Traffic Injection</p>
                    <p className="text-sm text-gray-400">Replay PCAP files</p>
                  </div>
                </div>
              </AccentLink>

              <AccentLink to="/topology" className="no-underline">
                <div className="flex items-center gap-3 rounded-lg border border-white/10 bg-white/5 p-4 text-left hover:bg-white/10 hover:border-violet-500/30 transition-all group">
                  <div className="flex-shrink-0 h-10 w-10 rounded-lg bg-emerald-500/20 flex items-center justify-center group-hover:scale-110 transition-transform">
                    <Network className="h-5 w-5 text-emerald-400" />
                  </div>
                  <div>
                    <p className="font-medium text-white">View Topology</p>
                    <p className="text-sm text-gray-400">Network graph</p>
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
          <CardContent className="space-y-4">
            <div className="flex items-center justify-between">
              <H2 className="mb-0">Recent Runs</H2>
              <Tag colorScheme="gray">History</Tag>
            </div>
            <div className="space-y-3">
              {(history ?? []).slice(0, 4).map((item) => (
                <div
                  key={item.id}
                  className="rounded-lg border border-white/5 bg-gray-950/50 p-3 hover:border-white/10 transition-colors"
                >
                  <div className="flex items-center justify-between mb-1">
                    <p className="font-mono text-xs text-violet-300">
                      {formatTime(item.startedAt)}
                    </p>
                    <Tag colorScheme="gray" className="text-[10px]">
                      {item.deviceCount} dev
                    </Tag>
                  </div>
                  <p className="text-white font-medium text-sm truncate">{item.configName}</p>
                  <div className="flex gap-3 mt-1 text-xs text-gray-500">
                    <span>RX {formatNumber(item.packetsReceived)}</span>
                    <span>TX {formatNumber(item.packetsSent)}</span>
                  </div>
                </div>
              ))}
              {history?.length === 0 && (
                <div className="text-center py-6 text-gray-500">
                  <Activity className="h-8 w-8 mx-auto mb-2 opacity-50" />
                  <p className="text-sm">No run history yet</p>
                </div>
              )}
            </div>
            {history && history.length > 0 && (
              <AccentLink to="/analysis" className="text-sm">
                View all history →
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
  ({ errorTypes, info }: { errorTypes: ErrorType[]; info: string }) => (
    <div className="mt-4 rounded-xl border border-yellow-500/20 bg-yellow-900/10 p-4">
      <div className="mb-3 flex items-start gap-2">
        <PlugZap className="mt-0.5 h-5 w-5 text-yellow-400" />
        <div>
          <p className="font-semibold text-yellow-200">Error Injection Types</p>
          <SmallText className="text-yellow-300/80">{info}</SmallText>
        </div>
      </div>
      <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
        {errorTypes.map((errorType) => (
          <div
            key={errorType.type}
            className="rounded-lg border border-white/10 bg-gray-900/50 p-3"
          >
            <p className="font-semibold text-white">{errorType.type}</p>
            <SmallText className="text-gray-400">{errorType.description}</SmallText>
          </div>
        ))}
      </div>
      <div className="mt-3 rounded-lg bg-blue-900/20 p-3 text-sm text-blue-200">
        <strong>TUI Mode:</strong> Run{' '}
        <code className="rounded bg-black/30 px-1.5 py-0.5 font-mono text-xs">
          niac interactive [interface] [config]
        </code>{' '}
        to access interactive error injection with keyboard controls (press 'i' for menu, keys 1-7
        for quick injection).
      </div>
    </div>
  ),
);

ErrorInjectionPanel.displayName = 'ErrorInjectionPanel';

/**
 * Automation Timeline - Shows recent run history
 */
const AutomationTimeline = memo(({ history }: { history: HistoryRecord[] | null }) => {
  const timeline = (history ?? []).slice(0, 4).map((run) => ({
    id: run.id,
    title: run.configName,
    detail: `${run.deviceCount} devices • duration ${formatDuration(run.duration)}`,
    time: formatTime(run.startedAt),
  }));

  if (timeline.length === 0) {
    return (
      <Card className="border-white/5 bg-gray-900/70">
        <CardContent>
          <SmallText className="text-gray-400">
            Automation updates will appear after the first run completes.
          </SmallText>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className="border-white/5 bg-gray-900/70">
      <CardContent className="space-y-4">
        <div className="flex items-center justify-between">
          <H2 className="mb-0 flex items-center gap-2">
            <SatelliteDish className="h-5 w-5 text-violet-300" />
            Automation timeline
          </H2>
          <Tag colorScheme="gray">Latest events</Tag>
        </div>
        <div className="space-y-4">
          {timeline.map((event) => (
            <div
              key={event.id}
              className="flex flex-col gap-1 rounded-lg border border-white/5 bg-gray-950/50 p-4 sm:flex-row sm:items-center sm:justify-between"
            >
              <div>
                <SmallText className="text-blue-300">{event.time}</SmallText>
                <p className="font-semibold text-white">{event.title}</p>
                <SmallText className="text-gray-400">{event.detail}</SmallText>
              </div>
              <Button
                variant="ghost"
                size="sm"
                className="mt-2 sm:mt-0"
                leftIcon={<Bot className="h-4 w-4" />}
              >
                View details
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
