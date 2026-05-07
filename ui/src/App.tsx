import type { LucideIcon } from 'lucide-react';
import {
  Activity,
  FileBox,
  FileCode,
  FileSearch,
  GitCompare,
  LineChart,
  Network,
  PlugZap,
  Radar,
  Server,
  ShieldCheck,
  Terminal,
  Workflow,
  Wrench,
  Zap,
} from 'lucide-react';
import { type FC, lazy, memo, type ReactNode, Suspense } from 'react';
import { Navigate, Route, Routes, useLocation } from 'react-router-dom';
import { ErrorBoundary, PageErrorBoundary } from './components/ErrorBoundary';
import { AppProvider, useAppState } from './contexts/AppContext';
import { useKeyboardShortcuts } from './hooks/useKeyboardShortcuts';
import { Breadcrumbs } from './ui/Breadcrumbs';
import { PageHeader } from './ui/Layout';
import type { SidebarNavGroup } from './ui/Sidebar';
import { SidebarLayout } from './ui/Sidebar';
import { ToastContainer } from './ui/ToastContainer';
import './App.css';

// Eagerly loaded pages (small, frequently used)
import { DashboardPage } from './pages/DashboardPage';
import { DevicesPage } from './pages/DevicesPage';
import { RuntimeControlPage } from './pages/RuntimeControlPage';

// Lazy loaded pages (large dependencies or infrequently used)
const TopologyPage = lazy(() =>
  import('./pages/TopologyPage').then((m) => ({ default: m.TopologyPage })),
);
const ConfigDiffPage = lazy(() =>
  import('./pages/ConfigDiffPage').then((m) => ({ default: m.ConfigDiffPage })),
);
const DeviceEditorPage = lazy(() =>
  import('./pages/DeviceEditorPage').then((m) => ({ default: m.DeviceEditorPage })),
);
const DeviceListPage = lazy(() =>
  import('./pages/DeviceListPage').then((m) => ({ default: m.DeviceListPage })),
);
const TemplatesPage = lazy(() =>
  import('./pages/TemplatesPage').then((m) => ({ default: m.TemplatesPage })),
);
const AnalysisPage = lazy(() =>
  import('./pages/AnalysisPage').then((m) => ({ default: m.AnalysisPage })),
);
const AutomationPage = lazy(() =>
  import('./pages/AutomationPage').then((m) => ({ default: m.AutomationPage })),
);
const DebugConsolePage = lazy(() =>
  import('./pages/DebugConsolePage').then((m) => ({ default: m.DebugConsolePage })),
);
const PacketInspectorPage = lazy(() =>
  import('./pages/PacketInspectorPage').then((m) => ({ default: m.PacketInspectorPage })),
);
const PcapAnalyzerPage = lazy(() =>
  import('./pages/PcapAnalyzerPage').then((m) => ({ default: m.PcapAnalyzerPage })),
);
const TrafficInjectionPage = lazy(() =>
  import('./pages/TrafficInjectionPage').then((m) => ({ default: m.TrafficInjectionPage })),
);
const NeighborsPage = lazy(() =>
  import('./pages/NeighborsPage').then((m) => ({ default: m.NeighborsPage })),
);
const WalkValidatorPage = lazy(() =>
  import('./pages/WalkValidatorPage').then((m) => ({ default: m.WalkValidatorPage })),
);

// Loading fallback for lazy-loaded pages
const PageLoader = () => (
  <div className="flex items-center justify-center min-h-[400px]">
    <div className="flex flex-col items-center gap-3">
      <div className="h-8 w-8 animate-spin rounded-full border-4 border-violet-500 border-t-transparent" />
      <p className="text-sm text-gray-400">Loading...</p>
    </div>
  </div>
);

type PageConfig = {
  path: string;
  label: string;
  title: string;
  description: string;
  icon: LucideIcon;
  component: FC;
  badge?: string;
  help?: ReactNode;
};

const pages: PageConfig[] = [
  {
    path: '/',
    label: 'Dashboard',
    title: 'Dashboard',
    description: 'Live counters, run snapshots, and automation status for the active NIAC stack.',
    icon: Activity,
    component: DashboardPage,
  },
  {
    path: '/runtime',
    label: 'Simulation',
    title: 'Simulation Control',
    description: 'Monitor runtime status, view network interfaces, and manage NIAC configuration.',
    icon: PlugZap,
    component: RuntimeControlPage,
    help: (
      <>
        <p>
          Start, stop, and inspect the simulation. Pick a network interface, point to a YAML config,
          and the daemon spawns the requested device personas on that interface.
        </p>
        <h4>Tips</h4>
        <ul>
          <li>
            <strong>Download YAML</strong> exports the running config — equivalent to the legacy
            CLI's runtime config dump. Useful for capturing the merged result of imports / template
            usage and committing it to source control.
          </li>
          <li>
            <code>capture_playbacks:</code> in the YAML auto-starts a PCAP replay alongside the sim.
            The Replay page (Analysis → Replay) lets you switch the playback file at runtime.
          </li>
          <li>
            Use <code>lo0</code> (macOS) / <code>lo</code> (Linux) for safe local testing.
          </li>
        </ul>
      </>
    ),
  },
  {
    path: '/devices',
    label: 'Live Devices',
    title: 'Live Devices (running config)',
    description:
      'Read-only view of devices in the current running configuration plus a YAML preview.',
    icon: Server,
    component: DevicesPage,
    help: (
      <>
        <p>
          What's currently running on the daemon, derived from the YAML the daemon was started with.
          Read-only.
        </p>
        <h4>For editing</h4>
        <p>
          To modify your saved device library, go to <strong>Device Library</strong>. To swap the
          running config wholesale, restart the simulation from <strong>Simulation</strong>.
        </p>
      </>
    ),
  },
  {
    path: '/device-config',
    label: 'Device Library',
    title: 'Device Library',
    description:
      'Manage saved device configurations: search, filter, edit, clone, and delete. Click a device to open the visual editor.',
    icon: Wrench,
    component: DeviceListPage,
    help: (
      <>
        <p>
          Your library of saved device configurations. Edits here update the YAML stored on disk but
          don't push to the running simulation — start a new simulation to load the changes.
        </p>
        <h4>Common actions</h4>
        <ul>
          <li>
            <strong>Click a device</strong> — open the visual editor.
          </li>
          <li>
            <strong>Clone</strong> — duplicate a device with a new hostname.
          </li>
          <li>
            <strong>Bulk select</strong> — checkbox in each row, then bulk-delete from the toolbar.
          </li>
        </ul>
      </>
    ),
  },
  {
    path: '/topology',
    label: 'Topology & Neighbors',
    title: 'Topology & Neighbor Insight',
    description: 'LLDP/CDP/EDP/FDP visibility for verifying intent before exporting to Graphviz.',
    icon: Network,
    component: TopologyPage,
  },
  {
    path: '/analysis',
    label: 'Replay',
    title: 'Replay & Playback',
    description: 'Replay PCAPs, inspect SNMP walks, and publish bundles directly from the UI.',
    icon: LineChart,
    component: AnalysisPage,
  },
  {
    path: '/automation',
    label: 'Alerts',
    title: 'Alerts',
    description: 'Configure alert thresholds and webhook targets for the running daemon.',
    icon: Workflow,
    component: AutomationPage,
    help: (
      <>
        <p>
          The daemon emits an alert webhook when the total packet count crosses the configured
          threshold. Useful for catching runaway traffic from a misconfigured device persona.
        </p>
        <h4>Webhook security</h4>
        <p>
          Outbound webhook destinations are gated by an SSRF defence: raw private / loopback /
          link-local IPs are rejected, and if the daemon was started with{' '}
          <code>--webhook-allowed-host</code>, only those hostnames are allowed. Set the allowlist
          in production rather than relying on the implicit IP filter — see{' '}
          <a
            className="text-violet-300 underline"
            href="https://github.com/krisarmstrong/niac-go/blob/main/SECURITY.md"
          >
            SECURITY.md
          </a>
          .
        </p>
        <h4>Disabling alerts</h4>
        <p>Leave the threshold blank or set it to 0 to disable packet alerts entirely.</p>
      </>
    ),
  },
  {
    path: '/traffic',
    label: 'Traffic Injection',
    title: 'Traffic & Error Injection',
    description: 'Inject network errors and replay PCAP traffic for testing and simulation.',
    icon: Zap,
    component: TrafficInjectionPage,
  },
  {
    path: '/debug',
    label: 'Protocol Debug',
    title: 'Protocol Debug',
    description: 'Real-time log streaming and debugging tools for monitoring NIAC operations.',
    icon: Terminal,
    component: DebugConsolePage,
  },
  {
    path: '/packets',
    label: 'Live Packets',
    title: 'Live Packets',
    description:
      'Real-time packet hex dump viewing with protocol filtering and search capabilities.',
    icon: FileSearch,
    component: PacketInspectorPage,
  },
  {
    path: '/templates',
    label: 'Templates',
    title: 'Configuration Templates',
    description: 'Browse and use pre-configured network templates to quickly start simulations.',
    icon: FileCode,
    component: TemplatesPage,
    help: (
      <>
        <p>
          Built-in templates are starter YAMLs for common scenarios (single router, layer-2 lab,
          etc.). Pick one, give it a name, and it's saved to your Device Library.
        </p>
        <h4>Importing legacy configs</h4>
        <p>
          The <strong>Import legacy config</strong> card at the bottom converts a Java-DSL{' '}
          <code>.cfg</code> file to YAML — the same operation as <code>niac config export</code> on
          the CLI. Useful for moving off the v1 format.
        </p>
      </>
    ),
  },
  {
    path: '/config-diff',
    label: 'Config Diff',
    title: 'Config Diff & Merge',
    description: 'Compare and merge YAML configuration files with visual diff and merge controls.',
    icon: GitCompare,
    component: ConfigDiffPage,
    help: (
      <>
        <p>Two ways to merge two YAML configs:</p>
        <ul>
          <li>
            <strong>Block-by-block (top of page)</strong> — choose left or right for each diff
            block. Best when reviewing a small targeted change.
          </li>
          <li>
            <strong>Server-side overlay merge (bottom card)</strong> — same semantics as{' '}
            <code>niac config merge</code>: overlay devices REPLACE base devices with the same name;
            new devices are appended; base-only devices are kept. Best when applying a patch to a
            base config.
          </li>
        </ul>
      </>
    ),
  },
  {
    path: '/pcap-analyzer',
    label: 'PCAP Analysis',
    title: 'PCAP Analysis',
    description: 'Upload and analyze PCAP files with packet inspection, filtering, and statistics.',
    icon: FileBox,
    component: PcapAnalyzerPage,
  },
  {
    path: '/neighbors',
    label: 'Neighbors',
    title: 'Discovery Neighbors',
    description: 'Live CDP / LLDP / EDP / FDP neighbor table for the running simulation.',
    icon: Radar,
    component: NeighborsPage,
    help: (
      <>
        <p>
          Each row is one neighbor advertisement received by a simulated device's CDP / LLDP / EDP /
          FDP listener. The table polls every 5 seconds.
        </p>
        <h4>How it differs from Topology</h4>
        <p>
          The <em>Topology</em> page renders the static, designed network from the YAML config
          (links you specified). This page shows what the simulator's running protocol stack has
          actually <em>discovered</em> over the wire — useful when validating that a device's
          advertisements are seen by its neighbours.
        </p>
        <h4>Tips</h4>
        <ul>
          <li>Filter by protocol with the chips, or search by device / chassis ID.</li>
          <li>
            TTL counts down toward zero; when an advertisement isn't refreshed in time the entry
            ages out of the table.
          </li>
        </ul>
      </>
    ),
  },
  {
    path: '/walk-validator',
    label: 'Walk Validator',
    title: 'SNMP Walk Validator',
    description: 'Validate and auto-fix SNMP walk files (the same engine niac analyze-walk uses).',
    icon: ShieldCheck,
    component: WalkValidatorPage,
    help: (
      <>
        <p>
          A "walk file" is an <code>snmpwalk</code> capture — a flat list of OID = value lines that
          NIAC replays via its simulated SNMP agent. This page wraps the same validator the CLI's{' '}
          <code>niac analyze-walk</code> uses.
        </p>
        <h4>What "Validate" reports</h4>
        <ul>
          <li>
            <strong>error</strong> — line is malformed enough that the parser can't replay it.
          </li>
          <li>
            <strong>warning</strong> — likely-wrong field (suspicious type, encoding mismatch).
          </li>
          <li>
            <strong>info</strong> — stylistic (e.g. unquoted strings). Replay still works.
          </li>
        </ul>
        <h4>Auto-fix</h4>
        <p>
          Auto-fix rewrites the walk in place. A <code>.bak</code> next to the original is created
          before the rewrite so you can roll back. Re-run Validate after fixing to confirm.
        </p>
      </>
    ),
  },
];

// Organize navigation into logical groups
const navGroups: SidebarNavGroup[] = [
  {
    label: 'Control',
    items: [
      { path: '/', label: 'Dashboard', icon: Activity },
      { path: '/runtime', label: 'Simulation', icon: PlugZap },
    ],
  },
  {
    label: 'Configuration',
    items: [
      { path: '/devices', label: 'Live Devices', icon: Server },
      {
        path: '/device-config',
        label: 'Device Library',
        icon: Wrench,
      },
      { path: '/templates', label: 'Templates', icon: FileCode },
      { path: '/config-diff', label: 'Config Diff', icon: GitCompare },
    ],
  },
  {
    label: 'Network',
    items: [
      { path: '/topology', label: 'Topology', icon: Network },
      { path: '/neighbors', label: 'Neighbors', icon: Radar },
      { path: '/traffic', label: 'Traffic Injection', icon: Zap },
    ],
  },
  {
    label: 'Analysis',
    items: [
      { path: '/analysis', label: 'Replay', icon: LineChart },
      { path: '/debug', label: 'Protocol Debug', icon: Terminal },
      { path: '/packets', label: 'Live Packets', icon: FileSearch },
      { path: '/pcap-analyzer', label: 'PCAP Analysis', icon: FileBox },
      { path: '/walk-validator', label: 'Walk Validator', icon: ShieldCheck },
    ],
  },
  {
    label: 'Automation',
    items: [
      {
        path: '/automation',
        label: 'Alerts',
        icon: Workflow,
      },
    ],
  },
];

export default function App() {
  return (
    <ErrorBoundary>
      <AppProvider>
        <AppShell />
      </AppProvider>
    </ErrorBoundary>
  );
}

function AppShell() {
  const { data: version } = useAppState('version');

  useKeyboardShortcuts();

  return (
    <SidebarLayout groups={navGroups} version={version?.version}>
      <ToastContainer />
      <Suspense fallback={<PageLoader />}>
        <Routes>
          {pages.map((page) => (
            <Route
              key={page.path}
              path={page.path}
              element={
                <PageWithErrorBoundary page={page}>
                  <page.component />
                </PageWithErrorBoundary>
              }
            />
          ))}
          {/* Dynamic routes for device editor */}
          <Route
            path="/device-config/new"
            element={
              <PageWithErrorBoundary
                page={{
                  path: '/device-config/new',
                  label: 'New Device',
                  title: 'New Device',
                  description: 'Create a new network device configuration.',
                  icon: Wrench,
                  component: DeviceEditorPage,
                }}
              >
                <DeviceEditorPage />
              </PageWithErrorBoundary>
            }
          />
          <Route
            path="/device-config/:hostname"
            element={
              <PageWithErrorBoundary
                page={{
                  path: '/device-config/:hostname',
                  label: 'Edit Device',
                  title: 'Edit Device',
                  description: 'Edit device configuration settings.',
                  icon: Wrench,
                  component: DeviceEditorPage,
                }}
              >
                <DeviceEditorPage />
              </PageWithErrorBoundary>
            }
          />
          <Route path="*" element={<Navigate to="/" replace={true} />} />
        </Routes>
      </Suspense>
    </SidebarLayout>
  );
}

// Wrapper component to provide location-based key for error boundary reset
// FIX: Error state was persisting across pages because PageErrorBoundary
// wasn't resetting on route changes. Using location.pathname as key forces
// React to unmount/remount the error boundary when navigating.
const PageWithErrorBoundary = memo(
  ({ page, children }: { page: PageConfig; children: ReactNode }) => {
    const location = useLocation();
    return (
      <PageErrorBoundary key={location.pathname}>
        <section className="space-y-6">
          <Breadcrumbs />
          <PageHeader
            icon={page.icon}
            title={page.title}
            description={page.description}
            help={page.help}
          />
          {children}
        </section>
      </PageErrorBoundary>
    );
  },
);

PageWithErrorBoundary.displayName = 'PageWithErrorBoundary';
