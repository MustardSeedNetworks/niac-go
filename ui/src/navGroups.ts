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
import type { SidebarNavGroup } from './ui/Sidebar';

/**
 * navGroups drives the left sidebar — the grouped list of routes the
 * user can click to. The labels here are what appear in the sidebar;
 * the page titles / descriptions live in pageRegistry alongside the
 * route handlers themselves. Keep the two in rough sync but they're
 * deliberately not the same source: sidebar wants short, page header
 * wants verbose.
 */
export const navGroups: SidebarNavGroup[] = [
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
      { path: '/devices', label: 'Running Devices', icon: Server },
      {
        path: '/device-config',
        label: 'Device Library',
        icon: Wrench,
      },
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
      { path: '/packets', label: 'Packet Capture', icon: FileSearch },
      { path: '/pcap-analyzer', label: 'PCAP Inspector', icon: FileBox },
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
