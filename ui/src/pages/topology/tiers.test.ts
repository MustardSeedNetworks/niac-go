import { describe, expect, it } from 'vitest';
import { required } from '../../test/required';
import { deriveTiers } from './tiers';
import type { DeviceNode } from './types';

/** node builds the minimum DeviceNode deriveTiers reads: id and y. */
function node(id: string, y: number): DeviceNode {
  return {
    id,
    type: 'device',
    position: { x: 0, y },
    data: { label: id, type: 'switch' },
  };
}

describe('deriveTiers', () => {
  it('names the top band Core and the bottom band Access', () => {
    const tiers = deriveTiers([node('core-1', 40), node('acc-1', 460), node('acc-2', 460)]);

    expect(tiers.map((t) => t.label)).toEqual(['Core', 'Access']);
    expect(tiers[0]?.deviceCount).toBe(1);
    expect(tiers[1]?.deviceCount).toBe(2);
  });

  it('labels every band between the first and last Distribution', () => {
    const tiers = deriveTiers([
      node('core-1', 40),
      node('dist-1', 460),
      node('dist-2', 880),
      node('acc-1', 1300),
    ]);

    expect(tiers.map((t) => t.label)).toEqual(['Core', 'Distribution', 'Distribution', 'Access']);
  });

  // A flat graph has no hierarchy to show. Drawing one band labelled "Core"
  // over every device would assert a tier the layout never derived.
  it('returns no bands when every device shares one rank', () => {
    expect(deriveTiers([node('a', 40), node('b', 40), node('c', 40)])).toEqual([]);
  });

  it('returns no bands for an empty graph', () => {
    expect(deriveTiers([])).toEqual([]);
  });

  // Dagre centres nodes on a rank but ReactFlow positions are top-left, so
  // sibling y values drift by a pixel or two. Bands must not split on that.
  it('groups ranks that differ by less than a node height', () => {
    const tiers = deriveTiers([node('a', 40), node('b', 43), node('c', 460)]);

    expect(tiers).toHaveLength(2);
    expect(tiers[0]?.deviceCount).toBe(2);
  });

  it('spans each band across the devices it contains', () => {
    const core = required(
      deriveTiers([node('core-1', 40), node('acc-1', 460)])[0],
      'the core band',
    );

    expect(core.y).toBeLessThanOrEqual(40);
    expect(core.height).toBeGreaterThan(0);
  });
});
