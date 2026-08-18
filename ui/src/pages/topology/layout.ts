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

// Card-sized tuneables for the grid layout below. Tuned to the
// DeviceNode max-width (260 px) plus the trunk-edge label band that
// floats between cards. NODE_GAP_X is generous so labels like
// "Gi0/1 ↔ Gi0/1 (VLANs 1-30)" sit comfortably without overrunning
// either card.
const NODE_WIDTH = 280;
const NODE_HEIGHT = 180;
const NODE_GAP_X = 160;
const NODE_GAP_Y = 100;

// Left margin so the legend Panel (rendered as a top-left overlay,
// ~260 px wide) doesn't obscure the first column of cards.
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
  // Spacing tuned by trial against the kitchen-sink template:
  //  - nodesep doubled (160 → 320) so siblings on the same rank
  //    don't crowd each other; the trunk-edge floating labels
  //    ("Gi0/1 ↔ Gi0/1 · VLANs 1-30") sit comfortably between cards.
  //  - ranksep tripled (100+60 → 280) so the per-side interface
  //    labels at each edge endpoint clear the cards above and below.
  //  - edgesep added so dagre doesn't collapse parallel edges between
  //    two ranks into a visual stack.
  g.setGraph({
    rankdir: 'TB',
    nodesep: NODE_GAP_X * 2,
    ranksep: NODE_GAP_Y + 180,
    edgesep: 60,
    marginx: LAYOUT_LEFT_OFFSET,
    marginy: LAYOUT_TOP_OFFSET,
  });
  g.setDefaultEdgeLabel(() => ({}));

  for (const device of devices) {
    g.setNode(device.name, { width: NODE_WIDTH, height: NODE_HEIGHT });
  }
  // Dedup parallel edges between the same node pair — dagre's ranking
  // doesn't benefit from seeing the same pair twice (we already merge
  // bidirectional trunks in BuildTopology, but neighbour-discovered
  // adjacencies can echo a trunk).
  const seenEdgePairs = new Set<string>();
  for (const link of links) {
    const key = [link.source, link.target].sort().join('|');
    if (seenEdgePairs.has(key)) continue;
    seenEdgePairs.add(key);
    g.setEdge(link.source, link.target);
  }

  dagre.layout(g);

  return devices.map((device) => {
    const node = g.node(device.name);
    // Dagre positions reference the centre of the box; ReactFlow
    // positions reference the top-left. Offset accordingly.
    const x = (node?.x ?? 0) - NODE_WIDTH / 2;
    const y = (node?.y ?? 0) - NODE_HEIGHT / 2;
    return {
      id: device.name,
      type: 'device',
      position: { x, y },
      data: makeData(device),
    };
  });
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
  const speedMatch = /(\d+)([MGT])?/i.exec(label);
  if (!speedMatch) {
    return;
  }
  const num = Number.parseInt(speedMatch[1], 10);
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
  return Number.parseInt(vlanMatch[1], 10);
}

/**
 * Get edge color based on link type and speed
 */
export function getEdgeColor(data: LinkEdgeData): string {
  if (data.linkType === 'trunk' || data.linkType === 'lag') {
    return linkSpeedColors.trunk;
  }
  if (data.speed && linkSpeedColors[data.speed]) {
    return linkSpeedColors[data.speed];
  }
  return 'var(--color-border-muted)';
}

/**
 * Create edges from topology links with styling based on type/speed.
 *
 * Prefers the structured fields the daemon now returns
 * (link.linkType / link.speed / link.vlans) over regex-parsing the
 * display label — the label is human-friendly text and rarely
 * contains the literal "trunk" / "1G" keywords the old parser needed.
 */
export function createEdges(links: TopologyLink[]): LinkEdge[] {
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
