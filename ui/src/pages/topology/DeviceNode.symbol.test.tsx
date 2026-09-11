/**
 * DeviceNode.symbol.test.tsx — locks the symbol-and-label node shape (#2056).
 *
 * The node used to be a 280x180 card carrying the name, the type, the first
 * IP and up to three protocol chips. It is now a device symbol with the name
 * beneath it, so the facts that left the canvas have to be reachable from the
 * node itself — the details panel is a second click and a second component,
 * and would not catch this regressing.
 */
import { render, within } from '@testing-library/react';
import { ReactFlowProvider } from '@xyflow/react';
import { I18nextProvider } from 'react-i18next';
import { describe, expect, it } from 'vitest';
import i18n from '../../i18n';
import { DeviceNode } from './DeviceNode';
import type { DeviceNodeData } from './types';

function renderNode(data: Partial<DeviceNodeData> = {}) {
  const { container } = render(
    <I18nextProvider i18n={i18n}>
      <ReactFlowProvider>
        <DeviceNode
          data={
            {
              label: 'edge-router-01',
              type: 'router',
              ips: ['10.0.0.1', '10.0.1.1'],
              protocols: ['BGP', 'OSPF', 'SNMP'],
              ...data,
            } as DeviceNodeData
          }
        />
      </ReactFlowProvider>
    </I18nextProvider>,
    { container: document.body.appendChild(document.createElement('div')) },
  );
  return within(container).getByTestId('topology-device-node');
}

describe('DeviceNode — symbol and label', () => {
  it('draws the device symbol as inline SVG so the PNG export can rasterise it', () => {
    const svg = renderNode().querySelector('svg');
    expect(svg).not.toBeNull();
    expect(svg?.querySelector('path')).not.toBeNull();
  });

  it('keeps the device name on the canvas', () => {
    expect(renderNode().textContent).toContain('edge-router-01');
  });

  it('drops the IP and protocol detail the card used to carry', () => {
    const node = renderNode();
    expect(node.textContent).not.toContain('10.0.0.1');
    expect(node.textContent).not.toContain('BGP');
  });

  it('keeps IPs and protocols reachable from the node, in its accessible name', () => {
    const label = renderNode().getAttribute('aria-label') ?? '';
    expect(label).toContain('10.0.0.1');
    expect(label).toContain('10.0.1.1');
    expect(label).toContain('BGP');
    expect(label).toContain('OSPF');
    expect(label).toContain('SNMP');
  });

  it('draws a different symbol for a different device type', () => {
    const router = renderNode({ type: 'router' }).querySelector('path')?.getAttribute('d');
    const ap = renderNode({ type: 'access_point' }).querySelector('path')?.getAttribute('d');
    expect(router).toBeTruthy();
    expect(ap).not.toBe(router);
  });
});
