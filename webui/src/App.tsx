import { createElement, type ReactNode, type FC } from 'react';
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
  Link2,
  SatelliteDish,
  FileCog,
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
    description: 'Monitor simulation health, packet pipelines, and recent automation tasks at a glance.',
    icon: Activity,
    Component: DashboardPage,
  },
  {
    path: '/devices',
    label: 'Devices & Config',
    title: 'Devices & Configuration',
    description: 'Author YAML configs, attach SNMP walks, and manage DHCP/DNS personas for every simulated node.',
    icon: Server,
    Component: DevicesPage,
  },
  {
    path: '/topology',
    label: 'Topology & Neighbors',
    title: 'Topology & Neighbor Insight',
    description: 'Visualize LLDP/CDP/EDP/FDP discoveries, link states, and intent gaps before exporting to Graphviz.',
    icon: Network,
    Component: TopologyPage,
  },
  {
    path: '/analysis',
    label: 'Analysis',
    title: 'Analysis & Playback',
    description: 'Replay captures, explore SNMP walks, and publish shareable analysis bundles directly from the browser.',
    icon: LineChart,
    Component: AnalysisPage,
  },
  {
    path: '/automation',
    label: 'Automation',
    title: 'Automation & Alerts',
    description: 'Define alert thresholds, webhook targets, and upcoming workflow automations for NIAC runs.',
    icon: Workflow,
    Component: AutomationPage,
    badge: 'Beta',
  },
];

const navItems: NavItem[] = pages.map((page) => ({
  label: page.label,
  path: page.path,
  icon: createElement(page.icon, { className: 'h-4 w-4' }),
  badge: page.badge,
}));

export default function App() {
  return (
    <PageShell>
      <div className="space-y-10">
        <div className="rounded-2xl border border-white/5 bg-gray-900/60 p-4 backdrop-blur">
          <PrimaryNav
            items={navItems}
            className="flex-wrap gap-2"
          />
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
      </div>
    </PageShell>
  );
}

function PageTemplate({ page, children }: { page: PageConfig; children: ReactNode }) {
  return (
    <section className="space-y-6">
      <PageHeader icon={page.icon} title={page.title} description={page.description} />
      {children}
    </section>
  );
}

function DashboardPage() {
  const telemetry = [
    { label: 'SNMP Walk Streams', value: '12 active', detail: 'ciscolive.walk, dc-core.walk' },
    { label: 'Packet Playback', value: '3 pcaps', detail: 'ospf-lab.pcap, access-edge.pcap' },
    { label: 'DHCP/DNS Profiles', value: '4 services', detail: 'campus-dhcp, lab-dns' },
  ];

  const automation = [
    { time: 'Just now', title: 'Topology export ready', detail: 'DOT + PNG generated from neighbor table' },
    { time: '2 min', title: 'Webhook delivered', detail: 'Alerted when packet count exceeded 25k' },
    { time: '15 min', title: 'Run history snapshot', detail: 'Persisted BoltDB checkpoint to storage path' },
  ];

  return (
    <div className="space-y-6">
      <div className="grid gap-6 lg:grid-cols-3">
        <Card className="lg:col-span-2 border-white/5 bg-gradient-to-br from-gray-900/70 to-gray-950/80">
          <CardContent className="space-y-5">
            <div className="flex flex-wrap items-center gap-3">
              <Tag>RUNNING</Tag>
              <Tag colorScheme="purple">Debug: 2</Tag>
            </div>
            <H2 className="mb-2">Atlas campus fabric</H2>
            <P className="text-gray-300">
              Simulating 11 nodes, 3 VLAN domains, and dual-stack services. Traffic generation and SNMP agents are active
              with scheduled analysis exports every 15 minutes.
            </P>
            <div className="grid gap-4 sm:grid-cols-3">
              <div>
                <SmallText className="uppercase tracking-wide text-gray-400">Neighbors</SmallText>
                <p className="text-3xl font-bold text-white">26</p>
                <SmallText className="text-blue-300">LLDP/CDP/EDP/FDP</SmallText>
              </div>
              <div>
                <SmallText className="uppercase tracking-wide text-gray-400">Packets</SmallText>
                <p className="text-3xl font-bold text-white">148k</p>
                <SmallText className="text-gray-300">RX/TX combined</SmallText>
              </div>
              <div>
                <SmallText className="uppercase tracking-wide text-gray-400">Uptime</SmallText>
                <p className="text-3xl font-bold text-white">01h 22m</p>
                <SmallText className="text-gray-300">live since refresh</SmallText>
              </div>
            </div>
            <div className="flex flex-wrap gap-3">
              <Button tone="violet" leftIcon={<Activity className="h-4 w-4" />}>Open live logs</Button>
              <Button variant="outline" leftIcon={<PlugZap className="h-4 w-4" />}>Inject traffic</Button>
              <Button variant="ghost" leftIcon={<ShieldCheck className="h-4 w-4" />}>Trigger alert test</Button>
            </div>
          </CardContent>
        </Card>
        <Card className="border-white/5 bg-gray-900/70">
          <CardContent className="space-y-4">
            <H2 className="mb-1">Active telemetry</H2>
            <div className="space-y-4">
              {telemetry.map((stream) => (
                <div
                  key={stream.label}
                  className="rounded-xl border border-white/5 bg-gray-900/60 p-4"
                >
                  <SmallText className="uppercase tracking-wide text-gray-400">{stream.label}</SmallText>
                  <p className="text-xl font-semibold text-white">{stream.value}</p>
                  <SmallText className="text-gray-400">{stream.detail}</SmallText>
                </div>
              ))}
            </div>
            <AccentLink to="/analysis" className="text-indigo-300">
              Manage feeds in Analysis →
            </AccentLink>
          </CardContent>
        </Card>
      </div>

      <div className="grid gap-6 lg:grid-cols-3">
        <Card className="border-white/5 bg-gray-900/70 lg:col-span-2">
          <CardContent className="space-y-4">
            <div className="flex items-center justify-between">
              <H2 className="mb-0 flex items-center gap-2">
              <SatelliteDish className="h-5 w-5 text-violet-300" />
                Automation timeline
              </H2>
              <Tag colorScheme="gray">Latest events</Tag>
            </div>
            <div className="space-y-4">
              {automation.map((event) => (
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
        <Card className="border-white/5 bg-gray-900/70">
          <CardContent className="space-y-4">
            <H2 className="mb-1 flex items-center gap-2">
              <BellRing className="h-5 w-5 text-amber-300" />
              Upcoming tasks
            </H2>
            <ul className="space-y-3 text-sm text-gray-300">
              <li className="rounded-lg border border-white/5 bg-gray-950/50 p-3">
                Reschedule packet replay for access-edge.pcap after current capture completes.
              </li>
              <li className="rounded-lg border border-white/5 bg-gray-950/50 p-3">
                Append newly learned neighbors to Graphviz export and re-publish.
              </li>
              <li className="rounded-lg border border-white/5 bg-gray-950/50 p-3">
                Validate DHCP scope alignment before rolling over to lab-nightly config.
              </li>
            </ul>
            <Button variant="outline" size="sm" leftIcon={<PlugZap className="h-4 w-4" />}>
              Add task
            </Button>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

function DevicesPage() {
  return (
    <div className="space-y-6">
      <Card className="border-white/5 bg-gray-900/70">
        <CardContent className="space-y-4">
          <H2 className="mb-0 flex items-center gap-2">
            <Server className="h-5 w-5 text-cyan-300" />
            Config workspace
          </H2>
          <P>
            The NIAC runtime still sources truth from YAML configs. Use the editor below (or your favorite IDE)
            to adjust devices, SNMP walks, DHCP/DNS personas, and packet playback settings.
          </P>
          <ol className="list-decimal space-y-3 pl-5 text-gray-300">
            <li>Load an existing config via <code className="rounded bg-black/40 px-1">docs/examples</code> or upload your own.</li>
            <li>Associate SNMP walks (*.walk) with each device to seed realistic responses.</li>
            <li>Attach PCAP files for replay, toggling continuous or on-demand playback.</li>
            <li>Save the draft and hit <strong>Apply to runtime</strong> to restart services with the new definition.</li>
          </ol>
          <div className="flex flex-wrap gap-3">
            <Button tone="violet" leftIcon={<Server className="h-4 w-4" />}>Open editor</Button>
            <Button variant="outline" leftIcon={<Link2 className="h-4 w-4" />}>Attach SNMP walk</Button>
            <Button variant="ghost" leftIcon={<PlugZap className="h-4 w-4" />}>Queue packet replay</Button>
          </div>
        </CardContent>
      </Card>

      <Card className="border-white/5 bg-gray-900/70">
        <CardContent className="space-y-3">
          <H2 className="mb-0">Recent config events</H2>
          <ul className="space-y-2 text-sm text-gray-300">
            <li>Imported <code>campus-core.yaml</code> (devices: 11)</li>
            <li>Linked <code>dist-stack.walk</code> to DIST01/DIST02</li>
            <li>Scheduled <code>access-edge.pcap</code> for 23:00 replay</li>
          </ul>
          <AccentLink to="/analysis">See history &gt;</AccentLink>
        </CardContent>
      </Card>
    </div>
  );
}

function TopologyPage() {
  const neighbors = [
    { device: 'CORE01', remote: 'DIST02', via: 'Gi0/1', protocol: 'LLDP', mgmt: '10.10.20.2', latency: '1.2 ms' },
    { device: 'CORE01', remote: 'DIST01', via: 'Gi0/2', protocol: 'CDP', mgmt: '10.10.20.1', latency: '1.0 ms' },
    { device: 'DIST02', remote: 'ACCESS-A', via: 'Gi1/0/24', protocol: 'EDP', mgmt: '10.10.30.12', latency: '3.4 ms' },
    { device: 'ACCESS-A', remote: 'AP-113', via: 'Gi0/5', protocol: 'FDP', mgmt: '10.10.40.113', latency: '0.8 ms' },
  ];

  return (
    <div className="space-y-6">
      <Card className="border-white/5 bg-gray-900/70">
        <CardContent className="space-y-4">
          <H2 className="mb-0">Discovered neighbors</H2>
          <div className="overflow-x-auto rounded-xl border border-white/5">
            <table className="min-w-full divide-y divide-white/10 text-sm">
              <thead className="bg-gray-900/60 text-xs uppercase tracking-wide text-gray-400">
                <tr>
                  <th className="px-4 py-3 text-left">Local device</th>
                  <th className="px-4 py-3 text-left">Remote</th>
                  <th className="px-4 py-3 text-left">Protocol</th>
                  <th className="px-4 py-3 text-left">Mgmt address</th>
                  <th className="px-4 py-3 text-left">Last seen</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-white/5 text-gray-300">
                {neighbors.map((neighbor) => (
                  <tr key={`${neighbor.device}-${neighbor.remote}`}>
                    <td className="px-4 py-3 font-semibold text-white">{neighbor.device} · {neighbor.via}</td>
                    <td className="px-4 py-3 text-white/90">{neighbor.remote}</td>
                    <td className="px-4 py-3">
                      <Tag colorScheme="purple">{neighbor.protocol}</Tag>
                    </td>
                    <td className="px-4 py-3 font-mono text-sm">{neighbor.mgmt}</td>
                    <td className="px-4 py-3 text-gray-400">{neighbor.latency}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <div className="flex flex-wrap gap-3">
            <Button leftIcon={<Network className="h-4 w-4" />}>Launch live topology</Button>
            <Button variant="outline" leftIcon={<LineChart className="h-4 w-4" />}>Export Graphviz</Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

function AnalysisPage() {
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
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="rounded-xl border border-white/5 bg-gray-950/50 p-4">
              <SmallText className="uppercase tracking-wide text-gray-400">PCAP queue</SmallText>
              <p className="text-2xl font-semibold text-white">ospf-core-lab.pcap</p>
              <SmallText className="text-gray-400">Ready to replay (~4 min)</SmallText>
            </div>
            <div className="rounded-xl border border-white/5 bg-gray-950/50 p-4">
              <SmallText className="uppercase tracking-wide text-gray-400">Walk bundle</SmallText>
              <p className="text-2xl font-semibold text-white">dc-core.walk</p>
              <SmallText className="text-gray-400">Applied to CORE01/02</SmallText>
            </div>
          </div>
          <div className="flex flex-wrap gap-3">
            <Button tone="violet" leftIcon={<LineChart className="h-4 w-4" />}>Open analyzer</Button>
            <Button variant="outline" leftIcon={<PlugZap className="h-4 w-4" />}>Queue replay</Button>
            <Button variant="ghost" leftIcon={<FileCog className="h-4 w-4" />}>Export bundle</Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

function AutomationPage() {
  return (
    <div className="space-y-6">
      <Card className="border-white/5 bg-gray-900/70">
        <CardContent className="space-y-4">
          <H2 className="mb-0 flex items-center gap-2">
            <Workflow className="h-5 w-5 text-yellow-300" />
            Automation roadmap
          </H2>
          <P>
            Define alert thresholds, webhook routes, and (soon) runnable workflow automations. The browser UI is the new home
            for kickstarting replay jobs, rotating configs, and acknowledging alerts.
          </P>
          <ul className="space-y-3 text-sm text-gray-300">
            <li className="rounded-lg border border-white/5 bg-gray-950/50 p-3">Webhooks inherit settings from the `--alert-webhook` flag and can be overridden per run.</li>
            <li className="rounded-lg border border-white/5 bg-gray-950/50 p-3">Packet/neighbor thresholds align with CLI options so headless + web control stay in sync.</li>
            <li className="rounded-lg border border-white/5 bg-gray-950/50 p-3">Future: orchestrate multi-run scenarios and publish signed run reports.</li>
          </ul>
          <Button variant="outline" leftIcon={<BellRing className="h-4 w-4" />}>Configure alerts</Button>
        </CardContent>
      </Card>
    </div>
  );
}
