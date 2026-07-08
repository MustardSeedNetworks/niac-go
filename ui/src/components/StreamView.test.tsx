/**
 * StreamView.test.tsx — direction glyphs.
 *
 * Regression coverage for direction being conveyed by color alone, which
 * fails for colorblind users. Each stream segment must render a text glyph
 * (independent of color) indicating client-to-server vs server-to-client.
 */
import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import '../i18n';
import type { PcapPacket } from '../api/types';
import { StreamView } from './StreamView';

function makePacket(overrides: Partial<PcapPacket>): PcapPacket {
  return {
    id: '1',
    number: 1,
    timestamp: new Date().toISOString(),
    protocol: 'TCP',
    sourceIp: '10.0.0.1',
    destIp: '10.0.0.2',
    sourcePort: 1234,
    destPort: 80,
    length: 64,
    info: 'packet',
    rawData: '68656c6c6f',
    ...overrides,
  };
}

describe('StreamView — direction glyphs', () => {
  it('renders a client-to-server glyph for segments from the client endpoint', () => {
    render(
      <StreamView
        packets={[makePacket({ sourceIp: '10.0.0.1', sourcePort: 1234 })]}
        clientEndpoint="10.0.0.1:1234"
        onClose={() => undefined}
      />,
    );

    const glyphs = screen.getAllByTestId('stream-direction-glyph');
    expect(glyphs).toHaveLength(1);
    expect(glyphs[0]).toHaveTextContent('→');
    expect(glyphs[0]).toHaveAttribute('aria-label', 'Client to server');
  });

  it('renders a server-to-client glyph for segments from a different endpoint', () => {
    render(
      <StreamView
        packets={[makePacket({ sourceIp: '10.0.0.2', sourcePort: 80 })]}
        clientEndpoint="10.0.0.1:1234"
        onClose={() => undefined}
      />,
    );

    const glyphs = screen.getAllByTestId('stream-direction-glyph');
    expect(glyphs).toHaveLength(1);
    expect(glyphs[0]).toHaveTextContent('←');
    expect(glyphs[0]).toHaveAttribute('aria-label', 'Server to client');
  });
});
