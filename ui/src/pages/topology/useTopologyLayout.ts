import { useMemo } from 'react';
import type { DeviceSummary, NeighborRecord, TopologyGraph } from '../../api/types';
import {
  buildLayout,
  filterGraph,
  mergeLinks,
  type TopologyFilter,
  type TopologyLayout,
} from './graph';
import type { LayoutMode } from './layout';

interface TopologyLayoutInput {
  devices: DeviceSummary[] | null;
  topology: TopologyGraph | null;
  neighbors: NeighborRecord[] | null;
  filter: TopologyFilter;
  layoutMode: LayoutMode;
  /** Bumped to force a fresh layout over unchanged data (Reset, Refresh). */
  revision: number;
}

/**
 * The laid-out graph for the current data, filters and layout mode.
 *
 * Memoised so that what changes often (the focused node, the hovered edge, the
 * label toggle) never reaches the layout: dagre over 543 devices is hundreds of
 * milliseconds, and only a change to the data or the filters is worth it.
 */
export function useTopologyLayout({
  devices,
  topology,
  neighbors,
  filter,
  layoutMode,
  revision,
}: TopologyLayoutInput): TopologyLayout | null {
  const links = useMemo(
    () => (topology ? mergeLinks(topology.links ?? [], neighbors ?? []) : null),
    [topology, neighbors],
  );
  const visible = useMemo(
    () => (devices && links ? filterGraph(devices, links, filter) : null),
    [devices, links, filter],
  );
  return useMemo(() => {
    // The revision carries no data; naming it here is what makes a bump re-run this.
    void revision;
    return visible ? buildLayout(visible.devices, visible.links, layoutMode) : null;
  }, [visible, layoutMode, revision]);
}
