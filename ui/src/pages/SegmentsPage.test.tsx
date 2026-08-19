/**
 * SegmentsPage.test.tsx — the VLAN-segment device view (ADR 0008).
 *
 * Locks the core contract: GET /api/v1/segments renders one section per
 * segment with a VLAN (or "Untagged") header, a device-count tag, and the
 * shared DeviceTable for that segment's devices — plus the empty states
 * for zero segments and a segment with zero devices.
 */
import { render, screen, within } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { SegmentSummary } from '../api/types';
import '../i18n'; // initialise i18next before the page renders (uses t('segments.*'))
import { SegmentsPage } from './SegmentsPage';

const fetchSegments = vi.fn();

vi.mock('../api/client', () => ({
  fetchSegments: () => fetchSegments(),
}));

// Runtime reads name their session, so a page rendered on its own has to say
// which scenario it is looking at.
vi.mock('../contexts/AppContext', () => ({
  useAppContext: () => ({ sessionId: 'test-session', setSessionId: vi.fn() }),
}));

const taggedSegments: SegmentSummary[] = [
  {
    vlanTag: 200,
    devices: [{ name: 'sw-200', type: 'switch', ips: ['10.0.0.1'], protocols: ['SNMP'] }],
  },
  {
    vlanTag: 300,
    devices: [
      { name: 'sw-300', type: 'switch', ips: ['10.0.1.1'], protocols: ['SNMP'] },
      { name: 'rtr-300', type: 'router', ips: ['10.0.1.2'], protocols: [] },
    ],
  },
];

describe('SegmentsPage', () => {
  beforeEach(() => {
    fetchSegments.mockReset();
  });

  it('renders one section per VLAN segment with its devices grouped underneath', async () => {
    fetchSegments.mockResolvedValue(taggedSegments);

    render(<SegmentsPage />);

    const results = await screen.findByTestId('segments-results');

    const seg200 = within(results).getByTestId('segment-200');
    expect(within(seg200).getByText('VLAN 200')).toBeInTheDocument();
    expect(within(seg200).getByText('sw-200')).toBeInTheDocument();
    expect(within(seg200).getByTestId('segment-count-200')).toHaveTextContent('1');
    expect(within(seg200).getByText('device')).toBeInTheDocument();
    expect(within(seg200).queryByText('sw-300')).not.toBeInTheDocument();

    const seg300 = within(results).getByTestId('segment-300');
    expect(within(seg300).getByText('VLAN 300')).toBeInTheDocument();
    expect(within(seg300).getByText('sw-300')).toBeInTheDocument();
    expect(within(seg300).getByText('rtr-300')).toBeInTheDocument();
    expect(within(seg300).getByTestId('segment-count-300')).toHaveTextContent('2');
    expect(within(seg300).getByText('devices')).toBeInTheDocument();
  });

  it('labels the native VLAN segment "Untagged" for a flat config', async () => {
    const flat: SegmentSummary[] = [
      {
        vlanTag: 0,
        untagged: true,
        devices: [{ name: 'r1', type: 'router', ips: ['10.0.0.1'], protocols: [] }],
      },
    ];
    fetchSegments.mockResolvedValue(flat);

    render(<SegmentsPage />);

    const results = await screen.findByTestId('segments-results');
    const segment = within(results).getByTestId('segment-0');
    expect(within(segment).getByText('Untagged')).toBeInTheDocument();
    expect(within(segment).getByText('r1')).toBeInTheDocument();
    expect(within(segment).getByTestId('segment-count-0')).toHaveTextContent('1');
    expect(within(segment).getByText('device')).toBeInTheDocument();
  });

  // A segment's device count changes as the simulation runs, which makes it a
  // figure. The archetype monospaces figures so a column of them stays
  // comparable at a glance; prose beside them must not be monospaced.
  it('renders each segment device count as a figure', async () => {
    fetchSegments.mockResolvedValue(taggedSegments);

    render(<SegmentsPage />);

    const results = await screen.findByTestId('segments-results');
    const count = within(results).getByTestId('segment-count-300');
    expect(count).toHaveClass('figure');
    expect(count).toHaveTextContent('2');
  });

  it('shows the empty state when no segments are defined', async () => {
    fetchSegments.mockResolvedValue([]);

    render(<SegmentsPage />);

    const results = await screen.findByTestId('segments-results');
    expect(
      within(results).getByText('No segments defined in the loaded configuration.'),
    ).toBeInTheDocument();
  });

  it('shows a per-segment empty state when a segment has no devices', async () => {
    const emptySegment: SegmentSummary[] = [{ vlanTag: 400, devices: [] }];
    fetchSegments.mockResolvedValue(emptySegment);

    render(<SegmentsPage />);

    const results = await screen.findByTestId('segments-results');
    const segment = within(results).getByTestId('segment-400');
    expect(within(segment).getByText('VLAN 400')).toBeInTheDocument();
    expect(within(segment).getByText('No devices in this segment.')).toBeInTheDocument();
  });
});
