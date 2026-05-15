import type { LucideIcon } from 'lucide-react';
import {
  Activity,
  FileBox,
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
import { type FC, lazy, type ReactNode } from 'react';

// Eagerly loaded pages (small, frequently used)
import { DashboardPage } from './pages/DashboardPage';
import { DevicesPage } from './pages/DevicesPage';
import { RuntimeControlPage } from './pages/RuntimeControlPage';

// Lazy loaded pages (large dependencies or infrequently used).
// Kept here next to the registry so the route list, the lazy-import,
// and the per-page metadata stay in one place — adding a new page
// means editing exactly one file.
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

/**
 * PageConfig is one entry in the application's route table. Beyond the
 * usual path/component pair it carries the page header metadata
 * (title, description, icon) and an optional contextual help body
 * rendered by the page chrome.
 */
export type PageConfig = {
  path: string;
  label: string;
  title: string;
  description: string;
  icon: LucideIcon;
  component: FC;
  badge?: string;
  help?: ReactNode;
};

/**
 * DeviceEditorPageRef is exported so dynamic routes in App.tsx
 * (/device-config/new and /device-config/:hostname) can reuse the
 * lazy import without re-declaring it.
 */
export const DeviceEditorPageRef = DeviceEditorPage;

/**
 * pages is the route table consumed by App.tsx. Each entry has a
 * stable path, a memoised component, and per-page chrome content.
 * Ordering here doesn't affect routing but DOES affect any code
 * that iterates pages in display order (none does today).
 */
export const pages: PageConfig[] = [
  {
    path: '/',
    label: 'Dashboard',
    title: 'Dashboard',
    description: 'Live counters, run snapshots, and automation status for the active NIAC stack.',
    icon: Activity,
    component: DashboardPage,
    help: (
      <>
        <p>
          Real-time view of the running daemon: packet counters per protocol, recent simulation
          runs, device count, and recent alert state.
        </p>
        <h4>Where to go from here</h4>
        <ul>
          <li>
            <strong>Simulation</strong> to start/stop a run.
          </li>
          <li>
            <strong>Running Devices</strong> to see what the running config produced.
          </li>
          <li>
            <strong>Packet Capture</strong> to watch traffic in real time.
          </li>
          <li>
            <strong>Alerts</strong> to set the threshold and webhook target.
          </li>
        </ul>
      </>
    ),
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
    label: 'Running Devices',
    title: 'Running Devices',
    description:
      'Read-only view of the devices the daemon is currently simulating, plus the running YAML.',
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
    help: (
      <>
        <p>
          Visual graph of the configured topology — devices and the links you declared in the YAML (
          <code>trunk_ports:</code>, <code>port_channels:</code>, etc.). Use this to sanity-check
          your design before starting the simulation.
        </p>
        <h4>Live discovery vs design</h4>
        <p>
          The graph shows the <em>design</em>. Use <strong>Neighbors</strong> to see which
          adjacencies actually formed at runtime via CDP / LLDP / EDP / FDP — useful for catching
          mistyped <code>remote_device</code> references.
        </p>
        <h4>Export</h4>
        <p>
          Topology can be exported as Graphviz <code>.dot</code> or JSON via the export button.
        </p>
      </>
    ),
  },
  {
    path: '/analysis',
    label: 'Replay',
    title: 'Replay & Playback',
    description: 'Replay PCAPs, inspect SNMP walks, and publish bundles directly from the UI.',
    icon: LineChart,
    component: AnalysisPage,
    help: (
      <>
        <p>
          Drive the daemon's PCAP replay engine: pick a file, set a loop interval and time scale,
          start. Equivalent of <code>POST /api/v1/replay</code>.
        </p>
        <h4>Auto-started replays</h4>
        <p>
          A <code>capture_playbacks:</code> block in your YAML auto-starts when the simulation
          starts. This page reflects whatever is currently running and lets you swap files without
          editing the config.
        </p>
        <h4>File sources</h4>
        <ul>
          <li>Files placed under the daemon's pcap directory show up in the dropdown.</li>
          <li>
            Use <strong>PCAP Analysis</strong> to inspect a file before replaying.
          </li>
        </ul>
      </>
    ),
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
    help: (
      <>
        <p>
          Inject controlled errors (drops, delays, corruption) into the running simulation to test
          how upstream tooling reacts. Drives <code>POST /api/v1/errors</code>.
        </p>
        <h4>Common faults</h4>
        <ul>
          <li>
            <strong>Drop rate</strong> — percentage of packets to drop on the device's egress.
          </li>
          <li>
            <strong>Delay</strong> — extra latency in milliseconds.
          </li>
          <li>
            <strong>Corruption</strong> — flip random bits in the payload.
          </li>
        </ul>
        <p>
          Errors clear when you stop the simulation, or via <code>niac inject clear</code> on the
          CLI.
        </p>
      </>
    ),
  },
  {
    path: '/debug',
    label: 'Protocol Debug',
    title: 'Protocol Debug',
    description: 'Real-time log streaming and debugging tools for monitoring NIAC operations.',
    icon: Terminal,
    component: DebugConsolePage,
    help: (
      <>
        <p>
          Live log tail from the daemon (Server-Sent Events from <code>/api/v1/stream/logs</code>).
          Pause to freeze the buffer, filter by protocol or severity, set per-protocol debug levels
          (0=quiet, 3=verbose).
        </p>
        <h4>Per-protocol debug levels</h4>
        <p>
          Equivalent of the CLI's <code>--debug-arp</code> / <code>--debug-icmp</code> /{' '}
          <code>--debug-snmp</code> family of flags. Levels are applied to the running stack
          immediately.
        </p>
      </>
    ),
  },
  {
    path: '/packets',
    label: 'Packet Capture',
    title: 'Packet Capture',
    description:
      'Real-time packet hex dump viewing with protocol filtering and search capabilities.',
    icon: FileSearch,
    component: PacketInspectorPage,
    help: (
      <>
        <p>
          Live wire view of every packet the daemon sees — hex + decoded fields, BPF filter, freeze
          frame, save to PCAP. Streams from <code>/api/v1/stream/packets</code>.
        </p>
        <h4>Performance</h4>
        <p>
          Frames flow as fast as the wire. If the page lags, narrow the BPF filter (e.g. limit to a
          single protocol or device IP) instead of trying to render everything.
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
    label: 'PCAP Inspector',
    title: 'PCAP Inspector',
    description:
      'Open and inspect a captured PCAP file offline — protocol breakdown, per-conversation stats, full per-packet decode.',
    icon: FileBox,
    component: PcapAnalyzerPage,
    help: (
      <>
        <p>
          Offline analysis of a PCAP: protocol breakdown, per-conversation stats, full per-packet
          decode. Equivalent of <code>niac analyze-pcap</code> on the CLI.
        </p>
        <h4>Live vs offline</h4>
        <p>
          For traffic on the running simulator, use <strong>Packet Capture</strong>. This page is
          for static analysis of files you've captured elsewhere.
        </p>
      </>
    ),
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
