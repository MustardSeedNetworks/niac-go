/**
 * ProtocolTree.test.tsx — expand-all / collapse-all controls.
 *
 * Expanded state is per-layer by default (buildProtocolLayers doesn't mark
 * any layer collapsed for a TCP packet), so these tests toggle individual
 * layers closed first to prove the buttons actually drive every layer at
 * once rather than being a no-op.
 */
import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { Packet } from './PacketList';
import { ProtocolTree } from './ProtocolTree';

function makePacket(): Packet {
  return {
    id: '1',
    timestamp: new Date().toISOString(),
    protocol: 'TCP',
    sourceIp: '10.0.0.1',
    destIp: '10.0.0.2',
    sourcePort: 1234,
    destPort: 80,
    size: 64,
    summary: 'packet 1',
    rawData: '00112233445566778899aabbccddeeff0011223344',
  };
}

function layerToggleButtons(container: HTMLElement): HTMLButtonElement[] {
  return Array.from(container.querySelectorAll('.text-brand-accent')).map(
    (span) => span.closest('button') as HTMLButtonElement,
  );
}

describe('ProtocolTree — expand-all / collapse-all', () => {
  it('collapses every layer when Collapse all is clicked', () => {
    const { container } = render(<ProtocolTree packet={makePacket()} />);

    const layerButtons = layerToggleButtons(container);
    expect(layerButtons.length).toBeGreaterThan(1);
    expect(container.querySelectorAll('.border-l.border-surface-border').length).toBe(
      layerButtons.length,
    );

    fireEvent.click(screen.getByTestId('protocol-tree-collapse-all'));

    expect(container.querySelectorAll('.border-l.border-surface-border').length).toBe(0);
  });

  it('expands every layer when Expand all is clicked after some are collapsed', () => {
    const { container } = render(<ProtocolTree packet={makePacket()} />);

    const layerButtons = layerToggleButtons(container);
    fireEvent.click(layerButtons[0]);
    fireEvent.click(layerButtons[1]);
    expect(container.querySelectorAll('.border-l.border-surface-border').length).toBe(
      layerButtons.length - 2,
    );

    fireEvent.click(screen.getByTestId('protocol-tree-expand-all'));

    expect(container.querySelectorAll('.border-l.border-surface-border').length).toBe(
      layerButtons.length,
    );
  });
});
