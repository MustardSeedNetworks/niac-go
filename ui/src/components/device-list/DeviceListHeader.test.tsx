/**
 * DeviceListHeader.test.tsx — Device Library -> Running Devices cross-link
 * (naming-collision resolution, PR 2a).
 *
 * The Device Library (`/device-config`) previously had no way back to the
 * read-only Running Devices view (`/devices`). Pins the new "View running
 * state" action navigating there.
 */
import { fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import '../../i18n';
import { DeviceListHeader } from './DeviceListHeader';

const mockNavigate = vi.fn();
vi.mock('react-router-dom', async (importOriginal) => {
  const actual = await importOriginal<typeof import('react-router-dom')>();
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  };
});

afterEach(() => {
  vi.clearAllMocks();
});

describe('DeviceListHeader', () => {
  it('navigates to /devices when "View running state" is clicked', () => {
    render(
      <DeviceListHeader deviceCount={0} filteredDevices={[]} loading={false} onRefresh={vi.fn()} />,
    );

    fireEvent.click(screen.getByTestId('view-running-state'));
    expect(mockNavigate).toHaveBeenCalledWith('/devices');
  });
});
