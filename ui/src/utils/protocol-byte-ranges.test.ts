import { describe, expect, it } from 'vitest';
import { buildProtocolLayers, computeHeaderBoundary } from './protocol-layers';

describe('decoder-provided byte ranges', () => {
  it('includes decoded options and extension header boundaries', () => {
    const layers = buildProtocolLayers(
      { tcp: { srcPort: 80 } },
      {
        protocol: 'TCP',
        byteRanges: [
          { layer: 'IPv6HopByHop', field: '', start: 54, end: 62 },
          { layer: 'tcp', field: 'Options', start: 82, end: 86 },
          { layer: 'tcp', field: '', start: 62, end: 86 },
        ],
      },
    );
    expect(layers.find((layer) => layer.name === 'IPv6HopByHop')?.byteEnd).toBe(62);
    expect(layers.flatMap((layer) => layer.fields)).toContainEqual({
      name: 'Options',
      value: '4 bytes',
      byteStart: 82,
      byteEnd: 86,
    });
    expect(computeHeaderBoundary(layers)).toBe(86);
  });
  it('highlights TCP at 38 for a tagged frame', () => {
    const layers = buildProtocolLayers(
      { tcp: { srcPort: 80 } },
      {
        protocol: 'TCP',
        byteRanges: [
          { layer: 'tcp', field: 'Source Port', start: 38, end: 40 },
          { layer: 'tcp', field: '', start: 38, end: 58 },
        ],
      },
    );
    expect(layers.find((layer) => layer.name.includes('(TCP)'))?.fields[0]).toMatchObject({
      byteStart: 38,
      byteEnd: 40,
    });
    expect(computeHeaderBoundary(layers)).toBe(58);
  });

  it('does not invent byte positions when the decoder supplied none', () => {
    const layers = buildProtocolLayers(
      { ipv6: { src: '::1', dst: '::2' }, tcp: { srcPort: 80 } },
      { protocol: 'TCP' },
    );
    expect(
      layers.flatMap((layer) => layer.fields).every((field) => field.byteStart === undefined),
    ).toBe(true);
    expect(computeHeaderBoundary(layers)).toBe(0);
  });
});
