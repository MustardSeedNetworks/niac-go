import { createElement, memo, useCallback, useMemo, type ChangeEvent, type FC, type ReactNode, useEffect, useState } from 'react';
import { Routes, Route, Navigate } from 'react-router-dom';
import type { LucideIcon } from 'lucide-react';
import {
  Activity,
  Server,
  Network,
  LineChart,
  Workflow,
  ShieldCheck,
  PlugZap,
  Bot,
  BellRing,
  SatelliteDish,
  FileCog,
  Zap,
} from 'lucide-react';
import {
  PageShell,
  PrimaryNav,
  PageHeader,
  Card,
  CardContent,
  Button,
  Tag,
  H2,
  P,
  SmallText,
  AccentLink,
} from '@krisarmstrong/web-foundation';
import type { NavItem } from '@krisarmstrong/web-foundation';
import { useApiResource } from './hooks/useApiResource';
import { useVirtualScroll } from './hooks/useVirtualScroll';
import {
  fetchStats,
  fetchDevices,
  fetchHistory,
  fetchNeighbors,
  fetchConfig,
  updateConfig,
  fetchReplayStatus,
  startReplay,
  stopReplay,
  fetchAlerts,
  updateAlerts,
  fetchFiles,
  fetchVersion,
  fetchTopology,
  fetchErrorTypes,
  fetchInterfaces,
  fetchSimulationStatus,
  startSimulation,
  stopSimulation,
} from './api/client';
import type { DeviceSummary, HistoryRecord, NeighborRecord, AlertConfig, ReplayRequest, FileEntry, TopologyGraph, ErrorType, NetworkInterface } from './api/types';
import { TrafficInjectionPage } from './pages/TrafficInjectionPage';
import './App.css';

type PageConfig = {
  path: string;
  label: string;
  title: string;
  description: string;
  icon: LucideIcon;
  Component: FC;
  badge?: string;
};

const pages: PageConfig[] = [
  {
    path: '/',
    label: 'Command Center',
    title: 'Command Center',
    description: 'Live counters, run snapshots, and automation status for the active NIAC stack.',
    icon: Activity,
    Component: DashboardPage,
  },
  {
    path: '/runtime',
    label: 'Runtime Control',
    title: 'Runtime Control',
    description: 'Monitor runtime status, view network interfaces, and manage NIAC configuration.',
    icon: PlugZap,
    Component: RuntimeControlPage,
  },
  {
    path: '/devices',
    label: 'Devices & Config',
    title: 'Devices & Configuration',
    description: 'Review YAML-derived devices, SNMP walks, DHCP/DNS personas, and packet playback targets.',
    icon: Server,
    Component: DevicesPage,
  },
  {
    path: '/topology',
    label: 'Topology & Neighbors',
    title: 'Topology & Neighbor Insight',
    description: 'LLDP/CDP/EDP/FDP visibility for verifying intent before exporting to Graphviz.',
    icon: Network,
    Component: TopologyPage,
  },
  {
    path: '/analysis',
    label: 'Analysis',
    title: 'Analysis & Playback',
    description: 'Replay PCAPs, inspect SNMP walks, and publish bundles directly from the UI.',
    icon: LineChart,
    Component: AnalysisPage,
  },
  {
    path: '/automation',
    label: 'Automation',
    title: 'Automation & Alerts',
    description: 'Configure alert thresholds, webhook targets, and future workflow automations.',
    icon: Workflow,
    Component: AutomationPage,
    badge: 'Beta',
  },
  {
    path: '/traffic',
    label: 'Traffic Injection',
    title: 'Traffic & Error Injection',
    description: 'Inject network errors and replay PCAP traffic for testing and simulation.',
    icon: Zap,
    Component: TrafficInjectionPage,
  },
];

const navItems: NavItem[] = pages.map((page) => ({
  label: page.label,
  path: page.path,
  icon: createElement(page.icon, { className: 'h-4 w-4' }),
  badge: page.badge,
}));

// Polling intervals (in milliseconds)
const POLL_INTERVALS = {
  FAST: 2000,      // 2s - Real-time simulation status
  MEDIUM: 5000,    // 5s - Live stats
  SLOW: 15000,     // 15s - Historical data
  VERY_SLOW: 60000, // 1m - Static data like version
} as const;

export default function App() {
  const { data: version } = useApiResource(fetchVersion, [], { intervalMs: POLL_INTERVALS.VERY_SLOW });

  return (
    <PageShell>
      <div className="space-y-10">
        <div className="rounded-2xl border border-white/5 bg-gray-900/60 p-4 backdrop-blur">
          <PrimaryNav items={navItems} className="flex-wrap gap-2" />
        </div>

        <Routes>
          {pages.map((page) => (
            <Route
              key={page.path}
              path={page.path}
              element={
                <PageTemplate page={page}>
                  <page.Component />
                </PageTemplate>
              }
            />
          ))}
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>

        {version && (
          <div className="rounded-xl border border-white/5 bg-gray-900/40 px-4 py-3 text-center text-sm text-gray-400">
            NIAC-Go {version.version} • Network In A Can
          </div>
        )}
      </div>
    </PageShell>
  );
}

// FEATURE #125: Memoize PageTemplate to prevent unnecessary re-renders
const PageTemplate = memo(({ page, children }: { page: PageConfig; children: ReactNode }) => {
  return (
    <section className="space-y-6">
      <PageHeader icon={page.icon} title={page.title} description={page.description} />
      {children}
    </section>
  );
});

function RuntimeControlPage() {
  const [refetchTrigger, setRefetchTrigger] = useState(0);
  const { data: simStatus } = useApiResource(fetchSimulationStatus, [refetchTrigger], { intervalMs: POLL_INTERVALS.FAST });
  const { data: interfaces } = useApiResource(fetchInterfaces, []);

  const [selectedInterface, setSelectedInterface] = useState('');
  const [configPath, setConfigPath] = useState('');
  const [configFile, setConfigFile] = useState<File | null>(null);
  const [starting, setStarting] = useState(false);
  const [stopping, setStopping] = useState(false);
  const [message, setMessage] = useState<{ tone: 'success' | 'error'; text: string } | null>(null);

  const isDaemonMode = simStatus !== null; // If endpoint responds, we're in daemon mode

  // FEATURE #125: Memoize handlers to prevent unnecessary re-renders
  const handleStart = useCallback(async () => {
    if (!selectedInterface && !configPath && !configFile) {
      setMessage({ tone: 'error', text: 'Please select an interface and provide a config file' });
      return;
    }

    setStarting(true);
    setMessage(null);

    try {
      let configData: string | undefined;

      if (configFile) {
        configData = await fileToText(configFile);
      }

      await startSimulation({
        interface: selectedInterface,
        config_path: configPath || undefined,
        config_data: configData,
      });

      setMessage({ tone: 'success', text: 'Simulation started successfully!' });
      setRefetchTrigger((t) => t + 1);
      setConfigFile(null);
    } catch (err) {
      setMessage({ tone: 'error', text: (err as Error).message });
    } finally {
      setStarting(false);
    }
  }, [selectedInterface, configPath, configFile]);

  const handleStop = useCallback(async () => {
    // Confirm before stopping simulation
    if (!window.confirm('Are you sure you want to stop the simulation? This will interrupt the current run.')) {
      return;
    }

    setStopping(true);
    setMessage(null);

    try {
      await stopSimulation();
      setMessage({ tone: 'success', text: 'Simulation stopped' });
      setRefetchTrigger((t) => t + 1);
    } catch (err) {
      setMessage({ tone: 'error', text: getErrorMessage(err) });
    } finally {
      setStopping(false);
    }
  }, []);

  return (
    <div className="space-y-6">
      {!isDaemonMode && (
        <Card className="border-yellow-500/30 bg-yellow-900/20">
          <CardContent className="space-y-3">
            <div className="flex items-start gap-3">
              <BellRing className="mt-1 h-5 w-5 text-yellow-400" />
              <div>
                <p className="font-semibold text-yellow-200">Daemon Mode Not Detected</p>
                <SmallText className="text-yellow-300/90">
                  To use simulation controls, start NIAC in daemon mode:
                </SmallText>
                <code className="mt-2 block rounded bg-black/40 p-3 font-mono text-sm text-yellow-100">
                  niac daemon --listen :8080 --token yourtoken
                </code>
                <SmallText className="mt-2 text-yellow-300/80">
                  Legacy mode (<code>niac --api-listen</code>) doesn't support start/stop controls.
                </SmallText>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {isDaemonMode && !simStatus?.running && (
        <Card className="border-white/5 bg-gradient-to-br from-violet-900/30 to-gray-900/70">
          <CardContent className="space-y-5">
            <div className="flex items-center gap-3">
              <PlugZap className="h-6 w-6 text-violet-400" />
              <H2 className="mb-0">Start Simulation</H2>
            </div>

            <div className="space-y-4">
              <div>
                <label htmlFor="network-interface" className="block text-sm text-gray-400">
                  Network Interface *
                </label>
                <select
                  id="network-interface"
                  className="mt-1 w-full rounded-lg border border-white/10 bg-gray-950/60 p-2 text-sm text-white focus:border-violet-400 focus:outline-none"
                  value={selectedInterface}
                  onChange={(e) => setSelectedInterface(e.target.value)}
                  aria-required="true"
                  aria-label="Select network interface for simulation"
                >
                  <option value="">Select interface...</option>
                  {interfaces?.interfaces.map((iface) => (
                    <option key={iface.name} value={iface.name}>
                      {iface.name} {iface.description ? `- ${iface.description}` : ''} {iface.addresses.length > 0 ? `(${iface.addresses[0]})` : ''}
                    </option>
                  ))}
                </select>
              </div>

              <div>
                <label htmlFor="config-path" className="block text-sm text-gray-400">
                  Config File Path
                </label>
                <input
                  id="config-path"
                  type="text"
                  className="mt-1 w-full rounded-lg border border-white/10 bg-gray-950/60 p-2 font-mono text-sm text-white focus:border-violet-400 focus:outline-none"
                  placeholder="/path/to/config.yaml"
                  value={configPath}
                  onChange={(e) => setConfigPath(e.target.value)}
                  aria-describedby="config-path-help"
                />
                <SmallText id="config-path-help" className="text-gray-500">
                  Or upload a config file below
                </SmallText>
              </div>

              <div>
                <label htmlFor="config-file-upload" className="block text-sm text-gray-400">
                  Upload Config File
                </label>
                <input
                  id="config-file-upload"
                  type="file"
                  accept=".yaml,.yml"
                  className="mt-1 w-full cursor-pointer rounded-lg border border-dashed border-white/10 bg-gray-950/40 p-2 text-sm text-white file:mr-3 file:rounded-md file:border-0 file:bg-violet-600 file:px-3 file:py-1 file:text-sm file:font-medium"
                  onChange={(e) => {
                    const file = e.target.files?.[0];
                    if (!file) {
                      setConfigFile(null);
                      return;
                    }

                    // Validate file size (10MB limit)
                    const MAX_SIZE = 10 * 1024 * 1024;
                    if (file.size > MAX_SIZE) {
                      setMessage({
                        tone: 'error',
                        text: `File too large. Maximum size is ${formatBytes(MAX_SIZE)}`
                      });
                      e.target.value = '';
                      return;
                    }

                    // Validate file type
                    if (!file.name.match(/\.(yaml|yml)$/i)) {
                      setMessage({
                        tone: 'error',
                        text: 'Please select a YAML file (.yaml or .yml)'
                      });
                      e.target.value = '';
                      return;
                    }

                    setConfigFile(file);
                  }}
                  aria-describedby="config-file-help"
                />
                {configFile && (
                  <SmallText id="config-file-help" className="mt-1 text-green-400">
                    Selected: {configFile.name}
                  </SmallText>
                )}
              </div>

              {message && (
                <SmallText
                  className={message.tone === 'success' ? 'text-emerald-300' : 'text-red-400'}
                  role="alert"
                  aria-live="polite"
                >
                  {message.text}
                </SmallText>
              )}

              <Button
                tone="violet"
                size="lg"
                disabled={!selectedInterface || (!configPath && !configFile) || starting}
                onClick={handleStart}
                leftIcon={<Activity className="h-5 w-5" />}
              >
                {starting ? 'Starting...' : 'Start Simulation'}
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      {isDaemonMode && simStatus?.running && (
        <Card className="border-green-500/30 bg-gradient-to-br from-green-900/30 to-gray-900/70">
          <CardContent className="space-y-5">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className="h-3 w-3 animate-pulse rounded-full bg-green-400" />
                <H2 className="mb-0">Simulation Running</H2>
              </div>
              <Tag colorScheme="green">ACTIVE</Tag>
            </div>

            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
              <StatBlock label="Interface" value={simStatus.interface || '—'} helper="Network interface" />
              <StatBlock label="Config" value={simStatus.config_name || '—'} helper={simStatus.config_path || 'Configuration file'} />
              <StatBlock label="Devices" value={simStatus.device_count.toString()} helper="Simulated devices" />
              <StatBlock label="Uptime" value={formatUptime(simStatus.uptime_seconds)} helper="Time running" />
              <StatBlock label="Started" value={simStatus.started_at ? formatTime(simStatus.started_at) : '—'} helper="Start time" />
            </div>

            {message && (
              <SmallText className={message.tone === 'success' ? 'text-emerald-300' : 'text-red-400'}>
                {message.text}
              </SmallText>
            )}

            <div className="flex flex-wrap gap-3">
              <Button
                variant="outline"
                disabled={stopping}
                onClick={handleStop}
                leftIcon={<Activity className="h-4 w-4" />}
              >
                {stopping ? 'Stopping...' : 'Stop Simulation'}
              </Button>
              <Button variant="ghost" leftIcon={<FileCog className="h-4 w-4" />} onClick={() => window.location.href = '/devices'}>
                View Devices
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      {interfaces && (
        <Card className="border-white/5 bg-gray-900/70">
          <CardContent className="space-y-4">
            <H2 className="mb-0 flex items-center gap-2">
              <Network className="h-5 w-5 text-cyan-300" />
              Available Network Interfaces
            </H2>
            <P className="text-gray-300">
              Network interfaces available on this system. Select one to use for simulation.
            </P>
            <InterfaceList interfaces={interfaces.interfaces} currentInterface={interfaces.current_interface} />
          </CardContent>
        </Card>
      )}
    </div>
  );
}

function InterfaceList({ interfaces }: { interfaces: NetworkInterface[]; currentInterface: string }) {
  if (!interfaces.length) {
    return <SmallText className="text-gray-400">No network interfaces found.</SmallText>;
  }

  return (
    <div className="space-y-2">
      {interfaces.map((iface) => (
        <div
          key={iface.name}
          className={`rounded-lg border p-4 ${
            iface.current
              ? 'border-violet-500/50 bg-violet-900/20'
              : 'border-white/10 bg-gray-950/50'
          }`}
        >
          <div className="flex items-center justify-between">
            <div>
              <div className="flex items-center gap-2">
                <p className="font-semibold text-white">{iface.name}</p>
                {iface.current && <Tag colorScheme="purple">ACTIVE</Tag>}
              </div>
              {iface.description && <SmallText className="text-gray-400">{iface.description}</SmallText>}
              {iface.addresses.length > 0 && (
                <div className="mt-1 flex flex-wrap gap-2">
                  {iface.addresses.map((addr) => (
                    <code key={addr} className="rounded bg-black/30 px-2 py-0.5 font-mono text-xs text-blue-300">
                      {addr}
                    </code>
                  ))}
                </div>
              )}
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}

function DashboardPage() {
  const { data: stats, loading: statsLoading, error: statsError } = useApiResource(fetchStats, [], {
    intervalMs: POLL_INTERVALS.MEDIUM,
  });
  const { data: neighbors } = useApiResource(fetchNeighbors, [], { intervalMs: POLL_INTERVALS.MEDIUM });
  const { data: history } = useApiResource(fetchHistory, [], { intervalMs: POLL_INTERVALS.SLOW });
  const { data: errorInfo } = useApiResource(fetchErrorTypes, []);
  const [showErrors, setShowErrors] = useState(false);

  // FEATURE #125: Memoize telemetry data to prevent recalculation on every render
  const telemetry = useMemo(() => [
    {
      label: 'Neighbors learned',
      value: neighbors ? `${neighbors.length}` : '—',
      detail: 'LLDP/CDP/EDP/FDP',
    },
    {
      label: 'Packets received',
      value: stats ? formatNumber(stats.stack.packets_received) : '—',
      detail: stats ? `${formatNumber(stats.stack.packets_sent)} sent` : 'Awaiting counters',
    },
    {
      label: 'Devices online',
      value: stats ? `${stats.device_count}` : '—',
      detail: stats?.interface ?? 'interface',
    },
  ], [neighbors, stats]);

  return (
    <div className="space-y-6">
      <div className="grid gap-6 lg:grid-cols-3">
        <Card className="lg:col-span-2 border-white/5 bg-gradient-to-br from-gray-900/70 to-gray-950/80">
          <CardContent className="space-y-5">
            <div className="flex flex-wrap items-center gap-3">
              <Tag>{stats ? 'RUNNING' : 'AWAITING DATA'}</Tag>
              <Tag colorScheme="purple">Devices: {stats?.device_count ?? '–'}</Tag>
            </div>
            <H2 className="mb-2">{stats?.interface ?? 'NIAC interface'}</H2>
            <P className="text-gray-300">
              {statsLoading && 'Collecting runtime statistics...'}
              {statsError && 'Unable to load statistics from the NIAC runtime.'}
              {!statsLoading && !statsError &&
                'NIAC is publishing counters, neighbor data, and run history through the REST API. The Web UI polls a few times a minute so it mirrors the CLI/TUI view.'}
            </P>
            <div className="grid gap-4 sm:grid-cols-3">
              {telemetry.map((item) => (
                <StatBlock key={item.label} label={item.label} value={item.value} helper={item.detail} />
              ))}
            </div>
            <div className="flex flex-wrap gap-3">
              <Button tone="violet" leftIcon={<Activity className="h-4 w-4" />}>Open live logs</Button>
              <Button variant="outline" leftIcon={<PlugZap className="h-4 w-4" />} onClick={() => setShowErrors(!showErrors)}>
                {showErrors ? 'Hide' : 'Show'} error injection
              </Button>
              <Button variant="ghost" leftIcon={<ShieldCheck className="h-4 w-4" />}>Trigger alert test</Button>
            </div>
            {showErrors && errorInfo && <ErrorInjectionPanel errorTypes={errorInfo.available_types} info={errorInfo.info} />}
          </CardContent>
        </Card>
        <Card className="border-white/5 bg-gray-900/70">
          <CardContent className="space-y-4">
            <div className="flex items-center justify-between">
              <H2 className="mb-0">Recent runs</H2>
              <Tag colorScheme="gray">BoltDB</Tag>
            </div>
            <div className="space-y-3">
              {(history ?? []).slice(0, 3).map((item) => (
                <div key={item.id} className="rounded-lg border border-white/5 bg-gray-950/50 p-3">
                  <p className="font-mono text-sm text-blue-200">{formatTime(item.started_at)}</p>
                  <p className="text-white font-semibold">{item.config_name}</p>
                  <SmallText className="text-gray-400">
                    {item.device_count} devices · RX {formatNumber(item.packets_received)} · TX {formatNumber(item.packets_sent)}
                  </SmallText>
                </div>
              ))}
              {!history?.length && <SmallText className="text-gray-400">No run history yet.</SmallText>}
            </div>
            <AccentLink to="/analysis" className="text-indigo-300">
              See all history →
            </AccentLink>
          </CardContent>
        </Card>
      </div>

      <AutomationTimeline history={history} />
    </div>
  );
}

function ErrorInjectionPanel({ errorTypes, info }: { errorTypes: ErrorType[]; info: string }) {
  return (
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
          <div key={errorType.type} className="rounded-lg border border-white/10 bg-gray-900/50 p-3">
            <p className="font-semibold text-white">{errorType.type}</p>
            <SmallText className="text-gray-400">{errorType.description}</SmallText>
          </div>
        ))}
      </div>
      <div className="mt-3 rounded-lg bg-blue-900/20 p-3 text-sm text-blue-200">
        <strong>TUI Mode:</strong> Run <code className="rounded bg-black/30 px-1.5 py-0.5 font-mono text-xs">niac interactive [interface] [config]</code> to access interactive error injection with keyboard controls (press 'i' for menu, keys 1-7 for quick injection).
      </div>
    </div>
  );
}

function AutomationTimeline({ history }: { history: HistoryRecord[] | null }) {
  const timeline = (history ?? []).slice(0, 4).map((run) => ({
    title: run.config_name,
    detail: `${run.device_count} devices • duration ${formatDuration(run.duration)}`,
    time: formatTime(run.started_at),
  }));

  if (!timeline.length) {
    return (
      <Card className="border-white/5 bg-gray-900/70">
        <CardContent>
          <SmallText className="text-gray-400">Automation updates will appear after the first run completes.</SmallText>
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
            <div key={event.title} className="flex flex-col gap-1 rounded-lg border border-white/5 bg-gray-950/50 p-4 sm:flex-row sm:items-center sm:justify-between">
              <div>
                <SmallText className="text-blue-300">{event.time}</SmallText>
                <p className="font-semibold text-white">{event.title}</p>
                <SmallText className="text-gray-400">{event.detail}</SmallText>
              </div>
              <Button variant="ghost" size="sm" className="mt-2 sm:mt-0" leftIcon={<Bot className="h-4 w-4" />}>
                View details
              </Button>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  );
}

function DevicesPage() {
  return (
    <div className="grid gap-6 xl:grid-cols-2">
      <DeviceListCard />
      <ConfigEditorCard />
    </div>
  );
}

function DeviceListCard() {
  const { data: devices, loading, error } = useApiResource(fetchDevices, [], { intervalMs: POLL_INTERVALS.SLOW });

  return (
    <Card className="border-white/5 bg-gray-900/70">
      <CardContent className="space-y-4">
        <H2 className="mb-0 flex items-center gap-2">
          <Server className="h-5 w-5 text-cyan-300" />
          Config workspace
        </H2>
        {loading && <SmallText className="text-gray-400">Loading devices...</SmallText>}
        {error && <SmallText className="text-red-400">Unable to load devices: {error.message}</SmallText>}
        {!loading && !error && <DeviceTable devices={devices ?? []} />}
        <SmallText className="text-gray-400">
          Devices are rendered directly from the active YAML config so the CLI/TUI and Web UI always agree.
        </SmallText>
      </CardContent>
    </Card>
  );
}

// FEATURE #125 & #126: Memoized DeviceTable with virtual scrolling for large device lists
const DeviceTable = memo(({ devices }: { devices: DeviceSummary[] }) => {
  // FEATURE #126: Use virtual scrolling for 100+ devices
  const useVirtualization = devices.length >= 100;
  const virtualScroll = useVirtualScroll(devices, {
    itemHeight: 60, // Approximate row height in pixels
    containerHeight: 600, // Max viewport height
    overscan: 5,
  });

  if (!devices.length) {
    return (
      <div className="rounded-xl border border-white/5 bg-gray-950/50 p-8 text-center text-gray-400">
        No devices defined in the loaded configuration.
      </div>
    );
  }

  if (!useVirtualization) {
    // Standard rendering for small lists
    return (
      <div className="overflow-x-auto rounded-xl border border-white/5">
        <table className="min-w-full divide-y divide-white/10 text-sm">
          <thead className="bg-gray-900/60 text-xs uppercase tracking-wide text-gray-400">
            <tr>
              <th className="px-4 py-3 text-left">Device</th>
              <th className="px-4 py-3 text-left">Type</th>
              <th className="px-4 py-3 text-left">IP addresses</th>
              <th className="px-4 py-3 text-left">Protocols</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-white/5 text-gray-300">
            {devices.map((device) => (
              <DeviceRow key={device.name} device={device} />
            ))}
          </tbody>
        </table>
      </div>
    );
  }

  // Virtual scrolling for large lists
  return (
    <div className="rounded-xl border border-white/5">
      <div className="bg-gray-900/60 px-4 py-2 text-xs text-gray-400">
        Showing {virtualScroll.visibleItems.length} of {devices.length} devices (virtual scrolling enabled)
      </div>
      <div {...virtualScroll.containerProps} className="overflow-auto">
        <div {...virtualScroll.spacerProps}>
          <div {...virtualScroll.contentProps}>
            <table className="min-w-full divide-y divide-white/10 text-sm">
              <thead className="bg-gray-900/60 text-xs uppercase tracking-wide text-gray-400">
                <tr>
                  <th className="px-4 py-3 text-left">Device</th>
                  <th className="px-4 py-3 text-left">Type</th>
                  <th className="px-4 py-3 text-left">IP addresses</th>
                  <th className="px-4 py-3 text-left">Protocols</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-white/5 text-gray-300">
                {virtualScroll.visibleItems.map(({ item: device }) => (
                  <DeviceRow key={device.name} device={device} />
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  );
});

// FEATURE #125: Memoize individual device row
const DeviceRow = memo(({ device }: { device: DeviceSummary }) => (
  <tr>
    <td className="px-4 py-3 font-semibold text-white">{device.name}</td>
    <td className="px-4 py-3">{device.type}</td>
    <td className="px-4 py-3 font-mono text-xs">{device.ips.join(', ') || '—'}</td>
    <td className="px-4 py-3">
      <div className="flex flex-wrap gap-2">
        {device.protocols.map((proto) => (
          <Tag key={`${device.name}-${proto}`} colorScheme="purple">
            {proto}
          </Tag>
        ))}
        {!device.protocols.length && <SmallText className="text-gray-400">None</SmallText>}
      </div>
    </td>
  </tr>
));

function ConfigEditorCard() {
  const { data, loading, error } = useApiResource(fetchConfig, [], { intervalMs: POLL_INTERVALS.VERY_SLOW });
  const { data: walkFiles } = useApiResource(() => fetchFiles('walks'), [], { intervalMs: POLL_INTERVALS.VERY_SLOW });
  const [value, setValue] = useState('');
  const [dirty, setDirty] = useState(false);
  const [saving, setSaving] = useState(false);
  const [status, setStatus] = useState<{ tone: 'success' | 'error'; message: string } | null>(null);

  useEffect(() => {
    if (data && !dirty) {
      setValue(data.content);
    }
  }, [data, dirty]);

  const handleChange = (event: ChangeEvent<HTMLTextAreaElement>) => {
    setValue(event.target.value);
    setDirty(true);
    setStatus(null);
  };

  const handleReset = () => {
    if (data) {
      setValue(data.content);
      setDirty(false);
      setStatus(null);
    }
  };

  const handleSave = async () => {
    if (!dirty || saving) {
      return;
    }
    setSaving(true);
    setStatus(null);
    try {
      const updated = await updateConfig({ content: value });
      setValue(updated.content);
      setDirty(false);
      setStatus({ tone: 'success', message: 'Configuration saved' });
    } catch (err) {
      setStatus({ tone: 'error', message: (err as Error).message });
    } finally {
      setSaving(false);
    }
  };

  const handleWalkCopy = async (path: string) => {
    try {
      await copyToClipboard(path);
      setStatus({ tone: 'success', message: `Copied ${path}` });
    } catch (err) {
      setStatus({ tone: 'error', message: (err as Error).message || 'Unable to copy path' });
    }
  };

  return (
    <Card className="border-white/5 bg-gray-900/70">
      <CardContent className="space-y-4">
        <H2 className="mb-0 flex items-center gap-2">
          <FileCog className="h-5 w-5 text-emerald-300" />
          YAML editor
        </H2>
        {loading && <SmallText className="text-gray-400">Loading configuration...</SmallText>}
        {error && <SmallText className="text-red-400">Unable to load config: {error.message}</SmallText>}
        {data && (
          <>
            <div className="flex flex-wrap gap-4 text-xs text-gray-400">
              <span>Path: <code className="font-mono text-white">{data.path}</code></span>
              <span>Updated: {formatTime(data.modified_at)}</span>
              <span>Size: {formatBytes(data.size_bytes)}</span>
            </div>
            <textarea
              className="h-72 w-full rounded-xl border border-white/10 bg-gray-950/70 p-3 font-mono text-sm text-white shadow-inner focus:border-violet-400 focus:outline-none"
              value={value}
              onChange={handleChange}
              spellCheck={false}
              disabled={loading || saving}
            />
            {status && (
              <SmallText className={status.tone === 'success' ? 'text-emerald-300' : 'text-red-400'}>
                {status.message}
              </SmallText>
            )}
            <div className="flex flex-wrap gap-3">
              <Button tone="violet" disabled={!dirty || saving} onClick={handleSave}>
                {saving ? 'Saving…' : 'Save changes'}
              </Button>
              <Button variant="outline" disabled={!dirty || saving} onClick={handleReset}>
                Discard
              </Button>
            </div>
            <SmallText className="text-gray-400">
              Saving runs full validation (same as `niac validate`) before persisting so runtime changes stay safe.
            </SmallText>
            <WalkFileBrowser files={walkFiles ?? []} onCopy={handleWalkCopy} />
          </>
        )}
      </CardContent>
    </Card>
  );
}

function WalkFileBrowser({ files, onCopy }: { files: FileEntry[]; onCopy: (path: string) => void }) {
  if (!files.length) {
    return null;
  }
  return (
    <div className="space-y-2">
      <SmallText className="text-gray-400">Available SNMP walks</SmallText>
      <div className="max-h-48 space-y-1 overflow-y-auto rounded-xl border border-white/10 bg-gray-950/50 p-2 text-sm text-gray-300">
        {files.map((file) => (
          <div key={file.path} className="flex items-center justify-between gap-2 rounded-lg border border-white/5 bg-gray-900/50 px-3 py-2">
            <div>
              <p className="text-white">{file.name}</p>
              <SmallText className="text-gray-500">{file.path}</SmallText>
            </div>
            <Button size="sm" variant="outline" onClick={() => onCopy(file.path)}>
              Copy path
            </Button>
          </div>
        ))}
      </div>
    </div>
  );
}

function TopologyPage() {
  const { data: neighbors, loading, error } = useApiResource(fetchNeighbors, [], { intervalMs: POLL_INTERVALS.MEDIUM });
  const { data: topology } = useApiResource(fetchTopology, [], { intervalMs: POLL_INTERVALS.SLOW });
  const [showGraph, setShowGraph] = useState(true);

  return (
    <div className="space-y-6">
      <Card className="border-white/5 bg-gray-900/70">
        <CardContent className="space-y-4">
          <H2 className="mb-0 flex items-center gap-2">
            <Network className="h-5 w-5 text-cyan-300" />
            Network topology
          </H2>
          <P className="text-gray-300">
            Visualize device connections derived from port-channels, trunk ports, and discovered neighbors.
          </P>
          <div className="flex flex-wrap gap-3">
            <Button
              tone={showGraph ? 'violet' : undefined}
              variant={showGraph ? undefined : 'outline'}
              leftIcon={<Network className="h-4 w-4" />}
              onClick={() => setShowGraph(!showGraph)}
            >
              {showGraph ? 'Hide topology' : 'Show topology'}
            </Button>
            <Button variant="outline" leftIcon={<LineChart className="h-4 w-4" />}>Export Graphviz</Button>
          </div>
          {showGraph && topology && <TopologyVisualization topology={topology} />}
        </CardContent>
      </Card>
      <Card className="border-white/5 bg-gray-900/70">
        <CardContent className="space-y-4">
          <H2 className="mb-0">Discovered neighbors</H2>
          {loading && <SmallText className="text-gray-400">Discovering peers...</SmallText>}
          {error && <SmallText className="text-red-400">Unable to load neighbors: {error.message}</SmallText>}
          {!loading && !error && <NeighborTable neighbors={neighbors ?? []} />}
        </CardContent>
      </Card>
    </div>
  );
}

function TopologyVisualization({ topology }: { topology: TopologyGraph }) {
  if (!topology.nodes.length) {
    return (
      <div className="rounded-xl border border-white/10 bg-gray-950/50 p-8 text-center">
        <SmallText className="text-gray-400">No topology data available. Configure trunk ports or port-channels to see connections.</SmallText>
      </div>
    );
  }

  const getNodeColor = (type: string) => {
    const colors: Record<string, string> = {
      router: 'bg-blue-500',
      switch: 'bg-green-500',
      'access-point': 'bg-purple-500',
      server: 'bg-orange-500',
      workstation: 'bg-gray-500',
      firewall: 'bg-red-500',
    };
    return colors[type] || 'bg-cyan-500';
  };

  return (
    <div className="rounded-xl border border-white/10 bg-gray-950/50 p-6">
      <div className="mb-4 flex flex-wrap gap-2">
        {topology.nodes.map((node) => (
          <div key={node.name} className="rounded-lg border border-white/10 bg-gray-900/70 px-4 py-2">
            <div className="flex items-center gap-2">
              <div className={`h-3 w-3 rounded-full ${getNodeColor(node.type)}`} />
              <span className="font-semibold text-white">{node.name}</span>
              <Tag colorScheme="gray" className="text-xs">{node.type}</Tag>
            </div>
          </div>
        ))}
      </div>
      {topology.links.length > 0 && (
        <div className="space-y-2">
          <SmallText className="font-semibold uppercase tracking-wide text-gray-400">Connections</SmallText>
          <div className="space-y-1">
            {topology.links.map((link, idx) => (
              <div key={idx} className="flex items-center gap-2 rounded-lg border border-white/5 bg-gray-900/50 px-3 py-2 text-sm">
                <span className="font-mono text-blue-300">{link.source}</span>
                <span className="text-gray-500">↔</span>
                <span className="font-mono text-blue-300">{link.target}</span>
                {link.label && <Tag colorScheme="purple" className="ml-auto text-xs">{link.label}</Tag>}
              </div>
            ))}
          </div>
        </div>
      )}
      {topology.links.length === 0 && (
        <SmallText className="text-gray-400">No connections configured. Add trunk ports or port-channels to see links.</SmallText>
      )}
    </div>
  );
}

// FEATURE #125: Memoize NeighborTable to prevent unnecessary re-renders
const NeighborTable = memo(({ neighbors }: { neighbors: NeighborRecord[] }) => {
  return (
    <div className="overflow-x-auto rounded-xl border border-white/5">
      <table className="min-w-full divide-y divide-white/10 text-sm">
        <thead className="bg-gray-900/60 text-xs uppercase tracking-wide text-gray-400">
          <tr>
            <th className="px-4 py-3 text-left">Local</th>
            <th className="px-4 py-3 text-left">Remote</th>
            <th className="px-4 py-3 text-left">Protocol</th>
            <th className="px-4 py-3 text-left">Mgmt address</th>
            <th className="px-4 py-3 text-left">Last seen</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-white/5 text-gray-300">
          {neighbors.map((neighbor) => (
            <tr key={`${neighbor.LocalDevice}-${neighbor.RemoteDevice}-${neighbor.RemotePort}`}>
              <td className="px-4 py-3 font-semibold text-white">
                {neighbor.LocalDevice}
                {neighbor.RemotePort && <span className="text-gray-400"> · {neighbor.RemotePort}</span>}
              </td>
              <td className="px-4 py-3 text-white/90">{neighbor.RemoteDevice}</td>
              <td className="px-4 py-3">
                <Tag colorScheme="purple">{neighbor.Protocol}</Tag>
              </td>
              <td className="px-4 py-3 font-mono text-xs">{neighbor.ManagementAddress || '—'}</td>
              <td className="px-4 py-3 text-gray-400">{formatRelativeTime(neighbor.LastSeen)}</td>
            </tr>
          ))}
          {!neighbors.length && (
            <tr>
              <td colSpan={5} className="px-4 py-4 text-center text-gray-400">
                No neighbors reported yet. Enable LLDP/CDP/EDP/FDP in your config to populate this table.
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
});

function AnalysisPage() {
  const { data: history } = useApiResource(fetchHistory, [], { intervalMs: POLL_INTERVALS.SLOW });

  return (
    <div className="space-y-6">
      <Card className="border-white/5 bg-gray-900/70">
        <CardContent className="space-y-4">
          <H2 className="mb-0 flex items-center gap-2">
            <LineChart className="h-5 w-5 text-emerald-300" />
            Capture & walk analysis
          </H2>
          <P>
            Replay PCAP files, filter packets, and bundle the results with SNMP walk exports. Every run can be
            published as downloadable evidence for troubleshooting or demo handoffs.
          </P>
          <div className="space-y-3 text-sm text-gray-300">
            {(history ?? []).slice(0, 5).map((item) => (
              <div key={item.id} className="rounded-lg border border-white/5 bg-gray-950/50 p-3">
                <p className="text-white font-semibold">{item.config_name}</p>
                <SmallText className="text-gray-400">
                  {formatTime(item.started_at)} · duration {formatDuration(item.duration)} · RX {formatNumber(item.packets_received)} · TX {formatNumber(item.packets_sent)}
                </SmallText>
              </div>
            ))}
            {!history?.length && <SmallText className="text-gray-400">No captured runs yet.</SmallText>}
          </div>
          <div className="flex flex-wrap gap-3">
            <Button tone="violet" leftIcon={<LineChart className="h-4 w-4" />}>Open analyzer</Button>
            <Button variant="outline" leftIcon={<FileCog className="h-4 w-4" />}>Export bundle</Button>
          </div>
        </CardContent>
      </Card>
      <ReplayPanel />
    </div>
  );
}

function ReplayPanel() {
  const { data: status, loading, error } = useApiResource(fetchReplayStatus, [], { intervalMs: 8000 });
  const { data: pcaps } = useApiResource(() => fetchFiles('pcaps'), [], { intervalMs: 45000 });
  const [pcapPath, setPcapPath] = useState('');
  const [loopMs, setLoopMs] = useState('');
  const [scale, setScale] = useState('');
  const [uploadFile, setUploadFile] = useState<File | null>(null);
  const [fileInputKey, setFileInputKey] = useState(0);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState<{ tone: 'success' | 'error'; text: string } | null>(null);

  const handleStart = async () => {
    if (busy) {
      return;
    }
    const effectiveName = pcapPath.trim() || uploadFile?.name || '';
    if (!effectiveName) {
      setMessage({ tone: 'error', text: 'Provide a PCAP path or upload a capture' });
      return;
    }
    setBusy(true);
    setMessage(null);
    try {
      const payload: ReplayRequest = { file: effectiveName };
      if (loopMs.trim()) {
        payload.loop_ms = Number(loopMs);
      }
      if (scale.trim()) {
        payload.scale = Number(scale);
      }
      if (uploadFile) {
        payload.data = await fileToBase64(uploadFile);
      }
      await startReplay(payload);
      setMessage({ tone: 'success', text: 'Replay started' });
      if (uploadFile) {
        setUploadFile(null);
        setFileInputKey((value) => value + 1);
      }
    } catch (err) {
      setMessage({ tone: 'error', text: (err as Error).message });
    } finally {
      setBusy(false);
    }
  };

  const handleStop = async () => {
    if (busy) return;

    // Confirm before stopping replay
    if (!window.confirm('Are you sure you want to stop the replay?')) {
      return;
    }

    setBusy(true);
    setMessage(null);
    try {
      await stopReplay();
      setMessage({ tone: 'success', text: 'Replay stopped' });
    } catch (err) {
      setMessage({ tone: 'error', text: getErrorMessage(err) });
    } finally {
      setBusy(false);
    }
  };

  return (
    <Card className="border-white/5 bg-gray-900/70">
      <CardContent className="space-y-4">
        <H2 className="mb-0 flex items-center gap-2">
          <PlugZap className="h-5 w-5 text-pink-300" />
          Packet replay
        </H2>
        <P className="text-gray-300">
          Point NIAC at a PCAP file to replay capture traffic through the live interface. Replay honors loop timing and
          scaling so you can rapidly reproduce demos without leaving the Web UI.
        </P>
        {loading && <SmallText className="text-gray-400">Checking replay engine…</SmallText>}
        {error && <SmallText className="text-red-400">Unable to read replay status: {error.message}</SmallText>}
        <div className="grid gap-4 md:grid-cols-3">
          <div>
            <label htmlFor="pcap-file-path" className="block text-sm text-gray-400">
              PCAP file
            </label>
            <input
              id="pcap-file-path"
              type="text"
              className="mt-1 w-full rounded-lg border border-white/10 bg-gray-950/60 p-2 font-mono text-sm text-white focus:border-violet-400 focus:outline-none"
              placeholder="/path/to/capture.pcap"
              value={pcapPath}
              onChange={(event) => setPcapPath(event.target.value)}
              aria-describedby="pcap-path-help"
            />
          </div>
          <div>
            <label htmlFor="loop-interval" className="block text-sm text-gray-400">
              Loop interval (ms)
            </label>
            <input
              id="loop-interval"
              type="number"
              className="mt-1 w-full rounded-lg border border-white/10 bg-gray-950/60 p-2 text-sm text-white focus:border-violet-400 focus:outline-none"
              placeholder="0"
              value={loopMs}
              onChange={(event) => setLoopMs(event.target.value)}
              aria-describedby="loop-help"
            />
          </div>
          <div>
            <label htmlFor="time-scale" className="block text-sm text-gray-400">
              Time scale
            </label>
            <input
              id="time-scale"
              type="number"
              step="0.1"
              className="mt-1 w-full rounded-lg border border-white/10 bg-gray-950/60 p-2 text-sm text-white focus:border-violet-400 focus:outline-none"
              placeholder="1.0"
              value={scale}
              onChange={(event) => setScale(event.target.value)}
              aria-describedby="scale-help"
            />
          </div>
        </div>
        {message && (
          <SmallText
            className={message.tone === 'success' ? 'text-emerald-300' : 'text-red-400'}
            role="alert"
            aria-live="polite"
          >
            {message.text}
          </SmallText>
        )}
        <div className="grid gap-4 md:grid-cols-2">
          <div>
            <label htmlFor="pcap-file-upload" className="block text-sm text-gray-400">
              Upload PCAP
            </label>
            <input
              id="pcap-file-upload"
              key={fileInputKey}
              type="file"
              accept=".pcap,.pcapng,application/vnd.tcpdump.pcap"
              className="mt-1 w-full cursor-pointer rounded-lg border border-dashed border-white/10 bg-gray-950/40 p-2 text-sm text-white file:mr-3 file:rounded-md file:border-0 file:bg-violet-600 file:px-3 file:py-1 file:text-sm file:font-medium"
              onChange={(event) => {
                const file = event.target.files?.[0];
                if (!file) {
                  setUploadFile(null);
                  return;
                }

                // Validate file size (100MB limit for PCAP files)
                const MAX_SIZE = 100 * 1024 * 1024;
                if (file.size > MAX_SIZE) {
                  setMessage({
                    tone: 'error',
                    text: `PCAP file too large. Maximum size is ${formatBytes(MAX_SIZE)}`
                  });
                  event.target.value = '';
                  return;
                }

                // Validate file type
                if (!file.name.match(/\.(pcap|pcapng)$/i)) {
                  setMessage({
                    tone: 'error',
                    text: 'Please select a PCAP file (.pcap or .pcapng)'
                  });
                  event.target.value = '';
                  return;
                }

                setUploadFile(file);
              }}
              disabled={busy}
            />
            <SmallText className="text-gray-500">
              If the server cannot access your filesystem, upload a capture directly from the browser.
            </SmallText>
          </div>
          {uploadFile && (
            <div className="rounded-lg border border-white/10 bg-gray-950/40 p-3 text-sm text-gray-300">
              <p className="font-semibold text-white">{uploadFile.name}</p>
              <SmallText className="text-gray-400">{formatBytes(uploadFile.size)}</SmallText>
            </div>
          )}
        </div>
        <div className="flex flex-wrap gap-3">
          <Button tone="violet" disabled={(!pcapPath.trim() && !uploadFile) || busy} onClick={handleStart}>
            {busy ? 'Working…' : 'Start replay'}
          </Button>
          <Button variant="outline" disabled={busy || !status?.running} onClick={handleStop}>
            Stop replay
          </Button>
        </div>
        {pcaps && pcaps.length > 0 && (
          <div className="space-y-2">
            <SmallText className="text-gray-400">Discovered captures</SmallText>
            <div className="max-h-48 space-y-1 overflow-y-auto rounded-xl border border-white/10 bg-gray-950/50 p-2 text-sm text-gray-300">
              {pcaps.map((file) => (
                <div key={file.path} className="flex items-center justify-between gap-2 rounded-lg border border-white/5 bg-gray-900/50 px-3 py-2">
                  <div>
                    <p className="text-white">{file.name}</p>
                    <SmallText className="text-gray-500">{file.path}</SmallText>
                  </div>
                  <div className="flex gap-2">
                    <Button size="sm" variant="outline" onClick={() => setPcapPath(file.path)}>
                      Use path
                    </Button>
                    <Button size="sm" variant="ghost" onClick={() => copyToClipboard(file.path)}>
                      Copy
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}
        {status && (
          <div className="rounded-lg border border-white/10 bg-gray-950/50 p-3 text-sm text-gray-300">
            <p className="font-semibold text-white">{status.running ? 'Running' : 'Idle'}</p>
            {status.file && <p className="font-mono text-xs text-gray-400">{status.file}</p>}
            {status.running && status.started_at && (
              <SmallText className="text-gray-400">Started {formatRelativeTime(status.started_at)}</SmallText>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function AutomationPage() {
  const { data: stats } = useApiResource(fetchStats, []);

  return (
    <div className="space-y-6">
      <Card className="border-white/5 bg-gray-900/70">
        <CardContent className="space-y-4">
          <H2 className="mb-0 flex items-center gap-2">
            <Workflow className="h-5 w-5 text-yellow-300" />
            Automation roadmap
          </H2>
          <P>
            Define alert thresholds, webhook routes, and (soon) runnable workflow automations. Use the CLI flags today,
            then manage them graphically here as the 2.0 UI matures. Current alert counter: {stats?.stack.errors ?? 0}.
          </P>
          <ul className="space-y-3 text-sm text-gray-300">
            <li className="rounded-lg border border-white/5 bg-gray-950/50 p-3">Webhooks inherit settings from the `--alert-webhook` flag and can be overridden per run.</li>
            <li className="rounded-lg border border-white/5 bg-gray-950/50 p-3">Packet thresholds mirror CLI/TUI options so headless and web control stay aligned.</li>
            <li className="rounded-lg border border-white/5 bg-gray-950/50 p-3">Future: orchestrate multi-run scenarios and publish signed run reports.</li>
          </ul>
        </CardContent>
      </Card>
      <AlertConfigCard />
    </div>
  );
}

function AlertConfigCard() {
  const { data, loading, error } = useApiResource(fetchAlerts, [], { intervalMs: 15000 });
  const [threshold, setThreshold] = useState('');
  const [webhook, setWebhook] = useState('');
  const [dirty, setDirty] = useState(false);
  const [saving, setSaving] = useState(false);
  const [status, setStatus] = useState<{ tone: 'success' | 'error'; text: string } | null>(null);

  useEffect(() => {
    if (data && !dirty) {
      setThreshold(data.packets_threshold ? String(data.packets_threshold) : '');
      setWebhook(data.webhook_url ?? '');
    }
  }, [data, dirty]);

  const commit = async () => {
    if (!dirty || saving) return;
    setSaving(true);
    setStatus(null);
    try {
      const payload: AlertConfig = {
        packets_threshold: threshold ? Number(threshold) : 0,
        webhook_url: webhook.trim(),
      };
      await updateAlerts(payload);
      setDirty(false);
      setStatus({ tone: 'success', text: 'Alert configuration saved' });
    } catch (err) {
      setStatus({ tone: 'error', text: (err as Error).message });
    } finally {
      setSaving(false);
    }
  };

  const reset = () => {
    if (!data) return;
    setThreshold(data.packets_threshold ? String(data.packets_threshold) : '');
    setWebhook(data.webhook_url ?? '');
    setDirty(false);
    setStatus(null);
  };

  return (
    <Card className="border-white/5 bg-gray-900/70">
      <CardContent className="space-y-4">
        <H2 className="mb-0 flex items-center gap-2">
          <BellRing className="h-5 w-5 text-orange-300" />
          Alert policy
        </H2>
        <P className="text-gray-300">
          Updates take effect immediately—no CLI restart required. Leave the threshold blank or zero to disable packet
          alerts entirely.
        </P>
        {loading && <SmallText className="text-gray-400">Loading alert configuration…</SmallText>}
        {error && <SmallText className="text-red-400">Unable to load alerts: {error.message}</SmallText>}
        {data && (
          <>
            <div className="grid gap-4 md:grid-cols-2">
              <div>
                <SmallText className="text-gray-400">Packet threshold</SmallText>
                <input
                  className="mt-1 w-full rounded-lg border border-white/10 bg-gray-950/60 p-2 text-sm text-white focus:border-violet-400 focus:outline-none"
                  type="number"
                  min="0"
                  placeholder="100000"
                  value={threshold}
                  onChange={(event) => {
                    setThreshold(event.target.value);
                    setDirty(true);
                    setStatus(null);
                  }}
                />
              </div>
              <div>
                <SmallText className="text-gray-400">Webhook URL</SmallText>
                <input
                  className="mt-1 w-full rounded-lg border border-white/10 bg-gray-950/60 p-2 text-sm text-white focus:border-violet-400 focus:outline-none"
                  placeholder="https://hooks.example.com/niac"
                  value={webhook}
                  onChange={(event) => {
                    setWebhook(event.target.value);
                    setDirty(true);
                    setStatus(null);
                  }}
                />
              </div>
            </div>
            {status && (
              <SmallText className={status.tone === 'success' ? 'text-emerald-300' : 'text-red-400'}>
                {status.text}
              </SmallText>
            )}
            <div className="flex flex-wrap gap-3">
              <Button tone="violet" disabled={!dirty || saving} onClick={commit}>
                {saving ? 'Saving…' : 'Save alerts'}
              </Button>
              <Button variant="outline" disabled={!dirty || saving} onClick={reset}>
                Reset
              </Button>
            </div>
          </>
        )}
      </CardContent>
    </Card>
  );
}

// FEATURE #125: Memoize StatBlock to prevent unnecessary re-renders
const StatBlock = memo(({ label, value, helper }: { label: string; value: string; helper: string }) => {
  return (
    <div>
      <SmallText className="uppercase tracking-wide text-gray-400">{label}</SmallText>
      <p className="text-3xl font-bold text-white">{value}</p>
      <SmallText className="text-gray-300">{helper}</SmallText>
    </div>
  );
});

function formatNumber(value: number) {
  return value.toLocaleString();
}

function formatTime(value: string) {
  return new Date(value).toLocaleString();
}

function formatDuration(value: string) {
  return value || '—';
}

function formatRelativeTime(timestamp: string) {
  if (!timestamp) return '—';
  const diff = Date.now() - new Date(timestamp).getTime();
  if (diff < 0) return 'just now';
  const seconds = Math.floor(diff / 1000);
  if (seconds < 60) return `${seconds}s ago`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  return `${days}d ago`;
}

// Helper function to safely extract error messages
export function getErrorMessage(err: unknown): string {
  if (err instanceof Error) return err.message;
  if (typeof err === 'string') return err;
  return 'An unexpected error occurred';
}

function formatBytes(size: number) {
  if (!Number.isFinite(size) || size <= 0) {
    return '0 B';
  }
  const units = ['B', 'KB', 'MB', 'GB'];
  let idx = 0;
  let value = size;
  while (value >= 1024 && idx < units.length - 1) {
    value /= 1024;
    idx++;
  }
  return `${value.toFixed(value >= 10 ? 0 : 1)} ${units[idx]}`;
}

async function copyToClipboard(value: string) {
  if (navigator?.clipboard?.writeText) {
    await navigator.clipboard.writeText(value);
    return;
  }
  const textarea = document.createElement('textarea');
  textarea.value = value;
  textarea.style.position = 'fixed';
  textarea.style.opacity = '0';
  document.body.appendChild(textarea);
  textarea.select();
  document.execCommand('copy');
  document.body.removeChild(textarea);
}

// Optimized file to base64 conversion using FileReader
async function fileToBase64(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => {
      const result = reader.result as string;
      // Remove data URL prefix (data:*/*;base64,)
      const base64 = result.split(',')[1] || result;
      resolve(base64);
    };
    reader.onerror = () => reject(new Error('Failed to read file'));
    reader.readAsDataURL(file);
  });
}

async function fileToText(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(reader.result as string);
    reader.onerror = reject;
    reader.readAsText(file);
  });
}

function formatUptime(seconds: number): string {
  if (seconds < 60) return `${Math.floor(seconds)}s`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ${Math.floor(seconds % 60)}s`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ${minutes % 60}m`;
  const days = Math.floor(hours / 24);
  return `${days}d ${hours % 24}h`;
}
