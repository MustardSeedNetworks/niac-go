import {
  Activity,
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
 * Groups are ordered to follow the natural session flow:
 *
 *   1. Overview   — am I running? start/stop the sim.
 *   2. Library    — pick the network + manage device definitions.
 *   3. Live View  — look at the currently running sim.
 *   4. Inspect    — debug logs, packets, walk files.
 *   5. Alerts     — notify me when things break.
 */
export const navGroups: SidebarNavGroup[] = [
  {
    label: 'Overview',
    items: [
      { path: '/', label: 'Dashboard', icon: Activity },
      { path: '/runtime', label: 'Simulation', icon: PlugZap },
    ],
  },
  {
    label: 'Library',
    items: [
      { path: '/device-config', label: 'Devices', icon: Wrench },
      { path: '/config-diff', label: 'Compare & Merge', icon: GitCompare },
    ],
  },
  {
    label: 'Live View',
    items: [
      { path: '/devices', label: 'Running Devices', icon: Server },
      { path: '/topology', label: 'Topology', icon: Network },
      { path: '/traffic', label: 'Traffic', icon: Zap },
    ],
  },
  {
    label: 'Inspect',
    items: [
      { path: '/debug', label: 'Logs', icon: Terminal },
      { path: '/packets', label: 'Packets', icon: FileBox },
      { path: '/walk-validator', label: 'SNMP Walks', icon: ShieldCheck },
    ],
  },
  {
    label: 'Alerts',
    items: [{ path: '/automation', label: 'Alerts', icon: Workflow }],
  },
];
