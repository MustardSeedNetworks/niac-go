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
  Server,
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
  },
  {
    path: '/devices',
    label: 'Devices & Config',
    title: 'Devices & Configuration',
    description:
      'Review YAML-derived devices, SNMP walks, DHCP/DNS personas, and packet playback targets.',
    icon: Server,
    component: DevicesPage,
  },
  {
    path: '/device-config',
    label: 'Config Builder',
    title: 'Visual Config Builder',
    description: 'Create, edit, and manage device configurations with a visual interface.',
    icon: Wrench,
    component: DeviceListPage,
    badge: 'New',
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
    label: 'Automation',
    title: 'Automation & Alerts',
    description: 'Configure alert thresholds, webhook targets, and future workflow automations.',
    icon: Workflow,
    component: AutomationPage,
    badge: 'Beta',
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
  },
  {
    path: '/config-diff',
    label: 'Config Diff',
    title: 'Config Diff & Merge',
    description: 'Compare and merge YAML configuration files with visual diff and merge controls.',
    icon: GitCompare,
    component: ConfigDiffPage,
  },
  {
    path: '/pcap-analyzer',
    label: 'PCAP Analysis',
    title: 'PCAP Analysis',
    description: 'Upload and analyze PCAP files with packet inspection, filtering, and statistics.',
    icon: FileBox,
    component: PcapAnalyzerPage,
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
      { path: '/devices', label: 'Devices & Config', icon: Server },
      {
        path: '/device-config',
        label: 'Config Builder',
        icon: Wrench,
        badge: 'New',
      },
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
      { path: '/analysis', label: 'Replay', icon: LineChart },
      { path: '/debug', label: 'Protocol Debug', icon: Terminal },
      { path: '/packets', label: 'Live Packets', icon: FileSearch },
      { path: '/pcap-analyzer', label: 'PCAP Analysis', icon: FileBox },
    ],
  },
  {
    label: 'Automation',
    items: [
      {
        path: '/automation',
        label: 'Alerts & Workflows',
        icon: Workflow,
        badge: 'Beta',
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
          <PageHeader icon={page.icon} title={page.title} description={page.description} />
          {children}
        </section>
      </PageErrorBoundary>
    );
  },
);

PageWithErrorBoundary.displayName = 'PageWithErrorBoundary';
