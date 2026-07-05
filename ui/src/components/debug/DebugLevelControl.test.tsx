/**
 * DebugLevelControl.test.tsx — the single global debug-level control.
 *
 * Verifies the real contract: the current level from GET is shown selected, a
 * click PUTs the chosen level and reflects it, and a fetch failure surfaces an
 * error and disables the control (no active simulation).
 */
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { DebugLevelControl } from './DebugLevelControl';

const fetchDebugLevel = vi.fn();
const updateDebugLevel = vi.fn();

vi.mock('../../api/client', () => ({
  fetchDebugLevel: () => fetchDebugLevel(),
  updateDebugLevel: (payload: unknown) => updateDebugLevel(payload),
}));

describe('DebugLevelControl', () => {
  beforeEach(() => {
    fetchDebugLevel.mockReset();
    updateDebugLevel.mockReset();
  });

  it('marks the current level from the API as selected', async () => {
    fetchDebugLevel.mockResolvedValue({ level: 'info', defaultLevel: 'basic' });

    render(<DebugLevelControl />);

    const infoRadio = await screen.findByTestId('debug-level-info');
    await waitFor(() => expect(infoRadio).toBeChecked());
    expect(screen.getByTestId('debug-level-off')).not.toBeChecked();
  });

  it('PUTs the chosen level and reflects it', async () => {
    fetchDebugLevel.mockResolvedValue({ level: 'basic', defaultLevel: 'basic' });
    updateDebugLevel.mockResolvedValue({ level: 'trace', defaultLevel: 'basic' });

    render(<DebugLevelControl />);

    await waitFor(() => expect(screen.getByTestId('debug-level-basic')).toBeChecked());

    fireEvent.click(screen.getByTestId('debug-level-trace'));

    await waitFor(() => expect(updateDebugLevel).toHaveBeenCalledWith({ level: 'trace' }));
    await waitFor(() => expect(screen.getByTestId('debug-level-trace')).toBeChecked());
  });

  it('surfaces an error and disables the control when the level is unavailable', async () => {
    fetchDebugLevel.mockRejectedValue(new Error('no active simulation'));

    render(<DebugLevelControl />);

    const offButton = await screen.findByTestId('debug-level-off');
    await waitFor(() => expect(offButton).toBeDisabled());
    expect(screen.getByRole('alert')).toBeInTheDocument();
    expect(updateDebugLevel).not.toHaveBeenCalled();
  });
});
