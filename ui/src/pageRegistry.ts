import type { LucideIcon } from 'lucide-react';
import {
  Activity,
  Database,
  FileBox,
  FileScan,
  GitCompare,
  Layers,
  Network,
  PlugZap,
  Server,
  ShieldCheck,
  Terminal,
  Wand2,
  Workflow,
  Wrench,
  Zap,
} from 'lucide-react';
import { type ComponentType, lazy, useMemo } from 'react';
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
const NewSimulationWizardPage = lazy(() =>
  import('./pages/NewSimulationWizardPage').then((m) => ({ default: m.NewSimulationWizardPage })),
);
const DevicesPage = lazy(() =>
  import('./pages/DevicesPage').then((m) => ({ default: m.DevicesPage })),
);
const SegmentsPage = lazy(() =>
  import('./pages/SegmentsPage').then((m) => ({ default: m.SegmentsPage })),
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
const WalkAnalyzerPage = lazy(() =>
  import('./pages/WalkAnalyzerPage').then((m) => ({ default: m.WalkAnalyzerPage })),
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
  /** Kicker above the title naming the product domain. */
  eyebrow?: string;
  title: string;
  description: string;
  icon: LucideIcon;
  component: ComponentType;
  badge?: string;
};

/**
 * PageI18nKey is the closed set of pages.* namespaces that carry a
 * matching {label,title,description} triple. Kept strict so adding a
 * new route forces a corresponding locale entry.
 */
type PageI18nKey =
  | 'dashboard'
  | 'runtime'
  | 'newSimWizard'
  | 'devices'
  | 'segments'
  | 'deviceLibrary'
  | 'topology'
  | 'automation'
  | 'traffic'
  | 'debug'
  | 'packets'
  | 'configDiff'
  | 'walkValidator'
  | 'walkAnalyzer'
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
  interpolation?: Record<string, string>;
  icon: LucideIcon;
  component: ComponentType;
  badge?: string;
};

/**
 * DeviceEditorPageRef is exported so dynamic routes in App.tsx
 * (/device-config/new and /device-config/:hostname) can reuse the
 * lazy import without re-declaring it.
 */
export const DeviceEditorPageRef = DeviceEditorPage;

/**
 * staticPages holds the route table — paths, components, icons.
 * Labels/titles/descriptions are resolved at render time by usePages()
 * via the pages.{i18nKey}.* locale keys; each page's help prose lives in
 * data/page-help.ts, keyed by the same path.
 */
const staticPages: PageDef[] = [
  {
    path: '/',
    i18nKey: 'dashboard',
    icon: Activity,
    component: DashboardPage,
  },
  {
    path: '/runtime',
    i18nKey: 'runtime',
    icon: PlugZap,
    component: RuntimeControlPage,
  },
  {
    path: '/new-simulation',
    i18nKey: 'newSimWizard',
    icon: Wand2,
    component: NewSimulationWizardPage,
  },
  {
    path: '/devices',
    i18nKey: 'devices',
    icon: Server,
    component: DevicesPage,
  },
  {
    path: '/segments',
    i18nKey: 'segments',
    icon: Layers,
    component: SegmentsPage,
  },
  {
    path: '/device-config',
    i18nKey: 'deviceLibrary',
    icon: Wrench,
    component: DeviceListPage,
  },
  {
    path: '/topology',
    i18nKey: 'topology',
    interpolation: { protocols: 'CDP/LLDP/EDP/FDP' },
    icon: Network,
    component: TopologyPage,
  },
  {
    path: '/automation',
    i18nKey: 'automation',
    icon: Workflow,
    component: AutomationPage,
  },
  {
    path: '/traffic',
    i18nKey: 'traffic',
    interpolation: { format: 'PCAP' },
    icon: Zap,
    component: TrafficInjectionPage,
  },
  {
    path: '/debug',
    i18nKey: 'debug',
    icon: Terminal,
    component: DebugConsolePage,
  },
  {
    path: '/packets',
    i18nKey: 'packets',
    interpolation: { format: 'PCAP' },
    icon: FileBox,
    component: PacketInspectorPage,
  },
  {
    path: '/config-diff',
    i18nKey: 'configDiff',
    interpolation: { format: 'YAML' },
    icon: GitCompare,
    component: ConfigDiffPage,
  },
  {
    path: '/walk-validator',
    i18nKey: 'walkValidator',
    interpolation: { protocol: 'SNMP' },
    icon: ShieldCheck,
    component: WalkValidatorPage,
  },
  {
    path: '/walk-analyzer',
    i18nKey: 'walkAnalyzer',
    icon: FileScan,
    component: WalkAnalyzerPage,
  },
  {
    path: '/library/walks',
    i18nKey: 'libraryWalks',
    interpolation: { protocol: 'SNMP' },
    icon: Database,
    component: LibraryWalksPage,
  },
  {
    path: '/library/pcaps',
    i18nKey: 'libraryPcaps',
    interpolation: { format: 'PCAP' },
    icon: FileBox,
    component: LibraryPcapsPage,
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
        label: t(`${p.i18nKey}.label`, p.interpolation),
        // A page has an eyebrow when its locale namespace declares one, so the
        // copy lives in one place instead of being mirrored by a flag here.
        // Pages still awaiting their archetype pass have none.
        eyebrow: t(`${p.i18nKey}.eyebrow`, { ...p.interpolation, defaultValue: '' }) || undefined,
        title: t(`${p.i18nKey}.title`, p.interpolation),
        description: t(`${p.i18nKey}.description`, p.interpolation),
        icon: p.icon,
        component: p.component,
        badge: p.badge,
      })),
    [t],
  );
}
