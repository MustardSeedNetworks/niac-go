/**
 * DeviceNode.icon.test.tsx — locks the device-type icon/colour fix (#2052).
 *
 * The canvas and the legend read two different maps, so an `access_point`
 * drew the `unknown` glyph and the grey `unknown` colour on the canvas while
 * the legend, which passed the hyphenated alias, drew a Wifi glyph and the
 * purple AP colour for the same device. These assertions are on the rendered
 * node, not on the map, because the map agreeing with itself is not the thing
 * that broke.
 */
import { render } from '@testing-library/react';
import { ReactFlowProvider } from '@xyflow/react';
import { I18nextProvider } from 'react-i18next';
import { describe, expect, it } from 'vitest';
import i18n from '../../i18n';
import { DeviceNode } from './DeviceNode';
import type { DeviceNodeData } from './types';

/**
 * iconTint renders one node and returns the colour it tints the icon well
 * with. Each call gets its own container so two types can be compared inside
 * a single test without the queries seeing both nodes at once.
 */
function iconTint(type: string): string {
  const { container } = render(
    <I18nextProvider i18n={i18n}>
      <ReactFlowProvider>
        <DeviceNode data={{ label: 'wifi-ap-01', type } as DeviceNodeData} />
      </ReactFlowProvider>
    </I18nextProvider>,
    { container: document.body.appendChild(document.createElement('div')) },
  );
  const well = container.querySelector<HTMLElement>('[style*="color-mix"]');
  return well?.style.backgroundColor ?? '';
}

describe('DeviceNode device-type styling', () => {
  it('gives an access point the AP colour, not the unknown fallback', () => {
    expect(iconTint('access_point')).toContain('--color-device-ap');
  });

  it('styles an aliased access point the same as the canonical type', () => {
    expect(iconTint('ap')).toBe(iconTint('access_point'));
  });

  it('falls back to the unknown colour for a type it has no mapping for', () => {
    expect(iconTint('toaster')).toContain('--color-device-unknown');
  });
});
