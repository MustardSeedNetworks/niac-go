/**
 * WalkAnalyzerPage.test.tsx — the SNMP walk analyzer.
 *
 * Locks the core contract: picking a library walk and clicking Analyze
 * calls analyzeWalk with the selected filename, renders the device /
 * statistics / interfaces / neighbors sections from the response, and
 * surfaces a failed analysis as an inline error without crashing.
 */
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { LibraryFileEntry } from '../api/library-client';
import type { WalkAnalysis } from '../api/types';
import '../i18n'; // initialise i18next before the page renders (uses t('walkAnalyzer.*'))
import { WalkAnalyzerPage } from './WalkAnalyzerPage';

const fetchLibraryWalks = vi.fn();
const analyzeWalk = vi.fn();

vi.mock('../api/client', () => ({
  analyzeWalk: (filename: string) => analyzeWalk(filename),
}));
vi.mock('../api/library-client', () => ({
  fetchLibraryWalks: () => fetchLibraryWalks(),
}));

const walkEntry: LibraryFileEntry = {
  name: 'cisco/c3900.walk',
  sizeBytes: 4096,
  modifiedAt: '2026-07-01T00:00:00Z',
  source: 'starter',
  edited: false,
};

const analysis: WalkAnalysis = {
  device: {
    sysname: 'core-switch-01',
    sysdescr: 'Cisco IOS Software, C3900 Software',
    sysobjectid: '1.3.6.1.4.1.9.1.1208',
    syscontact: 'noc@example.com',
  },
  interfaces: [
    {
      index: 1,
      name: 'GigabitEthernet0/1',
      type: 'ethernetCsmacd',
      speed: 1_000_000_000,
      adminStatus: 'up',
      operStatus: 'up',
      macAddress: 'AA:BB:CC:DD:EE:FF',
    },
    {
      index: 2,
      name: 'Loopback0',
      type: 'softwareLoopback',
    },
  ],
  neighbors: [
    {
      localInterface: 'GigabitEthernet0/1',
      remoteDevice: 'core-switch-02',
      remoteInterface: 'GigabitEthernet0/2',
      protocol: 'lldp',
    },
  ],
  statistics: {
    totalInterfaces: 2,
    physicalInterfaces: 1,
    logicalInterfaces: 1,
    totalNeighbors: 1,
  },
};

describe('WalkAnalyzerPage', () => {
  beforeEach(() => {
    fetchLibraryWalks.mockReset();
    analyzeWalk.mockReset();
  });

  it('calls analyzeWalk with the picked filename and renders the results', async () => {
    fetchLibraryWalks.mockResolvedValue([walkEntry]);
    analyzeWalk.mockResolvedValue({ success: true, result: analysis });

    render(<WalkAnalyzerPage />);

    const picker = await screen.findByTestId('walk-analyzer-picker');
    await waitFor(() => expect(picker).toHaveValue(walkEntry.name));

    fireEvent.click(screen.getByTestId('walk-analyzer-analyze-button'));

    await waitFor(() => expect(analyzeWalk).toHaveBeenCalledWith(walkEntry.name));

    const results = await screen.findByTestId('walk-analyzer-results');
    expect(within(results).getByText('core-switch-01')).toBeInTheDocument();
    expect(within(results).getByText('1.0 Gbps')).toBeInTheDocument();
    expect(within(results).getByText('LLDP')).toBeInTheDocument();
    expect(within(results).getByText('core-switch-02')).toBeInTheDocument();
  });

  it('shows the neighbors empty state when the walk has no LLDP/CDP adjacencies', async () => {
    fetchLibraryWalks.mockResolvedValue([walkEntry]);
    analyzeWalk.mockResolvedValue({
      success: true,
      result: {
        ...analysis,
        neighbors: [],
        statistics: { ...analysis.statistics, totalNeighbors: 0 },
      },
    });

    render(<WalkAnalyzerPage />);
    await screen.findByTestId('walk-analyzer-picker');
    fireEvent.click(screen.getByTestId('walk-analyzer-analyze-button'));

    const neighborsTable = await screen.findByTestId('walk-analyzer-neighbors-table');
    expect(within(neighborsTable).getByText('No LLDP/CDP neighbors found.')).toBeInTheDocument();
  });

  it('surfaces an error message when analysis fails, without crashing', async () => {
    fetchLibraryWalks.mockResolvedValue([walkEntry]);
    analyzeWalk.mockRejectedValue(new Error('walk file not found'));

    render(<WalkAnalyzerPage />);
    await screen.findByTestId('walk-analyzer-picker');
    fireEvent.click(screen.getByTestId('walk-analyzer-analyze-button'));

    expect(await screen.findByRole('alert')).toHaveTextContent('walk file not found');
    expect(screen.queryByTestId('walk-analyzer-results')).not.toBeInTheDocument();
  });
});
