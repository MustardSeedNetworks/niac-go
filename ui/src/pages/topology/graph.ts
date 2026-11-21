/**
 * The topology page's data pipeline, as pure steps: merge the link sources,
 * narrow to what the filters show, lay out, place labels.
 *
 * Each step takes only what it reads, so the page can memoise them separately.
 * They used to be one effect that also depended on the focused node and the
 * hovered edge, which re-ran dagre over the whole pack on every mouse move.
 */

import type { DeviceSummary, NeighborRecord, TopologyLink } from '../../api/types';
import { normalizeDeviceType } from '../../constants/device-types';
import { type EdgeLabelPlacement, placeLabels } from './labelPlacement';
import { createEdges, type LayoutMode, layoutNodes, pairKey } from './layout';
import type { DeviceNode, LinkEdge } from './types';

/**
 * Trunk-port links from the config, plus a discovered link for every neighbour
 * pair they do not already cover.
 */
export function mergeLinks(
  topologyLinks: TopologyLink[],
  neighbors: NeighborRecord[],
): TopologyLink[] {
  const merged = [...topologyLinks];
  const covered = new Set(topologyLinks.map((l) => pairKey(l.source, l.target)));
  for (const neighbor of neighbors) {
    const key = pairKey(neighbor.localDevice, neighbor.remoteDevice);
    if (covered.has(key)) {
      continue;
    }
    covered.add(key);
    merged.push({
      source: neighbor.localDevice,
      target: neighbor.remoteDevice,
      label: `${neighbor.protocol} discovery`,
      discovered: true,
    });
  }
  return merged;
}

export interface TopologyFilter {
  hidden: ReadonlySet<string>;
  /** Empty means every type. */
  types: ReadonlySet<string>;
  /** Case-insensitive substring of the device name. */
  search: string;
  /** Show only this site's devices; null shows every site. */
  site: string | null;
}

/** The site a pack stamps on each device (`properties.site`), if it has one. */
export function siteOf(device: DeviceSummary): string | undefined {
  return device.properties?.site;
}

/** Distinct sites in the pack, sorted. A one-site pack has nothing to group. */
export function sitesOf(devices: DeviceSummary[]): string[] {
  const sites = new Set<string>();
  for (const device of devices) {
    const site = siteOf(device);
    if (site) {
      sites.add(site);
    }
  }
  return [...sites].sort((a, b) => a.localeCompare(b));
}

/** The devices the filters show, and the links with both ends among them. */
export function filterGraph(
  devices: DeviceSummary[],
  links: TopologyLink[],
  filter: TopologyFilter,
): { devices: DeviceSummary[]; links: TopologyLink[] } {
  const query = filter.search.trim().toLowerCase();
  const visible = devices.filter(
    (device) =>
      !filter.hidden.has(device.name) &&
      (filter.types.size === 0 || filter.types.has(normalizeDeviceType(device.type))) &&
      (filter.site === null || siteOf(device) === filter.site) &&
      (!query || device.name.toLowerCase().includes(query)),
  );
  const names = new Set(visible.map((d) => d.name));
  return {
    devices: visible,
    links: links.filter((l) => names.has(l.source) && names.has(l.target)),
  };
}

export interface TopologyLayout {
  nodes: DeviceNode[];
  edges: LinkEdge[];
  placements: Map<string, EdgeLabelPlacement>;
  links: TopologyLink[];
}

/** Positions, edges and label placement: everything that is expensive. */
export function buildLayout(
  devices: DeviceSummary[],
  links: TopologyLink[],
  mode: LayoutMode,
): TopologyLayout {
  const nodes = layoutNodes(devices, links, mode);
  const edges = createEdges(links);
  return { nodes, edges, placements: placeLabels(nodes, edges), links };
}

/** The node itself and every node linked to it. */
export function neighbourhood(links: TopologyLink[], focused: string): Set<string> {
  const adjacent = new Set([focused]);
  for (const link of links) {
    if (link.source === focused) {
      adjacent.add(link.target);
    } else if (link.target === focused) {
      adjacent.add(link.source);
    }
  }
  return adjacent;
}
