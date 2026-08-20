/**
 * Tier bands for the Topology archetype.
 *
 * The archetype's shape is "tier bands (core → access), labelled links,
 * details panel". The bands are derived from the layout rather than declared:
 * the hierarchical layout ranks devices by their distance from a root in the
 * link graph, which is the same thing an operator means by core / distribution
 * / access. Nothing in the daemon reports a tier, so deriving one from the
 * ranking is the only honest source — and where the ranking says nothing, the
 * bands say nothing.
 */

import { useMemo } from 'react';
import type { LayoutMode } from './layout';
import type { DeviceNode } from './types';

/** One horizontal band behind a rank of devices. */
export interface Tier {
  /** Operator-facing name for the band. */
  label: 'Core' | 'Distribution' | 'Access';
  /** Canvas y of the band's top edge. */
  y: number;
  /** Band height in canvas units. */
  height: number;
  /** How many devices sit in this band. */
  deviceCount: number;
}

/**
 * RANK_EPSILON is how far two devices' y positions may drift and still count
 * as one rank. Dagre centres nodes on a rank and ReactFlow positions from the
 * top-left, so siblings land a few pixels apart; the gap between real ranks is
 * ranksep (280) plus a node height, an order of magnitude larger.
 */
const RANK_EPSILON = 120;

/** Vertical padding so a band reads as a band and not as a tight box. */
const BAND_PADDING = 60;

/** Approximate node height, so a band encloses the cards it sits behind. */
const NODE_HEIGHT = 180;

/**
 * deriveTiers groups laid-out nodes into ranked bands, top to bottom.
 *
 * A graph whose devices all share one rank has no hierarchy to draw, and a
 * single band labelled "Core" would assert a tier the layout never derived —
 * so that case returns nothing, as does an empty graph.
 */
export function deriveTiers(nodes: DeviceNode[]): Tier[] {
  if (nodes.length === 0) {
    return [];
  }

  const ranks = groupByRank(nodes);
  if (ranks.length < 2) {
    return [];
  }

  return ranks.map((rank, index) => ({
    label: labelFor(index, ranks.length),
    y: Math.min(...rank) - BAND_PADDING,
    height: Math.max(...rank) - Math.min(...rank) + NODE_HEIGHT + BAND_PADDING * 2,
    deviceCount: rank.length,
  }));
}

/** groupByRank buckets node y positions into ascending ranks. */
function groupByRank(nodes: DeviceNode[]): number[][] {
  const [firstY, ...restY] = nodes.map((n) => n.position.y).sort((a, b) => a - b);

  // No nodes means no ranks. The previous form seeded the first bucket with
  // sorted[0] unconditionally, so an empty graph produced one phantom tier
  // holding undefined — which the caller then counted as a device.
  if (firstY === undefined) {
    return [];
  }

  // The anchor is carried rather than re-read from the bucket: it is the value
  // every member of the current rank is compared against, which is a fact
  // about the rank, not about the array's first slot.
  let anchor = firstY;
  let current = [firstY];
  const ranks: number[][] = [current];

  for (const y of restY) {
    if (y - anchor <= RANK_EPSILON) {
      current.push(y);
      continue;
    }
    anchor = y;
    current = [y];
    ranks.push(current);
  }
  return ranks;
}

/** labelFor names a band by its position: first is core, last is access. */
function labelFor(index: number, total: number): Tier['label'] {
  if (index === 0) {
    return 'Core';
  }
  if (index === total - 1) {
    return 'Access';
  }
  return 'Distribution';
}

/** DeviceNode's max width — bands must clear the widest card. */
const NODE_WIDTH = 280;

/** Breathing room between a band's edge and the cards inside it. */
const BAND_MARGIN = 80;

/** Where the bands start and how wide they run. */
export interface TierExtent {
  left: number;
  width: number;
}

/**
 * useTierBands derives the bands and their horizontal extent from the nodes
 * currently on canvas.
 *
 * Bands are only honest under the hierarchical layout. Grid positions devices
 * by index, so its rows are pagination, not a core→access hierarchy; labelling
 * them would assert a structure the layout never derived.
 */
export function useTierBands(
  nodes: DeviceNode[],
  layoutMode: LayoutMode,
): { tiers: Tier[]; extent: TierExtent } {
  const tiers = useMemo(
    () => (layoutMode === 'hierarchical' ? deriveTiers(nodes) : []),
    [nodes, layoutMode],
  );

  const extent = useMemo(() => {
    if (nodes.length === 0) {
      return { left: 0, width: 0 };
    }
    const xs = nodes.map((n) => n.position.x);
    const left = Math.min(...xs) - BAND_MARGIN;
    return { left, width: Math.max(...xs) + NODE_WIDTH + BAND_MARGIN - left };
  }, [nodes]);

  return { tiers, extent };
}
