/**
 * PacketInspectorPage.test.tsx — export respects the active display filter.
 *
 * Regression coverage for a bug where the Export button always wrote the
 * full unfiltered packet buffer, ignoring whatever display filter the user
 * had dialed in. Export must write exactly the currently filtered packets.
 */
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import '../i18n'; // initialise i18next before the page renders (uses t('packets.inspector.*'))
import { PacketInspectorPage } from './PacketInspectorPage';

const fetchCaptureStatus = vi.fn();
const fetchSimulationStatus = vi.fn();
const fetchUsableInterfaces = vi.fn();
const startStandaloneCapture = vi.fn();
const stopStandaloneCapture = vi.fn();

vi.mock('../api/client', () => ({
  fetchCaptureStatus: () => fetchCaptureStatus(),
  fetchSimulationStatus: () => fetchSimulationStatus(),
  fetchUsableInterfaces: () => fetchUsableInterfaces(),
  startStandaloneCapture: (...args: unknown[]) => startStandaloneCapture(...args),
  stopStandaloneCapture: () => stopStandaloneCapture(),
}));

let capturedOnMessage: ((data: unknown) => void) | undefined;

vi.mock('../hooks/useEventSource', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../hooks/useEventSource')>();
  return {
    ...actual,
    usePacketStream: (options: { onMessage?: (data: unknown) => void }) => {
      capturedOnMessage = options.onMessage;
      return { connected: true, reconnect: vi.fn() };
    },
  };
});

// The page consumes the shared sim-status poll via useSimulationStatus
// (useAppState('simStatus') under an AppProvider). Mock the hook so this
// unit test doesn't need the whole AppProvider + its fan of client polls.
vi.mock('../hooks/useSimulationStatus', () => ({
  useSimulationStatus: () => ({
    data: { running: false },
    loading: false,
    error: null,
    refetch: vi.fn(),
  }),
}));

async function readBlobAsJson(blob: Blob): Promise<unknown> {
  const text = await blob.text();
  return JSON.parse(text);
}

describe('PacketInspectorPage — filtered export', () => {
  beforeEach(() => {
    fetchCaptureStatus.mockReset().mockResolvedValue({ running: true, interface: 'eth0' });
    fetchSimulationStatus.mockReset().mockResolvedValue({ running: false });
    fetchUsableInterfaces.mockReset().mockResolvedValue({ interfaces: [] });
    startStandaloneCapture.mockReset();
    stopStandaloneCapture.mockReset();
    capturedOnMessage = undefined;
  });

  it('exports only the packets matching the active display filter', async () => {
    render(
      <MemoryRouter>
        <PacketInspectorPage />
      </MemoryRouter>,
    );

    await waitFor(() => expect(capturedOnMessage).toBeTypeOf('function'));

    // Push one TCP and one UDP packet through the stream.
    act(() => {
      capturedOnMessage?.({
        type: 'packet',
        timestamp: '2026-07-24T12:00:00.000Z',
        data: {
          protocol: 'TCP',
          sourceIp: '10.0.0.1',
          destIp: '10.0.0.2',
          sourcePort: 1234,
          destPort: 80,
          size: 64,
          summary: 'tcp packet',
          rawData: '00112233',
        },
      });
      capturedOnMessage?.({
        type: 'packet',
        timestamp: '2026-07-24T12:00:01.000Z',
        data: {
          protocol: 'UDP',
          sourceIp: '10.0.0.3',
          destIp: '10.0.0.4',
          sourcePort: 53,
          destPort: 5353,
          size: 32,
          summary: 'udp packet',
          rawData: 'aabbccdd',
        },
      });
    });

    await waitFor(() => expect(screen.getByText('2 / 2 packets')).toBeInTheDocument());

    // Apply a display filter that only matches the TCP packet.
    const filterInput = screen.getByPlaceholderText(/display filter/i);
    fireEvent.change(filterInput, { target: { value: 'protocol == "TCP"' } });

    await waitFor(() => expect(screen.getByText('1 / 2 packets')).toBeInTheDocument());

    let capturedBlob: Blob | undefined;
    const createObjectURLSpy = vi
      .spyOn(URL, 'createObjectURL')
      .mockImplementation((blob: Blob | MediaSource) => {
        capturedBlob = blob as Blob;
        return 'blob:mock';
      });
    const revokeObjectURLSpy = vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => undefined);

    fireEvent.click(screen.getByRole('button', { name: /export/i }));

    expect(createObjectURLSpy).toHaveBeenCalledTimes(1);
    expect(capturedBlob).toBeDefined();
    const exported = (await readBlobAsJson(capturedBlob as Blob)) as Array<{ protocol: string }>;
    expect(exported).toHaveLength(1);
    expect(exported[0]?.protocol).toBe('TCP');

    createObjectURLSpy.mockRestore();
    revokeObjectURLSpy.mockRestore();
  });

  it('shows routed-lab decisions from the packet stream', async () => {
    render(
      <MemoryRouter>
        <PacketInspectorPage />
      </MemoryRouter>,
    );
    await waitFor(() => expect(capturedOnMessage).toBeTypeOf('function'));

    act(() => {
      capturedOnMessage?.({
        type: 'packet',
        timestamp: '2026-07-24T12:00:00.000Z',
        data: {
          protocol: 'IPv4',
          source_ip: '10.10.200.50',
          dest_ip: '10.20.0.10',
          size: 64,
          raw_data: '00112233',
          ingress_network: 'access',
          physical_vlan: 200,
          route_decision: 'forwarded',
          hop: 'edge:inside',
          egress_network: 'servers',
        },
      });
    });

    await userEvent.click(await screen.findByRole('button', { name: /10\.10\.200\.50/ }));

    expect(screen.getByText('Ingress network')).toBeInTheDocument();
    expect(screen.getByText('access')).toBeInTheDocument();
    expect(screen.getByText('forwarded')).toBeInTheDocument();
    expect(screen.getByText('edge:inside')).toBeInTheDocument();
    expect(screen.getByText('servers')).toBeInTheDocument();
    expect(screen.getByText('200')).toBeInTheDocument();
  });

  it('reads packet fields from the exact SSE wire envelope', async () => {
    render(
      <MemoryRouter>
        <PacketInspectorPage />
      </MemoryRouter>,
    );
    await waitFor(() => expect(capturedOnMessage).toBeTypeOf('function'));

    act(() => {
      capturedOnMessage?.({
        type: 'packet',
        timestamp: '2026-07-24T12:00:00.000Z',
        data: {
          protocol: 'TCP',
          source_ip: '192.0.2.10',
          dest_ip: '198.51.100.20',
          source_port: 49152,
          dest_port: 443,
          size: 74,
          summary: 'TCP 49152 → 443',
          raw_data: '00112233',
        },
      });
    });

    expect(
      await screen.findByRole('button', { name: /192\.0\.2\.10.*198\.51\.100\.20/ }),
    ).toBeInTheDocument();
    expect(screen.queryByText(/Unknown.*Unknown/)).not.toBeInTheDocument();
  });

  it('opens offline PCAP analysis when no live stream is active', async () => {
    fetchCaptureStatus.mockResolvedValue({ running: false });

    render(
      <MemoryRouter>
        <PacketInspectorPage />
      </MemoryRouter>,
    );

    await userEvent.click(await screen.findByRole('tab', { name: 'PCAP files' }));

    expect(screen.getByText('Drag & drop PCAP file')).toBeInTheDocument();
  });

  it('opens a bookmarked offline PCAP view directly', async () => {
    fetchCaptureStatus.mockResolvedValue({ running: false });

    render(
      <MemoryRouter initialEntries={['/packets?view=files']}>
        <PacketInspectorPage />
      </MemoryRouter>,
    );

    expect(await screen.findByText('Drag & drop PCAP file')).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: 'PCAP files' })).toHaveAttribute(
      'aria-selected',
      'true',
    );
  });
});
