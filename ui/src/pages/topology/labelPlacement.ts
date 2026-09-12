/**
 * Where every edge label goes, decided once with the whole graph in view.
 *
 * Each edge used to place its own labels. An edge knows its endpoints and
 * nothing else, so it could not tell that its interface name was landing on a
 * device it has no connection to, or on a label belonging to the edge beside
 * it — which is what left a dense pack with hundreds of overlapping labels
 * (#2104).
 *
 * This walks every candidate rectangle against the ones already accepted and
 * against the devices, and a label that cannot be placed clear is not drawn.
 * Leaving one out is the honest outcome: a label under another label conveys
 * nothing, and the topology header has a toggle for reading the diagram
 * without them.
 */

import { NODE_HEIGHT, NODE_WIDTH } from './layout';
import type { DeviceNode, LinkEdge, LinkEdgeData } from './types';

/** A label box in canvas units, centred on (x, y). */
interface Box {
  x: number;
  y: number;
  width: number;
  height: number;
}

/** Where one edge's labels ended up. A missing entry did not fit. */
export interface EdgeLabelPlacement {
  source?: { x: number; y: number };
  middle?: { x: number; y: number };
  target?: { x: number; y: number };
}

/**
 * Label geometry in canvas units. Labels scale with the canvas, so a label
 * covers the same span at every zoom and none of this needs a zoom term.
 */
const LABEL_HEIGHT = 21;
const LABEL_ROW = LABEL_HEIGHT + 8;
const LABEL_MIN_LENGTH = 48;

/** Clear space kept around a label, so "not overlapping" also reads as apart. */
const LABEL_MARGIN = 6;

/** How far the first label sits from its device, outside the node's footprint. */
const LABEL_START = Math.hypot(NODE_WIDTH, NODE_HEIGHT) / 2 + LABEL_HEIGHT;

/** How many rows out a label will try before it is left out. */
const MAX_ROWS = 6;

/**
 * Canvas units a label box occupies.
 *
 * The floor matters: for short text like "1M" the padding and border dominate
 * the width, and estimating from the character count alone put interface
 * labels on top of the line's own speed label.
 */
function labelLength(text: string): number {
  return Math.max(text.length * 5.6 + 14, LABEL_MIN_LENGTH);
}

function overlaps(a: Box, b: Box): boolean {
  return (
    Math.abs(a.x - b.x) * 2 < a.width + b.width && Math.abs(a.y - b.y) * 2 < a.height + b.height
  );
}

function labelBox(x: number, y: number, text: string): Box {
  return {
    x,
    y,
    width: labelLength(text) + LABEL_MARGIN * 2,
    height: LABEL_HEIGHT + LABEL_MARGIN * 2,
  };
}

/** The centre of a device's box, which is what edges run between. */
function centreOf(node: DeviceNode): Box {
  return {
    x: node.position.x + NODE_WIDTH / 2,
    y: node.position.y + NODE_HEIGHT / 2,
    width: NODE_WIDTH,
    height: NODE_HEIGHT,
  };
}

/**
 * placeLabels decides each edge's label positions against every device and
 * every label already placed.
 *
 * Edges are taken shortest first. A short edge has the least room to move its
 * labels, so giving it the space it has leaves the long edges — which have
 * rows to spare — to work around it.
 */
export function placeLabels(
  nodes: DeviceNode[],
  edges: LinkEdge[],
): Map<string, EdgeLabelPlacement> {
  const byId = new Map(nodes.map((node) => [node.id, node]));
  const taken: Box[] = nodes.map(centreOf);
  const placements = new Map<string, EdgeLabelPlacement>();

  const withLength = edges
    .map((edge) => {
      const source = byId.get(edge.source);
      const target = byId.get(edge.target);
      if (!source || !target) {
        return undefined;
      }
      const from = centreOf(source);
      const to = centreOf(target);
      return { edge, from, to, length: Math.hypot(to.x - from.x, to.y - from.y) };
    })
    .filter((entry) => entry !== undefined)
    .sort((left, right) => left.length - right.length);

  for (const { edge, from, to, length } of withLength) {
    if (length === 0) {
      continue;
    }
    const data = edge.data ?? {};
    const placement: EdgeLabelPlacement = {};
    const along = (distance: number): { x: number; y: number } => ({
      x: from.x + ((to.x - from.x) * distance) / length,
      y: from.y + ((to.y - from.y) * distance) / length,
    });

    /** Steps outward a row at a time until the box is clear, or gives up. */
    const tryPlace = (text: string, base: number, outward: 1 | -1): Box | undefined => {
      for (let row = 0; row < MAX_ROWS; row++) {
        const distance = base + outward * row * LABEL_ROW;
        if (distance <= 0 || distance >= length) {
          return undefined;
        }
        const point = along(distance);
        const box = labelBox(point.x, point.y, text);
        if (!taken.some((other) => overlaps(box, other))) {
          return box;
        }
      }
      return undefined;
    };

    const middleText = typeof data.label === 'string' ? data.label : '';
    const sourceText = typeof data.sourceInterface === 'string' ? data.sourceInterface : '';
    const targetText = typeof data.targetInterface === 'string' ? data.targetInterface : '';

    // The middle label goes first: it has one candidate spot, the midpoint,
    // while the end labels can step along the edge to get out of its way.
    if (middleText) {
      const box = tryPlace(middleText, length / 2, 1);
      if (box) {
        taken.push(box);
        placement.middle = { x: box.x, y: box.y };
      }
    }
    if (sourceText) {
      const box = tryPlace(sourceText, LABEL_START, 1);
      if (box) {
        taken.push(box);
        placement.source = { x: box.x, y: box.y };
      }
    }
    if (targetText) {
      const box = tryPlace(targetText, length - LABEL_START, -1);
      if (box) {
        taken.push(box);
        placement.target = { x: box.x, y: box.y };
      }
    }
    placements.set(edge.id, placement);
  }

  return placements;
}

/** What the page knows about an edge that placement does not. */
interface EdgePresentation {
  showLabels: boolean;
  focusOpacity: (source: string, target: string) => number;
  hoveredEdgeId: string | null;
}

/**
 * placedEdges runs the placement pass over the laid-out graph and hands each
 * edge its own result, alongside the presentation state the page owns.
 *
 * Assembling this on the page meant the page had to know that placement is a
 * whole-graph pass and that an edge reads its result out of `data`. It only
 * needs to know that labels are placed.
 */
export function placedEdges(
  nodes: DeviceNode[],
  edges: LinkEdge[],
  presentation: EdgePresentation,
): LinkEdge[] {
  const placements = placeLabels(nodes, edges);
  return edges.map((edge) => ({
    ...edge,
    data: {
      ...(edge.data as LinkEdgeData),
      showLabels: presentation.showLabels,
      focusOpacity: presentation.focusOpacity(edge.source, edge.target),
      hovered: presentation.hoveredEdgeId === edge.id,
      labelPlacement: placements.get(edge.id),
    },
  }));
}
