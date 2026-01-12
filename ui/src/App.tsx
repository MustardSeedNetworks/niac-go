import { memo, type FC, type ReactNode } from 'react';
import { Routes, Route, Navigate } from 'react-router-dom';
import type { LucideIcon } from 'lucide-react';
import {
  Activity,
  Server,
  Network,
  LineChart,
  Workflow,
  PlugZap,
  Zap,
  Terminal,
  FileSearch,
  FileCode,
  GitCompare,
  FileBox,
  Wrench,
} from 'lucide-react';
import { SidebarLayout, PageHeader } from './ui';
import type { SidebarNavGroup } from './ui';
import { useApiResource } from './hooks/useApiResource';
import { fetchVersion } from './api/client';
import { POLL_INTERVALS } from './constants';
import { TrafficInjectionPage } from './pages/TrafficInjectionPage';
import { DebugConsolePage } from './pages/DebugConsolePage';
import { PacketInspectorPage } from './pages/PacketInspectorPage';
import { TemplatesPage } from './pages/TemplatesPage';
import { ConfigDiffPage } from './pages/ConfigDiffPage';
import { PcapAnalyzerPage } from './pages/PcapAnalyzerPage';
import { DeviceListPage } from './pages/DeviceListPage';
import { DeviceEditorPage } from './pages/DeviceEditorPage';
import { TopologyPage } from './pages/TopologyPage';
import { RuntimeControlPage } from './pages/RuntimeControlPage';
import { DashboardPage } from './pages/DashboardPage';
import { DevicesPage } from './pages/DevicesPage';
import { AnalysisPage } from './pages/AnalysisPage';
import { AutomationPage } from './pages/AutomationPage';
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

// Memoize PageTemplate to prevent unnecessary re-renders
const PageTemplate = memo(({ page, children }: { page: PageConfig; children: ReactNode }) => {
  return (
    <section className="space-y-6">
      <PageHeader icon={page.icon} title={page.title} description={page.description} />
      {children}
    </section>
  );
});

PageTemplate.displayName = 'PageTemplate';
