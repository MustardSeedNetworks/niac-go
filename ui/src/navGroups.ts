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
import type { SidebarNavGroup } from './ui/Sidebar';

/**
 * navGroups drives the left sidebar — the grouped list of routes the
 * user can click to. The labels here are what appear in the sidebar;
 * the page titles / descriptions live in pageRegistry alongside the
 * route handlers themselves. Keep the two in rough sync but they're
 * deliberately not the same source: sidebar wants short, page header
 * wants verbose.
 *
 * Each item carries an `i18nKey` (namespace `pages`) so the sidebar
 * can render localised labels via `t(item.i18nKey, item.label)`.
 * The English `label` doubles as the i18next fallback when the
 * translation is missing, so reads continue to work even if a key
 * is renamed before the locale catches up.
 *
 * Groups are ordered to follow the natural session flow:
 *
 *   1. Overview   — am I running? start/stop the sim.
 *   2. Library    — pick the network + manage device definitions.
 *   3. Live View  — look at the currently running sim.
 *   4. Inspect    — debug logs, packets, walk files.
 *   5. Alerts     — notify me when things break.
 *
 * Network terminology that must remain verbatim across locales
 * (protocol names, standards, units, abbreviations like SNMP, PCAP,
 * IP, MAC, MTU, VLAN, CIDR) is kept out of the translated strings
 * and is therefore safe to read directly from the English label.
 */
export const navGroups: SidebarNavGroup[] = [
  {
    label: 'Overview',
    i18nKey: 'pages:groups.overview',
    items: [
      { path: '/', label: 'Dashboard', i18nKey: 'pages:dashboard.label', icon: Activity },
      { path: '/runtime', label: 'Simulation', i18nKey: 'pages:runtime.label', icon: PlugZap },
    ],
  },
  {
    label: 'Library',
    i18nKey: 'pages:groups.library',
    items: [
      {
        path: '/device-config',
        label: 'Devices',
        i18nKey: 'pages:deviceLibrary.label',
        icon: Wrench,
      },
      {
        path: '/library/walks',
        label: 'Walks',
        i18nKey: 'pages:libraryWalks.label',
        icon: Database,
      },
      { path: '/library/pcaps', label: 'PCAPs', icon: FileBox },
      {
        path: '/config-diff',
        label: 'Compare & Merge',
        i18nKey: 'pages:configDiff.label',
        icon: GitCompare,
      },
    ],
  },
  {
    label: 'Live View',
    i18nKey: 'pages:groups.liveView',
    items: [
      { path: '/devices', label: 'Running Devices', i18nKey: 'pages:devices.label', icon: Server },
      { path: '/topology', label: 'Topology', i18nKey: 'pages:topology.label', icon: Network },
      { path: '/traffic', label: 'Traffic', i18nKey: 'pages:traffic.label', icon: Zap },
    ],
  },
  {
    label: 'Inspect',
    i18nKey: 'pages:groups.inspect',
    items: [
      { path: '/debug', label: 'Logs', i18nKey: 'pages:debug.label', icon: Terminal },
      { path: '/packets', label: 'Packets', i18nKey: 'pages:packets.label', icon: FileBox },
      { path: '/walk-validator', label: 'SNMP Walks', icon: ShieldCheck },
    ],
  },
  {
    label: 'Alerts',
    i18nKey: 'pages:groups.alerts',
    items: [
      { path: '/automation', label: 'Alerts', i18nKey: 'pages:automation.label', icon: Workflow },
    ],
  },
];
