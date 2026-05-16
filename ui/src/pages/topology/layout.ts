/**
 * Layout algorithm and edge creation utilities for topology visualization
 */

import { MarkerType } from '@xyflow/react';
import type { DeviceSummary, TopologyLink } from '../../api/types';
import type { DeviceNode, DeviceNodeData, LinkEdge, LinkEdgeData } from './types';
import { linkSpeedColors } from './types';

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

// Minimum radius for the concentric-ring layout. Must be large enough
// that a node on the inner ring doesn't overlap the centre node — i.e.
// > one card-width plus padding. The previous 180 wasn't enough.
const CONCENTRIC_MIN_RADIUS = 320;

/**
 * Layout nodes in concentric circles based on connectivity. Most
 * connected devices are placed in the center.
 *
 * When the topology API returns nodes with no links (most built-in
 * templates declare devices without trunk_ports / port_channels), the
 * concentric-ring path's degree-driven radius schedule collapses to
 * radius=0 and the cards stack on top of one another. We fall back
 * to a fixed-pitch grid so the user at least sees every device — the
 * "no connections" banner above the canvas explains what to add to
 * the YAML to populate edges.
 */
export function layoutNodes(devices: DeviceSummary[], links: TopologyLink[]): DeviceNode[] {
  // For small device counts the concentric-ring layout collapses (the
  // centre node and the first ring overlap), so use the grid path even
  // when there are edges. The cutoff is tuned to the device-card width:
  // ≤4 devices fits in a 2×2 grid with the standard NODE_GAP padding.
  const useGrid = links.length === 0 || devices.length <= 4;

  if (useGrid) {
    return devices.map((device, index) => {
      const cols = Math.max(1, Math.ceil(Math.sqrt(devices.length)));
      const row = Math.floor(index / cols);
      const col = index % cols;
      return {
        id: device.name,
        type: 'device',
        position: {
          x: LAYOUT_LEFT_OFFSET + col * (NODE_WIDTH + NODE_GAP_X),
          y: LAYOUT_TOP_OFFSET + row * (NODE_HEIGHT + NODE_GAP_Y),
        },
        data: {
          label: device.name,
          type: device.type || 'unknown',
          ips: device.ips,
          protocols: device.protocols,
          status: 'online',
        } as DeviceNodeData,
      };
    });
  }

  const nodeMap = new Map<string, { x: number; y: number }>();

  // Create adjacency list
  const adjacency = new Map<string, Set<string>>();
  for (const device of devices) {
    adjacency.set(device.name, new Set());
  }
  for (const link of links) {
    adjacency.get(link.source)?.add(link.target);
    adjacency.get(link.target)?.add(link.source);
  }

  // Calculate degree for each node
  const degrees = new Map<string, number>();
  for (const [name, neighbors] of adjacency) {
    degrees.set(name, neighbors.size);
  }

  // Sort by degree (most connected first)
  const sorted = [...devices].sort(
    (a, b) => (degrees.get(b.name) || 0) - (degrees.get(a.name) || 0),
  );

  // Layout in concentric circles based on connectivity
  const centerX = 400;
  const centerY = 300;
  let currentRadius = 0;
  let angleOffset = 0;
  let nodesInRing = 1;
  let ringIndex = 0;

  for (const device of sorted) {
    if (ringIndex >= nodesInRing) {
      ringIndex = 0;
      currentRadius += CONCENTRIC_MIN_RADIUS;
      angleOffset += Math.PI / 6; // Stagger rings
      nodesInRing = Math.max(1, Math.floor((2 * Math.PI * currentRadius) / 280));
    }

    const angle = (2 * Math.PI * ringIndex) / nodesInRing + angleOffset;
    const x = centerX + currentRadius * Math.cos(angle);
    const y = centerY + currentRadius * Math.sin(angle);

    nodeMap.set(device.name, { x, y });
    ringIndex++;
  }

  // Create nodes with layout positions
  return devices.map((device, index) => {
    const pos = nodeMap.get(device.name) || {
      x: 100 + (index % 5) * 200,
      y: 100 + Math.floor(index / 5) * 150,
    };
    return {
      id: device.name,
      type: 'device',
      position: pos,
      data: {
        label: device.name,
        type: device.type || 'unknown',
        ips: device.ips,
        protocols: device.protocols,
        status: 'online',
      } as DeviceNodeData,
    };
  });
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
    // amber-500 — keeps in-house palette consistent with existing
    // status tints elsewhere in the UI.
    return { stroke: '#f59e0b', strokeWidth: 4 };
  }
  // red-500 for >= 85 %.
  return { stroke: '#ef4444', strokeWidth: 5 };
}
