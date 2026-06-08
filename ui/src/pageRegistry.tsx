import type { LucideIcon } from 'lucide-react';
import {
  Activity,
  Database,
  FileBox,
  GitCompare,
  Network,
  PlugZap,
  Server,
  ShieldCheck,
  Terminal,
  Workflow,
  Wrench,
  Zap,
} from 'lucide-react';
import { type ComponentType, lazy, type ReactNode, useMemo } from 'react';
import { useTranslation } from 'react-i18next';

// Lazy loaded pages.
// Kept here next to the registry so the route list, the lazy-import,
// and the per-page metadata stay in one place — adding a new page
// means editing exactly one file.
const DashboardPage = lazy(() =>
  import('./pages/DashboardPage').then((m) => ({ default: m.DashboardPage })),
);
const RuntimeControlPage = lazy(() =>
  import('./pages/RuntimeControlPage').then((m) => ({ default: m.RuntimeControlPage })),
);
const DevicesPage = lazy(() =>
  import('./pages/DevicesPage').then((m) => ({ default: m.DevicesPage })),
);
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
const AutomationPage = lazy(() =>
  import('./pages/AutomationPage').then((m) => ({ default: m.AutomationPage })),
);
const DebugConsolePage = lazy(() =>
  import('./pages/DebugConsolePage').then((m) => ({ default: m.DebugConsolePage })),
);
const PacketInspectorPage = lazy(() =>
  import('./pages/PacketInspectorPage').then((m) => ({ default: m.PacketInspectorPage })),
);
const TrafficInjectionPage = lazy(() =>
  import('./pages/TrafficInjectionPage').then((m) => ({ default: m.TrafficInjectionPage })),
);
const WalkValidatorPage = lazy(() =>
  import('./pages/WalkValidatorPage').then((m) => ({ default: m.WalkValidatorPage })),
);
const LibraryWalksPage = lazy(() =>
  import('./pages/LibraryFilesPage').then((m) => ({ default: m.LibraryWalksPage })),
);
const LibraryPcapsPage = lazy(() =>
  import('./pages/LibraryFilesPage').then((m) => ({ default: m.LibraryPcapsPage })),
);

/**
 * PageConfig is one entry in the application's route table, resolved
 * at render time via usePages() — `label/title/description` are
 * translations of the corresponding pages.{i18nKey}.* keys.
 */
export type PageConfig = {
  path: string;
  label: string;
  title: string;
  description: string;
  icon: LucideIcon;
  component: ComponentType;
  badge?: string;
  help?: ReactNode;
};

/**
 * PageI18nKey is the closed set of pages.* namespaces that carry a
 * matching {label,title,description} triple. Kept strict so adding a
 * new route forces a corresponding locale entry.
 */
type PageI18nKey =
  | 'dashboard'
  | 'runtime'
  | 'devices'
  | 'deviceLibrary'
  | 'topology'
  | 'automation'
  | 'traffic'
  | 'debug'
  | 'packets'
  | 'configDiff'
  | 'walkValidator'
  | 'libraryWalks'
  | 'libraryPcaps';

/**
 * PageDef is the static, language-agnostic definition stored in
 * pageRegistry. The matching translation lives at pages.{i18nKey}.{label,
 * title,description} in internal/i18n/locales/{en,es}/pages.json.
 */
type PageDef = {
  path: string;
  i18nKey: PageI18nKey;
  icon: LucideIcon;
  component: ComponentType;
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
 * staticPages holds the route table — paths, components, icons, and
 * the verbose help: JSX prose (still English-only; queued for Phase 7
 * help-content i18n). Labels/titles/descriptions are resolved at
 * render time by usePages() via the pages.{i18nKey}.* locale keys.
 */
const staticPages: PageDef[] = [
  {
    path: '/',
    i18nKey: 'dashboard',
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
            <strong>Packets</strong> to watch traffic in real time.
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
    i18nKey: 'runtime',
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
            <strong>Traffic</strong> lets you switch the playback file at runtime.
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
    i18nKey: 'devices',
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
          To modify saved device definitions, go to <strong>Devices</strong> (under Library). To
          swap the running network wholesale, restart the simulation from{' '}
          <strong>Simulation</strong>.
        </p>
      </>
    ),
  },
  {
    path: '/device-config',
    i18nKey: 'deviceLibrary',
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
    i18nKey: 'topology',
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
          The graph shows the <em>design</em> from your YAML; the neighbor table shows which
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
    path: '/automation',
    i18nKey: 'automation',
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
            className="text-brand-accent underline"
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
    i18nKey: 'traffic',
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
    i18nKey: 'debug',
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
    i18nKey: 'packets',
    icon: FileBox,
    component: PacketInspectorPage,
    help: (
      <>
        <h4>Live capture</h4>
        <p>
          Streams every packet the daemon sees from <code>/api/v1/stream/packets</code> — hex +
          decoded fields, BPF filter, freeze frame, save to PCAP. If the page lags, narrow the BPF
          filter (e.g. limit to a single protocol or device IP).
        </p>
        <h4>PCAP files</h4>
        <p>
          Switch to the <strong>PCAP files</strong> tab to open and inspect a captured file —
          protocol breakdown, per-conversation stats, full per-packet decode. Equivalent of{' '}
          <code>niac analyze-pcap</code> on the CLI.
        </p>
      </>
    ),
  },
  {
    path: '/config-diff',
    i18nKey: 'configDiff',
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
    path: '/walk-validator',
    i18nKey: 'walkValidator',
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
  {
    path: '/library/walks',
    i18nKey: 'libraryWalks',
    icon: Database,
    component: LibraryWalksPage,
    help: (
      <>
        <p>
          Shows every <code>.walk</code> file under <code>~/.niac/library/walks/</code> (or{' '}
          <code>/var/lib/niac/library/walks/</code> on packaged installs). Drop files in directly,
          or run <code>niac content install</code> to fetch the published bundle for this binary's
          version.
        </p>
        <p>
          The SNMP section on the Device editor uses the same endpoint to populate its walk picker,
          so anything that shows up here is immediately selectable when configuring a device.
        </p>
      </>
    ),
  },
  {
    path: '/library/pcaps',
    i18nKey: 'libraryPcaps',
    icon: FileBox,
    component: LibraryPcapsPage,
    help: (
      <>
        <p>
          Shows every capture under <code>~/.niac/library/pcaps/</code>. Same ingress paths as the
          walk library: drop files directly or use <code>niac content install</code>.
        </p>
        <p>
          The Packets and Traffic pages will reuse this listing for their PCAP pickers — the unified
          library means a PCAP added here is visible everywhere the daemon needs to pick one without
          extra plumbing.
        </p>
      </>
    ),
  },
];

/**
 * usePages returns the full route table with label/title/description
 * resolved against the active locale. Replaces the prior `pages` export
 * so consumers no longer ship hardcoded English in their JSX.
 *
 * The translation keys live at pages.{i18nKey}.{label,title,description}.
 * usePages does not depend on any runtime state, but is a hook so
 * react-i18next's languageChanged event re-renders consumers.
 */
export function usePages(): PageConfig[] {
  const { t } = useTranslation('pages');
  return useMemo(
    () =>
      staticPages.map((p) => ({
        path: p.path,
        label: t(`${p.i18nKey}.label`),
        title: t(`${p.i18nKey}.title`),
        description: t(`${p.i18nKey}.description`),
        icon: p.icon,
        component: p.component,
        badge: p.badge,
        help: p.help,
      })),
    [t],
  );
}
