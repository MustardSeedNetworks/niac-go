/**
 * Layout algorithm and edge creation utilities for topology visualization
 */

import { MarkerType } from '@xyflow/react';
import dagre from 'dagre';
import type { DeviceSummary, TopologyLink } from '../../api/types';
import type { DeviceNode, DeviceNodeData, LinkEdge, LinkEdgeData } from './types';
import { linkSpeedColors } from './types';

/**
 * LayoutMode selects which positioning algorithm `layoutNodes` runs.
 * Hierarchical is the default — it matches how operators draw network
 * diagrams on a whiteboard (core top, access bottom). Grid is the
 * fallback for small device sets or when the operator wants a flat
 * connectivity-agnostic view.
 *
 * Concentric was tried in earlier versions but operators found it
 * hard to read — rings around the most-connected device collapse on
 * small graphs and become a tangle past ~6 nodes. Removed in favour
 * of just the two modes that work.
 */
export type LayoutMode = 'hierarchical' | 'grid';

export const DEFAULT_LAYOUT_MODE: LayoutMode = 'hierarchical';

/** All modes, in the display order the picker pill row uses. */
export const LAYOUT_MODES: { mode: LayoutMode; label: string; description: string }[] = [
  {
    mode: 'hierarchical',
    label: 'Hierarchical',
    description: 'Layered top-down — core at top, access at bottom',
  },
  {
    mode: 'grid',
    label: 'Grid',
    description: 'Fixed-pitch grid; ignores connectivity',
  },
];

// The node's own footprint: DeviceNode is a 64 px symbol plate over a label
// allowed two lines, inside a fixed 112 px width, so every node occupies this
// box whatever its name is.
export const NODE_WIDTH = 112;
export const NODE_HEIGHT = 96;

// Separation is sized by the edge labels, not by the nodes — "Gi0/1 ↔ Gi0/1 ·
// VLANs 1-30" is the widest thing between two nodes and did not shrink when
// the cards became symbols. Deriving these from the node size (as they were,
// when the node was card-sized) tied two unrelated things together, so
// shrinking the node would have squeezed the labels.
const NODE_GAP_X = 160;
const NODE_GAP_Y = 100;

/** Space between leaves packed under one device. Tighter than a rank: these
 * are siblings on the same switch, and the block reads as one group. */
const LEAF_GAP_X = 28;
const LEAF_GAP_Y = 34;

/** Horizontal room between two nodes on the same rank, for a trunk label. */
const NODE_SEPARATION = 180;

/** Vertical room between ranks, for the per-endpoint interface labels. */
const RANK_SEPARATION = 160;

// Left margin so the legend Panel (rendered as a top-left overlay,
// ~260 px wide) doesn't obscure the first column of nodes.
const LAYOUT_LEFT_OFFSET = 280;
const LAYOUT_TOP_OFFSET = 40;

/**
 * layoutNodes returns ReactFlow node positions according to the
 * selected mode. Hierarchical is the default — it matches how
 * operators draw networks (core/distribution/access top-down) and
 * scales gracefully past 10+ devices where the concentric layout
 * becomes hard to read.
 *
 * Edge-less topologies (no trunk_ports / port_channels declared)
 * always render as a grid regardless of mode — without links there
 * is no hierarchy to derive, and the concentric path collapses to a
 * radius of zero. The "no connections" banner above the canvas
 * already nudges the user toward declaring some.
 */
export function layoutNodes(
  devices: DeviceSummary[],
  links: TopologyLink[],
  mode: LayoutMode = DEFAULT_LAYOUT_MODE,
): DeviceNode[] {
  if (links.length === 0 || devices.length === 0) {
    return gridLayout(devices);
  }
  switch (mode) {
    case 'grid':
      return gridLayout(devices);
    default:
      return hierarchicalLayout(devices, links);
  }
}

/**
 * gridLayout drops devices into a fixed-pitch sqrt(n)-wide grid. Used
 * by the no-edges fallback, the "Grid" picker option, and as the
 * concentric layout's small-set escape hatch (≤4 devices, where the
 * inner ring would overlap the centre).
 */
function gridLayout(devices: DeviceSummary[]): DeviceNode[] {
  const cols = Math.max(1, Math.ceil(Math.sqrt(devices.length)));
  return devices.map((device, index) => {
    const row = Math.floor(index / cols);
    const col = index % cols;
    return {
      id: device.name,
      type: 'device',
      position: {
        x: LAYOUT_LEFT_OFFSET + col * (NODE_WIDTH + NODE_GAP_X),
        y: LAYOUT_TOP_OFFSET + row * (NODE_HEIGHT + NODE_GAP_Y),
      },
      data: makeData(device),
    };
  });
}

/**
 * Devices whose only link is to one other device, grouped by that device.
 *
 * Most of a real network is leaves: the hospital pack is 78 devices and 56 of
 * them are an access point, a phone or a workstation hanging off one switch.
 * Ranked with everything else they land side by side on the deepest rank, so
 * the graph comes out about 5,000 units wide and six ranks tall — a shape no
 * screen can show at once, and the reason the packs were unreadable (#2106).
 *
 * A device with no links at all is not a leaf: it has no parent to sit under,
 * and dagre still has to place it.
 */
function leavesByParent(devices: DeviceSummary[], links: TopologyLink[]): Map<string, string[]> {
  const present = new Set(devices.map((device) => device.name));
  const neighbours = new Map<string, Set<string>>();
  for (const link of links) {
    if (!present.has(link.source) || !present.has(link.target)) {
      continue;
    }
    const add = (from: string, to: string): void => {
      const set = neighbours.get(from) ?? new Set<string>();
      set.add(to);
      neighbours.set(from, set);
    };
    add(link.source, link.target);
    add(link.target, link.source);
  }

  const grouped = new Map<string, string[]>();
  for (const device of devices) {
    const peers = neighbours.get(device.name);
    if (peers?.size !== 1) {
      continue;
    }
    const [parent] = [...peers];
    // A pair of devices linked only to each other are each other's only peer.
    // Packing one under the other would be arbitrary, so leave both ranked.
    if (parent === undefined || (neighbours.get(parent)?.size ?? 0) <= 1) {
      continue;
    }
    grouped.set(parent, [...(grouped.get(parent) ?? []), device.name]);
  }
  return grouped;
}

/** The grid a device's leaves are packed into, in columns and rows. */
function leafGrid(count: number): { columns: number; rows: number } {
  const columns = Math.max(1, Math.ceil(Math.sqrt(count)));
  return { columns, rows: Math.ceil(count / columns) };
}

/** The footprint a device needs: itself, plus the block of leaves beneath it. */
function packedSize(leafCount: number): { width: number; height: number } {
  if (leafCount === 0) {
    return { width: NODE_WIDTH, height: NODE_HEIGHT };
  }
  const { columns, rows } = leafGrid(leafCount);
  return {
    width: Math.max(NODE_WIDTH, columns * NODE_WIDTH + (columns - 1) * LEAF_GAP_X),
    height: NODE_HEIGHT + LEAF_GAP_Y + rows * NODE_HEIGHT + (rows - 1) * LEAF_GAP_Y,
  };
}

/**
 * hierarchicalLayout runs dagre's network-simplex ranking algorithm
 * top-to-bottom. Each device gets a row determined by its longest
 * incoming-edge path from a root; siblings get spread horizontally.
 *
 * For operators this reads like a whiteboard diagram: core devices at
 * the top, distribution in the middle, access at the bottom. The
 * default direction is TB (top-to-bottom) which matches the way
 * "uplinks" are drawn in physical network docs.
 *
 * dagre is a pure-JS DAG layouter (~30 KB gzipped). It treats the
 * trunk/access edges as directed but topology links from the daemon
 * are undirected — that's fine; dagre just picks an orientation,
 * which is enough to drive the rank assignment.
 */
function hierarchicalLayout(devices: DeviceSummary[], links: TopologyLink[]): DeviceNode[] {
  const g = new dagre.graphlib.Graph();
  // Spacing tuned by trial against the kitchen-sink template, and sized by the
  // edge labels rather than by the nodes:
  //  - NODE_SEPARATION so siblings on the same rank don't crowd each other;
  //    the trunk-edge floating labels ("Gi0/1 ↔ Gi0/1 · VLANs 1-30") sit
  //    comfortably between them.
  //  - RANK_SEPARATION so the per-side interface labels at each edge endpoint
  //    clear the nodes above and below.
  //  - edgesep so dagre doesn't collapse parallel edges between two ranks
  //    into a visual stack.
  g.setGraph({
    rankdir: 'TB',
    nodesep: NODE_SEPARATION,
    ranksep: RANK_SEPARATION,
    edgesep: 60,
    marginx: LAYOUT_LEFT_OFFSET,
    marginy: LAYOUT_TOP_OFFSET,
  });
  g.setDefaultEdgeLabel(() => ({}));

  // Leaves are packed under their parent rather than ranked, and the parent is
  // sized to hold them so dagre reserves the room rather than overlapping the
  // block with whatever it ranks next.
  const leaves = leavesByParent(devices, links);
  const packed = new Set([...leaves.values()].flat());

  for (const device of devices) {
    if (packed.has(device.name)) {
      continue;
    }
    g.setNode(device.name, packedSize(leaves.get(device.name)?.length ?? 0));
  }
  // Dedup parallel edges between the same node pair — dagre's ranking
  // doesn't benefit from seeing the same pair twice (we already merge
  // bidirectional trunks in BuildTopology, but neighbour-discovered
  // adjacencies can echo a trunk).
  const seenEdgePairs = new Set<string>();
  for (const link of links) {
    if (packed.has(link.source) || packed.has(link.target)) continue;
    const key = [link.source, link.target].sort().join('|');
    if (seenEdgePairs.has(key)) continue;
    seenEdgePairs.add(key);
    g.setEdge(link.source, link.target);
  }

  dagre.layout(g);

  // Dagre is used for what it is good at — which rank a device belongs to, and
  // the order of devices within it. Placement is ours, because dagre lays a
  // rank out as one row however wide it gets: enterprise-scale ranks 48
  // switches together, about 14,000 units across against a graph 2,500 tall,
  // which is a horizontal smear on any screen (#2106).
  const ranks = rankOrder(devices, packed, g);

  // A rank wider than this wraps onto another row, the way a paragraph does.
  // The width comes from the device count rather than being picked: the same
  // sqrt that gridLayout uses, which keeps a graph roughly as wide as it is
  // tall whatever its size.
  const wrapColumns = Math.max(1, Math.ceil(Math.sqrt(ranks.flat().length)));

  const rows: string[][] = [];
  const rowRank: number[] = [];
  ranks.forEach((rank, index) => {
    for (let start = 0; start < rank.length; start += wrapColumns) {
      rows.push(rank.slice(start, start + wrapColumns));
      rowRank.push(index);
    }
  });

  const sizeOf = (name: string): { width: number; height: number } =>
    packedSize(leaves.get(name)?.length ?? 0);
  const rowWidth = (row: string[]): number =>
    row.reduce((total, name) => total + sizeOf(name).width, 0) +
    Math.max(0, row.length - 1) * NODE_SEPARATION;
  const widest = Math.max(...rows.map(rowWidth), NODE_WIDTH);

  const positions = new Map<string, { x: number; y: number }>();
  const rankOf = new Map<string, number>();
  let cursorY = LAYOUT_TOP_OFFSET;

  rows.forEach((row, rowIndex) => {
    // Rows are centred on each other so a wrapped rank reads as one band
    // rather than drifting left.
    let cursorX = LAYOUT_LEFT_OFFSET + (widest - rowWidth(row)) / 2;
    let tallest = NODE_HEIGHT;

    for (const name of row) {
      const size = sizeOf(name);
      tallest = Math.max(tallest, size.height);
      positions.set(name, { x: cursorX + size.width / 2 - NODE_WIDTH / 2, y: cursorY });
      rankOf.set(name, rowRank[rowIndex] ?? 0);

      const children = leaves.get(name) ?? [];
      if (children.length > 0) {
        const { columns } = leafGrid(children.length);
        const blockWidth = columns * NODE_WIDTH + (columns - 1) * LEAF_GAP_X;
        const blockLeft = cursorX + (size.width - blockWidth) / 2;
        const blockTop = cursorY + NODE_HEIGHT + LEAF_GAP_Y;
        children.forEach((child, index) => {
          positions.set(child, {
            x: blockLeft + (index % columns) * (NODE_WIDTH + LEAF_GAP_X),
            y: blockTop + Math.floor(index / columns) * (NODE_HEIGHT + LEAF_GAP_Y),
          });
          rankOf.set(child, (rowRank[rowIndex] ?? 0) + 1);
        });
      }
      cursorX += size.width + NODE_SEPARATION;
    }
    cursorY += tallest + RANK_SEPARATION;
  });

  return devices.map((device) => ({
    id: device.name,
    type: 'device' as const,
    position: positions.get(device.name) ?? { x: LAYOUT_LEFT_OFFSET, y: LAYOUT_TOP_OFFSET },
    data: { ...makeData(device), rank: rankOf.get(device.name) ?? 0 },
  }));
}

/**
 * The ranked devices dagre produced, top rank first and in dagre's own
 * left-to-right order within each rank.
 *
 * Dagre centres a rank's nodes on one y, but a device sized to hold a block of
 * leaves is taller than its neighbours, so those y values drift. Devices are
 * bucketed rather than grouped by exact value, the same way the tier bands do
 * it.
 */
function rankOrder(
  devices: DeviceSummary[],
  packed: Set<string>,
  g: dagre.graphlib.Graph,
): string[][] {
  const placed = devices
    .filter((device) => !packed.has(device.name))
    .map((device) => ({ name: device.name, node: g.node(device.name) }))
    .filter((entry) => entry.node !== undefined)
    .sort((left, right) => left.node.y - right.node.y || left.node.x - right.node.x);

  const ranks: string[][] = [];
  let anchor = Number.NaN;
  for (const entry of placed) {
    if (Number.isNaN(anchor) || entry.node.y - anchor > NODE_HEIGHT) {
      anchor = entry.node.y;
      ranks.push([]);
    }
    ranks[ranks.length - 1]?.push(entry.name);
  }
  return ranks;
}

/** Common DeviceNodeData shape used by every layout fn. */
function makeData(device: DeviceSummary): DeviceNodeData {
  return {
    label: device.name,
    type: device.type || 'unknown',
    ips: device.ips,
    protocols: device.protocols,
  };
}

/**
 * Extract link speed from label string (e.g., "1G" -> "1000")
 */
export function getLinkSpeed(label?: string): string | undefined {
  if (!label) {
    return;
  }
  // A VLAN id in the label is not a speed. Drop it before parsing, so
  // 'vlan 10 100' reads 100 Mbps rather than 10.
  const withoutVlan = label.replace(/\bvlans?\s*\d+/gi, ' ');

  // A number carrying a unit is the speed even when a bare number precedes
  // it; only fall back to a bare number when the label has no unit at all.
  const speedMatch = /(\d+)\s*([MGT])\b/i.exec(withoutVlan) ?? /(\d+)/.exec(withoutVlan);
  if (!speedMatch) {
    return;
  }
  const digits = speedMatch[1];
  if (digits === undefined) {
    return;
  }
  const num = Number.parseInt(digits, 10);
  const unit = speedMatch[2]?.toUpperCase() || 'M';
  const multiplier = unit === 'G' ? 1000 : unit === 'T' ? 1000000 : 1;
  return String(num * multiplier);
}

/**
 * Detect link type from label (trunk, lag, standard)
 */
export function getLinkType(label?: string): LinkEdgeData['linkType'] | undefined {
  const normalized = label?.toLowerCase();
  if (!normalized) {
    return;
  }
  if (normalized.includes('trunk')) {
    return 'trunk';
  }
  if (normalized.includes('lag') || normalized.includes('po')) {
    return 'lag';
  }
  return;
}

/**
 * Extract VLAN number from label string
 */
export function getVlan(label?: string): number | undefined {
  if (!label) {
    return;
  }
  const vlanMatch = /vlan\s*(\d+)/i.exec(label);
  if (!vlanMatch) {
    return;
  }
  const digits = vlanMatch[1];
  return digits === undefined ? undefined : Number.parseInt(digits, 10);
}

/**
 * Get edge color based on link type and speed
 */
export function getEdgeColor(data: LinkEdgeData): string {
  // linkSpeedColors is keyed by speeds that arrive in the data, so it stays an
  // open Record and every read — including trunk — falls back rather than
  // asserting the key is there.
  const fallback = 'var(--color-border-muted)';
  if (data.linkType === 'trunk' || data.linkType === 'lag') {
    return linkSpeedColors.trunk ?? fallback;
  }
  return (data.speed ? linkSpeedColors[data.speed] : undefined) ?? fallback;
}

/**
 * Create edges from topology links with styling based on type/speed.
 *
 * Prefers the structured fields the daemon now returns
 * (link.linkType / link.speed / link.vlans) over regex-parsing the
 * display label — the label is human-friendly text and rarely
 * contains the literal "trunk" / "1G" keywords the old parser needed.
 */
/**
 * siblingIndices numbers each edge among all the edges touching its source
 * device, and again among all those touching its target.
 *
 * Counted per device, not per device-and-end. What crowds a switch is every
 * label that wants to sit next to it, and that is both the edges leaving it
 * and the edges arriving at it — measuring showed an arriving edge's target
 * label landing on a departing edge's source label when the two ends were
 * counted separately.
 */
function siblingIndices(links: TopologyLink[]): { source: number[]; target: number[] } {
  const seen = new Map<string, number>();
  const next = (device: string): number => {
    const index = seen.get(device) ?? 0;
    seen.set(device, index + 1);
    return index;
  };
  const source: number[] = [];
  const target: number[] = [];
  for (const link of links) {
    source.push(next(link.source));
    target.push(next(link.target));
  }
  return { source, target };
}

export function createEdges(links: TopologyLink[]): LinkEdge[] {
  const siblings = siblingIndices(links);

  return links.map((link, index) => {
    const data: LinkEdgeData = {
      label: link.label,
      // Prefer server-supplied structure; fall back to label parsing for
      // older API responses that didn't include the typed fields.
      speed: link.speed ?? getLinkSpeed(link.label),
      linkType: (link.linkType as LinkEdgeData['linkType']) ?? getLinkType(link.label),
      vlan: link.vlans && link.vlans.length > 0 ? link.vlans[0] : getVlan(link.label),
    };

    // Hydrate per-side interface names + vlan list so the custom
    // TrunkEdge component can render them as end-labels near each
    // arrow. The middle label (VLANs + speed) sits on the line; the
    // per-side labels float between line and device card.
    data.sourceInterface = link.sourceInterface;
    data.targetInterface = link.targetInterface;
    data.sourceSiblingIndex = siblings.source[index];
    data.targetSiblingIndex = siblings.target[index];
    data.vlans = link.vlans;
    data.discovered = link.discovered;
    data.utilizationPercent = link.utilizationPercent;

    const baseColor = getEdgeColor(data);
    const { stroke, strokeWidth: utilWidth } = utilizationStyle(data.utilizationPercent, baseColor);
    // Trunk + LAG links are bidirectional by nature (both sides send
    // and receive on the same wire) — show arrows on both ends. A
    // plain access link still uses a single end-arrow to indicate the
    // "source declared this" direction.
    const bidirectional = data.linkType === 'trunk' || data.linkType === 'lag';
    // Stroke width = max(base-by-direction, utilisation-driven). High
    // utilisation should visibly thicken even an access link; trunks
    // start at 3 px so they don't shrink when utilisation is unknown.
    const baseWidth = bidirectional ? 3 : 2;
    const strokeWidth = Math.max(baseWidth, utilWidth);
    const marker = {
      type: MarkerType.ArrowClosed,
      color: stroke,
      width: 14,
      height: 14,
    };

    return {
      id: `e-${link.source}-${link.target}-${index}`,
      source: link.source,
      target: link.target,
      // 'trunk' is our custom edge type registered in TopologyPage's
      // edgeTypes lookup. Labels are rendered by TrunkEdge, not via
      // the standard label/labelStyle props.
      type: 'trunk',
      animated: bidirectional,
      style: {
        stroke,
        strokeWidth,
      },
      markerEnd: marker,
      ...(bidirectional ? { markerStart: marker } : {}),
      // ReactFlow renders an invisible wider stroke for hit-testing;
      // 20 px gives the user a comfortable target for hover-tooltips
      // even on thin 2 px access links. Pan/zoom still works because
      // ReactFlow only consumes events on actual edge hover/click.
      interactionWidth: 20,
      data,
    };
  });
}

/**
 * utilizationStyle maps a 0–100 utilisation percent to a stroke
 * width + colour tint. Buckets match the design table in #552:
 *
 *   0 – 24   : default colour, base width (no change)
 *   25 – 59  : default colour, 3 px
 *   60 – 84  : amber, 4 px
 *   85+      : red, 5 px
 *
 * When utilisation is undefined or 0, we return the unmodified base
 * colour + the smallest non-zero width sentinel (0) so the caller's
 * Math.max picks the direction-based default. Callers MUST take
 * Math.max(baseWidth, returned-width).
 */
// Utilization edge colors, exported so the topology legend documents the
// exact same palette the graph renders — single source of truth, no drift
// between the swatch and the wire. (This file is an allow-listed domain
// palette in check-token-discipline.sh; the legend imports these instead of
// re-hardcoding the hex.)
export const UTILIZATION_HIGH_COLOR = '#f59e0b'; // amber-500, 60–84%
export const UTILIZATION_CRITICAL_COLOR = '#ef4444'; // red-500, >= 85%

function utilizationStyle(
  utilisation: number | undefined,
  baseColor: string,
): { stroke: string; strokeWidth: number } {
  if (utilisation === undefined || utilisation <= 0) {
    return { stroke: baseColor, strokeWidth: 0 };
  }
  if (utilisation < 25) {
    return { stroke: baseColor, strokeWidth: 0 };
  }
  if (utilisation < 60) {
    return { stroke: baseColor, strokeWidth: 3 };
  }
  if (utilisation < 85) {
    return { stroke: UTILIZATION_HIGH_COLOR, strokeWidth: 4 };
  }
  return { stroke: UTILIZATION_CRITICAL_COLOR, strokeWidth: 5 };
}
