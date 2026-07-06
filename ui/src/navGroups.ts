import {
  Activity,
  Database,
  FileBox,
  FileScan,
  GitCompare,
  KeyRound,
  Network,
  PlugZap,
  Server,
  ShieldCheck,
  Terminal,
  Workflow,
  Wrench,
  Zap,
} from 'lucide-react';
import { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import type { SidebarNavGroup } from './ui/Sidebar';

/**
 * useNavGroups returns the left sidebar's grouped route list, with
 * group labels and item labels resolved from the active locale.
 *
 * Groups are ordered to follow the natural session flow:
 *
 *   1. Overview   — am I running? start/stop the sim.
 *   2. Library    — pick the network + manage device definitions.
 *   3. Live View  — look at the currently running sim.
 *   4. Inspect    — debug logs, packets, walk files.
 *   5. Alerts     — notify me when things break.
 *
 * Sidebar labels are deliberately shorter than the corresponding page
 * header titles in pageRegistry; both source from the same pages.*
 * namespace so translators only see one canonical label per route.
 */
export function useNavGroups(): SidebarNavGroup[] {
  const { t } = useTranslation('pages');
  return useMemo(
    () => [
      {
        label: t('groups.overview'),
        items: [
          { path: '/', label: t('dashboard.label'), icon: Activity },
          { path: '/runtime', label: t('runtime.label'), icon: PlugZap },
          { path: '/license', label: t('license.label'), icon: KeyRound },
        ],
      },
      {
        label: t('groups.library'),
        items: [
          { path: '/device-config', label: t('deviceLibrary.label'), icon: Wrench },
          { path: '/library/walks', label: t('libraryWalks.label'), icon: Database },
          {
            path: '/library/pcaps',
            label: t('libraryPcaps.label', { format: 'PCAP' }),
            icon: FileBox,
          },
          { path: '/config-diff', label: t('configDiff.label'), icon: GitCompare },
        ],
      },
      {
        label: t('groups.liveView'),
        items: [
          { path: '/devices', label: t('devices.label'), icon: Server },
          { path: '/topology', label: t('topology.label'), icon: Network },
          { path: '/traffic', label: t('traffic.label'), icon: Zap },
        ],
      },
      {
        label: t('groups.inspect'),
        items: [
          { path: '/debug', label: t('debug.label'), icon: Terminal },
          { path: '/packets', label: t('packets.label'), icon: FileBox },
          {
            path: '/walk-validator',
            label: t('walkValidator.label', { protocol: 'SNMP' }),
            icon: ShieldCheck,
          },
          { path: '/walk-analyzer', label: t('walkAnalyzer.label'), icon: FileScan },
        ],
      },
      {
        label: t('groups.alerts'),
        items: [{ path: '/automation', label: t('automation.label'), icon: Workflow }],
      },
    ],
    [t],
  );
}
