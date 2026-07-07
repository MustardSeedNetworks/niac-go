/**
 * SynthesizeWalkControl.test.tsx
 *
 * Verifies the real contract: the vendor/model catalog renders as a
 * grouped, selectable picker; Generate POSTs {vendor, model} for the
 * selection and reports the new walkPath back to the caller; the
 * disabled (unsaved-device) state never lets the POST fire; and both a
 * generic failure and the specific 422 "unsupported_combo" failure
 * surface a readable error.
 */
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import '../../i18n';
import { ApiError } from '../../api/errors';
import type { ModelDescriptor } from '../../api/types';
import { SynthesizeWalkControl } from './SynthesizeWalkControl';

const fetchSynthesizeWalkModels = vi.fn();
const synthesizeWalk = vi.fn();

vi.mock('../../api/client', () => ({
  fetchSynthesizeWalkModels: () => fetchSynthesizeWalkModels(),
  synthesizeWalk: (hostname: string, payload: unknown) => synthesizeWalk(hostname, payload),
}));

const models: ModelDescriptor[] = [
  {
    vendor: 'cisco-ios',
    model: 'catalyst-9300-48p',
    label: 'Catalyst 9300-48P (48× 1G PoE+)',
    type: 'switch',
    ifCount: 48,
    speedMbps: 1000,
  },
  {
    vendor: 'cisco-ios',
    model: 'isr-4451-x',
    label: 'ISR 4451-X (4× 1G WAN/LAN)',
    type: 'router',
    ifCount: 4,
    speedMbps: 1000,
  },
  {
    vendor: 'junos',
    model: 'ex4300-48t',
    label: 'EX4300-48T (48× 1G)',
    type: 'switch',
    ifCount: 48,
    speedMbps: 1000,
  },
];

describe('SynthesizeWalkControl', () => {
  beforeEach(() => {
    fetchSynthesizeWalkModels.mockReset();
    synthesizeWalk.mockReset();
  });

  it('renders the catalog grouped by vendor, with each model selectable', async () => {
    fetchSynthesizeWalkModels.mockResolvedValue(models);

    render(<SynthesizeWalkControl hostname="edge-01" disabled={false} onSynthesized={vi.fn()} />);

    await screen.findByText('Catalyst 9300-48P (48× 1G PoE+)');

    const select = screen.getByTestId('synthesize-walk-model-select') as HTMLSelectElement;
    const optgroups = select.querySelectorAll('optgroup');
    expect(optgroups).toHaveLength(2);
    expect(optgroups[0]).toHaveAttribute('label', 'cisco-ios');
    expect(optgroups[1]).toHaveAttribute('label', 'junos');

    // Placeholder + 3 models across the two vendor groups.
    expect(select.querySelectorAll('option')).toHaveLength(4);
    expect(screen.getByText('ISR 4451-X (4× 1G WAN/LAN)')).toBeInTheDocument();
    expect(screen.getByText('EX4300-48T (48× 1G)')).toBeInTheDocument();
  });

  it('generates a walk for the selected model and reports the new walkPath', async () => {
    fetchSynthesizeWalkModels.mockResolvedValue(models);
    synthesizeWalk.mockResolvedValue({
      walkPath: 'synthesized/edge-01.walk',
      oidCount: 42,
      sizeBytes: 1234,
      originalPreserved: true,
    });
    const onSynthesized = vi.fn();

    render(
      <SynthesizeWalkControl hostname="edge-01" disabled={false} onSynthesized={onSynthesized} />,
    );

    await screen.findByText('Catalyst 9300-48P (48× 1G PoE+)');
    fireEvent.change(screen.getByTestId('synthesize-walk-model-select'), {
      target: { value: '0' },
    });
    fireEvent.click(screen.getByTestId('synthesize-walk-generate'));

    await waitFor(() =>
      expect(synthesizeWalk).toHaveBeenCalledWith('edge-01', {
        vendor: 'cisco-ios',
        model: 'catalyst-9300-48p',
      }),
    );
    await waitFor(() => expect(onSynthesized).toHaveBeenCalledWith('synthesized/edge-01.walk'));
    expect(
      screen.getByText(/Generated Catalyst 9300-48P.*42 OIDs.*1234 bytes.*original preserved/),
    ).toBeInTheDocument();
  });

  it('disables the control and never calls synthesizeWalk when the device is unsaved', async () => {
    fetchSynthesizeWalkModels.mockResolvedValue(models);

    render(<SynthesizeWalkControl hostname="new" disabled={true} onSynthesized={vi.fn()} />);

    expect(
      await screen.findByText('Save the device before generating a walk.'),
    ).toBeInTheDocument();
    expect(screen.getByTestId('synthesize-walk-model-select')).toBeDisabled();
    expect(screen.getByTestId('synthesize-walk-generate')).toBeDisabled();

    fireEvent.click(screen.getByTestId('synthesize-walk-generate'));
    expect(synthesizeWalk).not.toHaveBeenCalled();
  });

  it('surfaces a rejected synthesize call as an alert', async () => {
    fetchSynthesizeWalkModels.mockResolvedValue(models);
    synthesizeWalk.mockRejectedValue(new Error('write_failed: disk full'));

    render(<SynthesizeWalkControl hostname="edge-01" disabled={false} onSynthesized={vi.fn()} />);

    await screen.findByText('Catalyst 9300-48P (48× 1G PoE+)');
    fireEvent.change(screen.getByTestId('synthesize-walk-model-select'), {
      target: { value: '0' },
    });
    fireEvent.click(screen.getByTestId('synthesize-walk-generate'));

    await waitFor(() =>
      expect(screen.getByRole('alert')).toHaveTextContent('write_failed: disk full'),
    );
  });

  it('maps a 422 unsupported_combo failure to a specific message', async () => {
    fetchSynthesizeWalkModels.mockResolvedValue(models);
    synthesizeWalk.mockRejectedValue(
      new ApiError('synth: no profile for vendor=junos type=firewall', 422, 'unsupported_combo'),
    );

    render(<SynthesizeWalkControl hostname="edge-01" disabled={false} onSynthesized={vi.fn()} />);

    await screen.findByText('EX4300-48T (48× 1G)');
    fireEvent.change(screen.getByTestId('synthesize-walk-model-select'), {
      target: { value: '2' },
    });
    fireEvent.click(screen.getByTestId('synthesize-walk-generate'));

    await waitFor(() =>
      expect(screen.getByRole('alert')).toHaveTextContent(
        'This vendor has no profile for this device type.',
      ),
    );
  });
});
