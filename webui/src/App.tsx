import { memo, useCallback, type ChangeEvent, type FC, type ReactNode, useEffect, useState } from 'react';
import { Routes, Route, Navigate } from 'react-router-dom';
import type { LucideIcon } from 'lucide-react';
import {
  Activity,
  Server,
  Network,
  LineChart,
  Workflow,
  PlugZap,
  Bot,
  BellRing,
  SatelliteDish,
  FileCog,
  Zap,
  Terminal,
  FileSearch,
  FileCode,
  GitCompare,
  FileBox,
  Wrench,
} from 'lucide-react';
import {
  SidebarLayout,
  PageHeader,
  Card,
  CardContent,
  Button,
  Tag,
  H2,
  P,
  SmallText,
  AccentLink,
} from './ui';
import type { SidebarNavGroup } from './ui';
import { useApiResource } from './hooks/useApiResource';
import { useVirtualScroll } from './hooks/useVirtualScroll';
import {
  fetchStats,
  fetchDevices,
  fetchHistory,
  fetchConfig,
  updateConfig,
  fetchReplayStatus,
  startReplay,
  stopReplay,
  fetchAlerts,
  updateAlerts,
  fetchFiles,
  fetchVersion,
  fetchErrorTypes,
  fetchInterfaces,
  fetchSimulationStatus,
  startSimulation,
  stopSimulation,
} from './api/client';
import type { DeviceSummary, HistoryRecord, AlertConfig, ReplayRequest, FileEntry, ErrorType, NetworkInterface } from './api/types';
import { TrafficInjectionPage } from './pages/TrafficInjectionPage';
import { DebugConsolePage } from './pages/DebugConsolePage';
import { PacketInspectorPage } from './pages/PacketInspectorPage';
import { TemplatesPage } from './pages/TemplatesPage';
import { ConfigDiffPage } from './pages/ConfigDiffPage';
import { PcapAnalyzerPage } from './pages/PcapAnalyzerPage';
import { DeviceListPage } from './pages/DeviceListPage';
import { DeviceEditorPage } from './pages/DeviceEditorPage';
import { TopologyPage } from './pages/TopologyPage';
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
    path: '/device-config',
    label: 'Config Builder',
    title: 'Visual Config Builder',
    description: 'Create, edit, and manage device configurations with a visual interface.',
    icon: Wrench,
    Component: DeviceListPage,
    badge: 'New',
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
  {
    path: '/debug',
    label: 'Debug Console',
    title: 'Debug Console',
    description: 'Real-time log streaming and debugging tools for monitoring NIAC operations.',
    icon: Terminal,
    Component: DebugConsolePage,
  },
  {
    path: '/packets',
    label: 'Packet Inspector',
    title: 'Packet Inspector',
    description: 'Real-time packet hex dump viewing with protocol filtering and search capabilities.',
    icon: FileSearch,
    Component: PacketInspectorPage,
  },
  {
    path: '/templates',
    label: 'Templates',
    title: 'Configuration Templates',
    description: 'Browse and use pre-configured network templates to quickly start simulations.',
    icon: FileCode,
    Component: TemplatesPage,
  },
  {
    path: '/config-diff',
    label: 'Config Diff',
    title: 'Config Diff & Merge',
    description: 'Compare and merge YAML configuration files with visual diff and merge controls.',
    icon: GitCompare,
    Component: ConfigDiffPage,
  },
  {
    path: '/pcap-analyzer',
    label: 'PCAP Analyzer',
    title: 'PCAP Analyzer',
    description: 'Upload and analyze PCAP files with packet inspection, filtering, and statistics.',
    icon: FileBox,
    Component: PcapAnalyzerPage,
  },
];

// Organize navigation into logical groups
const navGroups: SidebarNavGroup[] = [
  {
    label: 'Control',
    items: [
      { path: '/', label: 'Command Center', icon: Activity },
      { path: '/runtime', label: 'Runtime Control', icon: PlugZap },
    ],
  },
  {
    label: 'Configuration',
    items: [
      { path: '/devices', label: 'Devices & Config', icon: Server },
      { path: '/device-config', label: 'Config Builder', icon: Wrench, badge: 'New' },
      { path: '/templates', label: 'Templates', icon: FileCode },
      { path: '/config-diff', label: 'Config Diff', icon: GitCompare },
    ],
  },
  {
    label: 'Network',
    items: [
      { path: '/topology', label: 'Topology', icon: Network },
      { path: '/traffic', label: 'Traffic Injection', icon: Zap },
    ],
  },
  {
    label: 'Analysis',
    items: [
      { path: '/analysis', label: 'Playback', icon: LineChart },
      { path: '/debug', label: 'Debug Console', icon: Terminal },
      { path: '/packets', label: 'Packet Inspector', icon: FileSearch },
      { path: '/pcap-analyzer', label: 'PCAP Analyzer', icon: FileBox },
    ],
  },
  {
    label: 'Automation',
    items: [
      { path: '/automation', label: 'Alerts & Workflows', icon: Workflow, badge: 'Beta' },
    ],
  },
];

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
    <SidebarLayout groups={navGroups} version={version?.version}>
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
        {/* Dynamic routes for device editor */}
        <Route
          path="/device-config/new"
          element={
            <PageTemplate
              page={{
                path: '/device-config/new',
                label: 'New Device',
                title: 'New Device',
                description: 'Create a new network device configuration.',
                icon: Wrench,
                Component: DeviceEditorPage,
              }}
            >
              <DeviceEditorPage />
            </PageTemplate>
          }
        />
        <Route
          path="/device-config/:hostname"
          element={
            <PageTemplate
              page={{
                path: '/device-config/:hostname',
                label: 'Edit Device',
                title: 'Edit Device',
                description: 'Edit device configuration settings.',
                icon: Wrench,
                Component: DeviceEditorPage,
              }}
            >
              <DeviceEditorPage />
            </PageTemplate>
          }
        />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </SidebarLayout>
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
  const { data: stats } = useApiResource(fetchStats, [], {
    intervalMs: POLL_INTERVALS.MEDIUM,
  });
  const { data: history } = useApiResource(fetchHistory, [], { intervalMs: POLL_INTERVALS.SLOW });
  const { data: errorInfo } = useApiResource(fetchErrorTypes, []);
  const { data: simStatus } = useApiResource(fetchSimulationStatus, [], { intervalMs: POLL_INTERVALS.FAST });
  const [showErrors, setShowErrors] = useState(false);

  const isRunning = simStatus?.running ?? false;
  // Get uptime from simulation status
  const uptimeSeconds = simStatus?.uptime_seconds ?? 0;

  return (
    <div className="space-y-6 animate-fade-in">
      {/* Status banner */}
      <Card className={`border-l-4 ${isRunning ? 'border-l-emerald-500' : 'border-l-gray-500'}`}>
        <CardContent className="py-4">
          <div className="flex flex-wrap items-center justify-between gap-4">
            <div className="flex items-center gap-4">
              <div className="relative">
                <div className={`h-3 w-3 rounded-full ${isRunning ? 'bg-emerald-500' : 'bg-gray-500'}`} />
                {isRunning && (
                  <div className="absolute inset-0 h-3 w-3 rounded-full bg-emerald-500 animate-ping opacity-75" />
                )}
              </div>
              <div>
                <p className="font-semibold text-white">
                  {isRunning ? 'Simulation Running' : 'Simulation Stopped'}
                </p>
                <p className="text-sm text-gray-400">
                  {stats?.interface ?? 'No interface'} • {stats?.device_count ?? 0} devices
                </p>
              </div>
            </div>
            <div className="flex items-center gap-2">
              {uptimeSeconds > 0 && (
                <Tag colorScheme="violet">
                  Uptime: {formatUptime(uptimeSeconds)}
                </Tag>
              )}
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Stat cards row */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <Card hover className="group">
          <CardContent className="space-y-1">
            <div className="flex items-center justify-between">
              <span className="text-sm font-medium text-gray-400">Devices Online</span>
              <Server className="h-5 w-5 text-violet-400 group-hover:scale-110 transition-transform" />
            </div>
            <p className="text-3xl font-bold text-white">{stats?.device_count ?? '—'}</p>
            <p className="text-xs text-gray-500">Active network devices</p>
          </CardContent>
        </Card>

        <Card hover className="group">
          <CardContent className="space-y-1">
            <div className="flex items-center justify-between">
              <span className="text-sm font-medium text-gray-400">Packets RX</span>
              <Activity className="h-5 w-5 text-emerald-400 group-hover:scale-110 transition-transform" />
            </div>
            <p className="text-3xl font-bold text-white">
              {stats ? formatNumber(stats.stack.packets_received) : '—'}
            </p>
            <p className="text-xs text-gray-500">
              {stats ? `${formatNumber(stats.stack.packets_sent)} sent` : 'Awaiting data'}
            </p>
          </CardContent>
        </Card>

        <Card hover className="group">
          <CardContent className="space-y-1">
            <div className="flex items-center justify-between">
              <span className="text-sm font-medium text-gray-400">DNS Queries</span>
              <SatelliteDish className="h-5 w-5 text-amber-400 group-hover:scale-110 transition-transform" />
            </div>
            <p className="text-3xl font-bold text-white">
              {stats ? formatNumber(stats.stack.dns_queries) : '—'}
            </p>
            <p className="text-xs text-gray-500">
              DHCP: {stats ? formatNumber(stats.stack.dhcp_requests) : '—'}
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

            {showErrors && errorInfo && <ErrorInjectionPanel errorTypes={errorInfo.available_types} info={errorInfo.info} />}
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
                <div key={item.id} className="rounded-lg border border-white/5 bg-gray-950/50 p-3 hover:border-white/10 transition-colors">
                  <div className="flex items-center justify-between mb-1">
                    <p className="font-mono text-xs text-violet-300">{formatTime(item.started_at)}</p>
                    <Tag colorScheme="gray" className="text-[10px]">{item.device_count} dev</Tag>
                  </div>
                  <p className="text-white font-medium text-sm truncate">{item.config_name}</p>
                  <div className="flex gap-3 mt-1 text-xs text-gray-500">
                    <span>RX {formatNumber(item.packets_received)}</span>
                    <span>TX {formatNumber(item.packets_sent)}</span>
                  </div>
                </div>
              ))}
              {!history?.length && (
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

// TopologyPage is now imported from ./pages/TopologyPage

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
