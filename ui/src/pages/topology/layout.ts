/**
 * Layout algorithm and edge creation utilities for topology visualization
 */

import { MarkerType } from '@xyflow/react';
import type { DeviceSummary, TopologyLink } from '../../api/types';
import type { DeviceNode, DeviceNodeData, LinkEdge, LinkEdgeData } from './types';
import { linkSpeedColors } from './types';

// Card-sized tuneables for the grid layout below. Keep neighbours far
// enough apart that they don't overlap at default zoom. DeviceNode has
// maxWidth 220 + 2px border + room for the connector handles ReactFlow
// draws, plus we give the user some breathing room.
const NODE_WIDTH = 240;
const NODE_HEIGHT = 160;
const NODE_GAP_X = 80;
const NODE_GAP_Y = 80;

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
          x: col * (NODE_WIDTH + NODE_GAP_X),
          y: row * (NODE_HEIGHT + NODE_GAP_Y),
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

    const edgeColor = getEdgeColor(data);

    return {
      id: `e-${link.source}-${link.target}-${index}`,
      source: link.source,
      target: link.target,
      type: 'smoothstep',
      animated: data.linkType === 'trunk' || data.linkType === 'lag',
      label: link.label,
      labelBgPadding: [8, 4] as [number, number],
      labelBgBorderRadius: 4,
      labelBgStyle: {
        fill: 'var(--color-bg-overlay)',
        fillOpacity: 0.9,
      },
      labelStyle: {
        fill: 'var(--color-text-secondary)',
        fontSize: 10,
        fontWeight: 500,
      },
      style: {
        stroke: edgeColor,
        strokeWidth: data.linkType === 'trunk' || data.linkType === 'lag' ? 3 : 2,
      },
      markerEnd: {
        type: MarkerType.ArrowClosed,
        color: edgeColor,
        width: 15,
        height: 15,
      },
      data,
    };
  });
}
