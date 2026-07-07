/**
 * PacketInspectorPage.test.tsx — export respects the active display filter.
 *
 * Regression coverage for a bug where the Export button always wrote the
 * full unfiltered packet buffer, ignoring whatever display filter the user
 * had dialed in. Export must write exactly the currently filtered packets.
 */
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
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

vi.mock('../hooks/useEventSource', () => ({
  usePacketStream: (options: { onMessage?: (data: unknown) => void }) => {
    capturedOnMessage = options.onMessage;
    return { connected: true, reconnect: vi.fn() };
  },
}));

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
    capturedOnMessage?.({
      protocol: 'TCP',
      sourceIp: '10.0.0.1',
      destIp: '10.0.0.2',
      sourcePort: 1234,
      destPort: 80,
      size: 64,
      summary: 'tcp packet',
      rawData: '00112233',
    });
    capturedOnMessage?.({
      protocol: 'UDP',
      sourceIp: '10.0.0.3',
      destIp: '10.0.0.4',
      sourcePort: 53,
      destPort: 5353,
      size: 32,
      summary: 'udp packet',
      rawData: 'aabbccdd',
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
});
