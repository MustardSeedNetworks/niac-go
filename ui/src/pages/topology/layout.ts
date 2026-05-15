/**
 * Layout algorithm and edge creation utilities for topology visualization
 */

import { MarkerType } from '@xyflow/react';
import type { DeviceSummary, TopologyLink } from '../../api/types';
import type { DeviceNode, DeviceNodeData, LinkEdge, LinkEdgeData } from './types';
import { linkSpeedColors } from './types';

// Card-sized tuneables for the edges == 0 grid fallback below. Keep
// neighbours far enough apart that they don't overlap at default zoom.
const NODE_WIDTH = 220;
const NODE_HEIGHT = 130;
const NODE_GAP_X = 60;
const NODE_GAP_Y = 60;

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
  if (links.length === 0) {
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
      currentRadius += 180;
      angleOffset += Math.PI / 6; // Stagger rings
      nodesInRing = Math.max(1, Math.floor((2 * Math.PI * currentRadius) / 200));
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
 * Create edges from topology links with styling based on type/speed
 */
export function createEdges(links: TopologyLink[]): LinkEdge[] {
  return links.map((link, index) => {
    // Parse link label for speed/type info
    const data: LinkEdgeData = {
      label: link.label,
    };

    data.speed = getLinkSpeed(link.label);
    data.linkType = getLinkType(link.label);
    data.vlan = getVlan(link.label);

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
