import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { type ReactNode, useEffect, useState } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { ScenarioDraft } from '../../api/library-client';
import { required } from '../../test/required';
import '../../i18n';
import { DraftTopologyComposer } from './DraftTopologyComposer';

const mutateScenarioDraftTopology = vi.hoisted(() => vi.fn());
const fitView = vi.hoisted(() => vi.fn());
const fetchScenarioProfiles = vi.hoisted(() => vi.fn());
const profiles = [
  {
    role: 'access',
    deviceType: 'switch',
    vendor: 'cisco',
    model: 'C9300-48P',
    platform: 'ios-xe',
    software: '17.15.3',
    sysObjectId: '1.3.6.1.4.1.9.1.2494',
  },
  {
    role: 'office-access',
    deviceType: 'switch',
    vendor: 'cisco',
    model: 'Captured Catalyst',
    platform: 'Cisco IOS XE',
    software: '17.15',
    sysObjectId: '1.3.6.1.4.1.9.1.2238',
    walkName: 'captured/office.walk',
    interfaceCount: 65,
    interfaces: [
      {
        name: 'GigabitEthernet1/0/48',
        type: 'ethernet',
        mtu: 1500,
        speed: 10_000_000_000,
        adminStatus: 'up',
        operStatus: 'down',
      },
    ],
    source: 'captured',
  },
];

vi.mock('../../api/library-client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../api/library-client')>();
  return { ...actual, mutateScenarioDraftTopology };
});

vi.mock('../../api/scenario-client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../api/scenario-client')>();
  return {
    ...actual,
    fetchScenarioProfiles,
  };
});

vi.mock('@xyflow/react', () => ({
  Background: () => null,
  BackgroundVariant: { Dots: 'dots' },
  MarkerType: { ArrowClosed: 'arrowclosed' },
  ReactFlow: ({
    children,
    edges,
    onEdgeClick,
    onInit,
  }: {
    children: ReactNode;
    edges: Array<{ source: string; target: string; data?: Record<string, unknown> }>;
    onEdgeClick: (event: unknown, edge: (typeof edges)[number]) => void;
    onInit?: (instance: { fitView: typeof fitView }) => void;
  }) => {
    useEffect(() => onInit?.({ fitView }), [onInit]);
    return (
      <div>
        {edges[0] && (
          <button
            type="button"
            onClick={() => onEdgeClick({}, required(edges[0], 'the first edge'))}
          >
            Edit first link
          </button>
        )}
        {children}
      </div>
    );
  },
  useEdgesState: <T,>(initial: T[]) => {
    const [edges, setEdges] = useState(initial);
    return [edges, setEdges, vi.fn()] as const;
  },
  useNodesState: <T,>(initial: T[]) => {
    const [nodes, setNodes] = useState(initial);
    return [nodes, setNodes, vi.fn()] as const;
  },
}));

const draft: ScenarioDraft = {
  name: 'campus-draft',
  content: 'devices: []\n',
  format: 'yaml',
  revision: 'revision-1',
  modifiedAt: '2026-07-29T12:00:00Z',
  sizeBytes: 12,
};

describe('DraftTopologyComposer', () => {
  beforeEach(() => {
    mutateScenarioDraftTopology.mockReset();
    fitView.mockReset();
    fetchScenarioProfiles.mockReset().mockResolvedValue(profiles);
  });

  it('surfaces a profile-loading failure and recovers on retry', async () => {
    const user = userEvent.setup();
    fetchScenarioProfiles
      .mockRejectedValueOnce(new Error('catalog unavailable'))
      .mockResolvedValueOnce(profiles);

    render(<DraftTopologyComposer draft={draft} onDraftUpdate={vi.fn()} onBusyChange={vi.fn()} />);

    expect(await screen.findByText('Device profiles could not be loaded.')).toBeVisible();
    expect(screen.getByRole('button', { name: 'Add device' })).toBeDisabled();
    await user.click(screen.getByRole('button', { name: 'Retry' }));

    await waitFor(() => expect(fetchScenarioProfiles).toHaveBeenCalledTimes(2));
    await waitFor(() => expect(screen.getByRole('button', { name: 'Add device' })).toBeEnabled());
    expect(screen.queryByText('Device profiles could not be loaded.')).not.toBeInTheDocument();
  });

  it('adds a profile-backed device through a revision-safe topology mutation', async () => {
    const user = userEvent.setup();
    const updated = { ...draft, revision: 'revision-2' };
    mutateScenarioDraftTopology.mockResolvedValue(updated);
    const onDraftUpdate = vi.fn();
    const onBusyChange = vi.fn();
    render(
      <DraftTopologyComposer
        draft={draft}
        onDraftUpdate={onDraftUpdate}
        onBusyChange={onBusyChange}
      />,
    );

    await user.click(await screen.findByRole('button', { name: 'Add device' }));
    const dialog = screen.getByRole('dialog');
    await user.type(within(dialog).getByLabelText('Device name'), 'COS-ACCESS-SW01');
    await user.click(within(dialog).getByRole('button', { name: 'Add device' }));

    await waitFor(() => expect(mutateScenarioDraftTopology).toHaveBeenCalledTimes(1));
    expect(mutateScenarioDraftTopology).toHaveBeenCalledWith(
      'campus-draft',
      'revision-1',
      expect.objectContaining({
        operation: 'add_device',
        device: expect.objectContaining({
          name: 'COS-ACCESS-SW01',
          type: 'switch',
          vendor: 'cisco',
          sysObjectId: '1.3.6.1.4.1.9.1.2494',
          interfaces: expect.arrayContaining([
            expect.objectContaining({ name: 'Ethernet1/1', mtu: 1500, duplex: 'full' }),
          ]),
        }),
      }),
    );
    expect(onDraftUpdate).toHaveBeenCalledWith(updated);
    expect(onBusyChange.mock.calls).toEqual([[true], [false]]);
  });

  it('uses the captured interface inventory when adding a reviewed profile', async () => {
    const user = userEvent.setup();
    mutateScenarioDraftTopology.mockResolvedValue({ ...draft, revision: 'revision-2' });
    render(<DraftTopologyComposer draft={draft} onDraftUpdate={vi.fn()} onBusyChange={vi.fn()} />);

    await user.click(await screen.findByRole('button', { name: 'Add device' }));
    const dialog = screen.getByRole('dialog');
    await user.type(within(dialog).getByLabelText('Device name'), 'COS-CAPTURED-SW01');
    await user.selectOptions(within(dialog).getByLabelText('Device profile'), 'office-access');
    expect(within(dialog).getByLabelText('Ethernet port count')).toBeDisabled();
    await user.click(within(dialog).getByRole('button', { name: 'Add device' }));

    await waitFor(() => expect(mutateScenarioDraftTopology).toHaveBeenCalledTimes(1));
    expect(mutateScenarioDraftTopology).toHaveBeenCalledWith(
      'campus-draft',
      'revision-1',
      expect.objectContaining({
        device: expect.objectContaining({
          profileRole: 'office-access',
          interfaces: [
            expect.objectContaining({
              name: 'GigabitEthernet1/0/48',
              speed: 10_000,
              mtu: 1500,
              operStatus: 'down',
            }),
          ],
        }),
      }),
    );
  });

  it('preserves FDB-only behavior when editing an authored link', async () => {
    const user = userEvent.setup();
    const linkedDraft = {
      ...draft,
      content: `
devices:
  - name: core-1
    type: switch
    interfaces:
      - { name: Ethernet1/1, type: ethernet, speed: 10000 }
    trunk_ports:
      - { interface: Ethernet1/1, remote_device: dist-1, remote_interface: Ethernet1/49, vlans: [200], fdb_only: true }
  - name: dist-1
    type: switch
    interfaces:
      - { name: Ethernet1/49, type: ethernet, speed: 10000 }
    trunk_ports:
      - { interface: Ethernet1/49, remote_device: core-1, remote_interface: Ethernet1/1, vlans: [200], fdb_only: true }
`,
    };
    mutateScenarioDraftTopology.mockResolvedValue({ ...linkedDraft, revision: 'revision-2' });
    render(
      <DraftTopologyComposer draft={linkedDraft} onDraftUpdate={vi.fn()} onBusyChange={vi.fn()} />,
    );

    await user.click(await screen.findByRole('button', { name: 'Edit first link' }));
    const dialog = screen.getByRole('dialog');
    expect(within(dialog).getByRole('checkbox')).toBeChecked();
    await user.click(within(dialog).getByRole('button', { name: 'Save link' }));

    await waitFor(() => expect(mutateScenarioDraftTopology).toHaveBeenCalledTimes(1));
    expect(mutateScenarioDraftTopology).toHaveBeenCalledWith(
      'campus-draft',
      'revision-1',
      expect.objectContaining({
        operation: 'update_link',
        link: expect.objectContaining({
          properties: { vlans: [200], nativeVlan: 0, fdbOnly: true },
        }),
      }),
    );
  });
  const switchPair = (portProps: string) => ({
    ...draft,
    content: `
devices:
  - name: core-1
    type: switch
    interfaces:
      - { name: Ethernet1/1, type: ethernet, speed: 10000 }
    trunk_ports:
      - { interface: Ethernet1/1, remote_device: dist-1, remote_interface: Ethernet1/49${portProps} }
  - name: dist-1
    type: switch
    interfaces:
      - { name: Ethernet1/49, type: ethernet, speed: 10000 }
    trunk_ports:
      - { interface: Ethernet1/49, remote_device: core-1, remote_interface: Ethernet1/1${portProps} }
`,
  });

  it('refuses a switch link that declares neither tagged nor native VLANs', async () => {
    const user = userEvent.setup();
    render(
      <DraftTopologyComposer
        draft={switchPair('')}
        onDraftUpdate={vi.fn()}
        onBusyChange={vi.fn()}
      />,
    );

    await user.click(await screen.findByRole('button', { name: 'Edit first link' }));
    const dialog = screen.getByRole('dialog');

    // The unfinished state the config validator warns about: authoring it here
    // produced a scenario that validated with a warning the author never saw.
    expect(within(dialog).getByRole('button', { name: 'Save link' })).toBeDisabled();
  });

  it('accepts a switch link carrying only a native VLAN as an access port', async () => {
    const user = userEvent.setup();
    render(
      <DraftTopologyComposer
        draft={switchPair(', native_vlan: 10')}
        onDraftUpdate={vi.fn()}
        onBusyChange={vi.fn()}
      />,
    );

    await user.click(await screen.findByRole('button', { name: 'Edit first link' }));
    const dialog = screen.getByRole('dialog');

    expect(within(dialog).getByRole('button', { name: 'Save link' })).toBeEnabled();
  });

  it('accepts a router-to-router link with no VLANs at all', async () => {
    const user = userEvent.setup();
    const routed = {
      ...draft,
      content: `
devices:
  - name: rtr-1
    type: router
    interfaces:
      - { name: Ethernet1/1, type: ethernet, speed: 10000 }
    trunk_ports:
      - { interface: Ethernet1/1, remote_device: rtr-2, remote_interface: Ethernet1/1 }
  - name: rtr-2
    type: router
    interfaces:
      - { name: Ethernet1/1, type: ethernet, speed: 10000 }
    trunk_ports:
      - { interface: Ethernet1/1, remote_device: rtr-1, remote_interface: Ethernet1/1 }
`,
    };
    render(<DraftTopologyComposer draft={routed} onDraftUpdate={vi.fn()} onBusyChange={vi.fn()} />);

    await user.click(await screen.findByRole('button', { name: 'Edit first link' }));
    const dialog = screen.getByRole('dialog');

    // A routed link is neither a trunk nor an access port, and config's
    // IsRoutedTopologyLink says so too.
    expect(within(dialog).getByRole('button', { name: 'Save link' })).toBeEnabled();
  });

  it('re-fits the viewport when a device is added', async () => {
    const user = userEvent.setup();
    const added = {
      ...draft,
      revision: 'revision-2',
      content: 'devices:\n  - name: COS-ACCESS-SW01\n    type: switch\n',
    };
    mutateScenarioDraftTopology.mockResolvedValue(added);
    const Harness = () => {
      const [current, setCurrent] = useState(draft);
      return (
        <DraftTopologyComposer draft={current} onDraftUpdate={setCurrent} onBusyChange={vi.fn()} />
      );
    };
    render(<Harness />);

    await user.click(await screen.findByRole('button', { name: 'Add device' }));
    const dialog = screen.getByRole('dialog');
    await user.type(within(dialog).getByLabelText('Device name'), 'COS-ACCESS-SW01');
    await user.click(within(dialog).getByRole('button', { name: 'Add device' }));

    // fitView on the ReactFlow prop only fits on mount, so a device added to a
    // panned graph landed off-screen.
    await waitFor(() => expect(fitView).toHaveBeenCalled());
  });
});
