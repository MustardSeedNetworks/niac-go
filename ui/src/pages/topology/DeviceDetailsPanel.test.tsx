import { screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { ObservedClient, SimulationStatus } from '../../api/types';
import { renderWithResources as render } from '../../test/renderWithResources';
import '../../i18n';
import { DeviceDetailsPanel } from './DeviceDetailsPanel';

const context = vi.hoisted(() => ({
  sessionId: null as string | null,
  selectedSession: null as SimulationStatus | null,
}));

vi.mock('../../contexts/AppContext', () => ({
  useAppContext: () => context,
}));
vi.mock('../../contexts/ScopeContext', () => ({
  useActionPermission: () => ({ disabled: false }),
}));
vi.mock('../../api/client', () => ({
  fetchSessionClients: (): Promise<ObservedClient[]> => Promise.resolve([]),
  pinSessionClient: vi.fn(),
}));

const session = (attachment: string): SimulationStatus => ({
  running: true,
  sessionId: 'hospital',
  deviceCount: 12,
  uptimeSeconds: 42,
  fabric: {
    topology: {
      binding: {
        attachment,
        interface: 'eth1',
        mode: 'direct',
        network: 'clinical',
        wireTagged: false,
      },
      networks: [],
      interfaces: [],
      routes: [],
      dhcpScopes: [],
      attachments: [
        {
          name: 'cyberscope',
          device: 'MED-ACC-SW01',
          network: 'clinical',
          ports: [
            {
              device: 'MED-ACC-SW01',
              interface: 'GigabitEthernet1/0/45',
              vlan: 210,
              network: 'clinical',
            },
          ],
        },
      ],
    },
    forwarded: 0,
    drops: 0,
    received: 0,
    transmitted: 0,
  },
});

const show = (name: string) =>
  render(
    <DeviceDetailsPanel
      device={{ name, type: 'switch', ips: [], protocols: [] }}
      onClose={() => {}}
    />,
  );

describe('DeviceDetailsPanel tester ports', () => {
  beforeEach(() => {
    context.sessionId = 'hospital';
    context.selectedSession = session('cyberscope');
  });

  it("shows the pool on a device that carries the session's pool ports", async () => {
    show('MED-ACC-SW01');
    expect(await screen.findByTestId('device-attachment-pool')).toBeInTheDocument();
  });

  it('shows no pool on a device outside it', () => {
    show('MED-DIST-SW01');
    expect(screen.queryByTestId('device-attachment-pool')).not.toBeInTheDocument();
  });

  it('shows no pool when the session binds another attachment', () => {
    context.selectedSession = session('lab-network');
    show('MED-ACC-SW01');
    expect(screen.queryByTestId('device-attachment-pool')).not.toBeInTheDocument();
  });
});
