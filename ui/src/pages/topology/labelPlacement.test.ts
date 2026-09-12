/**
 * placeLabels decides every edge label's position with the whole graph in view.
 *
 * Each edge used to place its own, which meant it could not see that its
 * interface name was landing on a device it has no connection to, or on the
 * label of the edge beside it. Measured on the generated packs before this:
 * 41 collisions on hospital, 552 on enterprise-scale. After: none (#2104).
 */

import { describe, expect, it } from 'vitest';
import { placeLabels } from './labelPlacement';
import type { DeviceNode, LinkEdge } from './types';

const NODE_W = 112;
const NODE_H = 96;

function node(id: string, x: number, y: number): DeviceNode {
  return {
    id,
    type: 'device',
    position: { x, y },
    data: { label: id, type: 'switch' },
  };
}

function edge(id: string, source: string, target: string, data = {}): LinkEdge {
  return {
    id,
    source,
    target,
    data: { sourceInterface: 'Gi1/0/1', targetInterface: 'Gi0/0/0', label: '1G', ...data },
  };
}

/** Every box the pass produced, plus the devices, as rectangles. */
function boxes(nodes: DeviceNode[], edges: LinkEdge[]): { x: number; y: number }[] {
  const placements = placeLabels(nodes, edges);
  return [...placements.values()].flatMap((placement) =>
    [placement.source, placement.middle, placement.target].filter((point) => point !== undefined),
  );
}

describe('placeLabels', () => {
  it('keeps every label it places clear of the devices', () => {
    const nodes = [node('sw-01', 0, 0), node('sw-02', 0, 600), node('sw-03', 400, 600)];
    const edges = [edge('e1', 'sw-01', 'sw-02'), edge('e2', 'sw-01', 'sw-03')];

    for (const point of boxes(nodes, edges)) {
      for (const device of nodes) {
        const clearX = Math.abs(point.x - (device.position.x + NODE_W / 2)) >= NODE_W / 2;
        const clearY = Math.abs(point.y - (device.position.y + NODE_H / 2)) >= NODE_H / 2;
        expect(clearX || clearY).toBe(true);
      }
    }
  });

  it('keeps the labels it places clear of each other', () => {
    const nodes = [
      node('sw-01', 0, 0),
      ...Array.from({ length: 6 }, (_, i) => node(`ap-${i}`, i * 160, 700)),
    ];
    const edges = Array.from({ length: 6 }, (_, i) => edge(`e${i}`, 'sw-01', `ap-${i}`));
    const points = boxes(nodes, edges);

    for (let i = 0; i < points.length; i++) {
      for (let j = i + 1; j < points.length; j++) {
        const a = points[i];
        const b = points[j];
        const apart =
          Math.abs((a?.x ?? 0) - (b?.x ?? 0)) >= 48 || Math.abs((a?.y ?? 0) - (b?.y ?? 0)) >= 21;
        expect(apart).toBe(true);
      }
    }
  });

  // Leaving a label out is the honest outcome when there is nowhere clear for
  // it: a label under another label conveys nothing.
  it('omits what it cannot place rather than stacking it', () => {
    const nodes = [node('a', 0, 0), node('b', 0, 130)];
    const placement = placeLabels(nodes, [edge('e1', 'a', 'b')]).get('e1');

    expect(placement).toBeDefined();
    expect(placement?.source).toBeUndefined();
    expect(placement?.target).toBeUndefined();
  });

  it('still places labels when there is room', () => {
    const nodes = [node('a', 0, 0), node('b', 0, 800)];
    const placement = placeLabels(nodes, [edge('e1', 'a', 'b')]).get('e1');

    expect(placement?.source).toBeDefined();
    expect(placement?.middle).toBeDefined();
    expect(placement?.target).toBeDefined();
  });

  it('ignores an edge whose devices are not on the canvas', () => {
    const placements = placeLabels([node('a', 0, 0)], [edge('e1', 'a', 'missing')]);

    expect(placements.has('e1')).toBe(false);
  });
});
