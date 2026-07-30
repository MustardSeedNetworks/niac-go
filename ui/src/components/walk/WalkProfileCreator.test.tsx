import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { WalkProfileReview } from '../../api/walk-profile-client';
import '../../i18n';
import { WalkProfileCreator } from './WalkProfileCreator';

const importWalkProfile = vi.fn();
const captureWalkProfile = vi.fn();
const createCapturedProfile = vi.fn();

vi.mock('../../api/walk-profile-client', () => ({
  importWalkProfile: (...args: unknown[]) => importWalkProfile(...args),
  captureWalkProfile: (...args: unknown[]) => captureWalkProfile(...args),
  createCapturedProfile: (...args: unknown[]) => createCapturedProfile(...args),
}));

const review: WalkProfileReview = {
  walkName: 'captured/office.walk',
  profile: {
    role: 'captured-switch',
    deviceType: 'switch',
    vendor: 'cisco',
    model: 'Catalyst 9300-48P',
    platform: 'Cisco IOS XE',
    software: '17.15',
    sysObjectId: '1.3.6.1.4.1.9.1.2238',
    interfaceCount: 48,
    supportedSnmpData: ['interfaces', 'lldp', 'system'],
    source: 'captured',
  },
  analysis: {
    device: {
      sysname: 'device-001',
      sysdescr: 'Cisco IOS XE',
      sysobjectid: '1.3.6.1.4.1.9.1.2238',
    },
    interfaces: [],
    neighbors: [],
    statistics: {
      totalInterfaces: 48,
      physicalInterfaces: 48,
      logicalInterfaces: 0,
      totalNeighbors: 1,
    },
  },
};

describe('WalkProfileCreator', () => {
  beforeEach(() => {
    importWalkProfile.mockReset();
    captureWalkProfile.mockReset();
    createCapturedProfile.mockReset();
  });

  it('imports a local walk and requires profile review before creation', async () => {
    importWalkProfile.mockResolvedValue(review);
    createCapturedProfile.mockResolvedValue({ ...review.profile, role: 'office-access' });
    render(<WalkProfileCreator />);

    const file = new File(['walk content'], 'office.snmpwalk', { type: 'text/plain' });
    fireEvent.change(screen.getByTestId('walk-profile-file'), { target: { files: [file] } });
    fireEvent.click(screen.getByTestId('walk-profile-import'));

    expect(await screen.findByTestId('walk-profile-review')).toBeInTheDocument();
    expect(importWalkProfile).toHaveBeenCalledWith(
      'office.walk',
      'walk content',
      expect.any(AbortSignal),
    );
    expect(screen.getByTestId('walk-profile-file')).toHaveValue('');
    fireEvent.change(screen.getByLabelText('Profile role'), { target: { value: 'office-access' } });
    fireEvent.click(screen.getByTestId('walk-profile-create'));

    await waitFor(() =>
      expect(createCapturedProfile).toHaveBeenCalledWith(
        expect.objectContaining({ role: 'office-access', walkName: 'captured/office.walk' }),
      ),
    );
    expect(await screen.findByRole('status')).toHaveTextContent('office-access');
  });

  it('clears request-only community data after direct capture', async () => {
    captureWalkProfile.mockResolvedValue(review);
    render(<WalkProfileCreator />);

    fireEvent.click(screen.getByRole('button', { name: 'Capture from device' }));
    fireEvent.change(screen.getByLabelText('Target IP address'), {
      target: { value: '192.0.2.10' },
    });
    fireEvent.change(screen.getByTestId('walk-profile-community'), {
      target: { value: 'private-secret' },
    });
    fireEvent.click(screen.getByTestId('walk-profile-capture'));

    await waitFor(() => expect(captureWalkProfile).toHaveBeenCalled());
    expect(captureWalkProfile.mock.calls[0][1]).toEqual(
      expect.objectContaining({
        target: '192.0.2.10',
        community: 'private-secret',
        timeoutSeconds: 20,
      }),
    );
    await waitFor(() => expect(screen.getByTestId('walk-profile-community')).toHaveValue(''));
    expect(await screen.findByTestId('walk-profile-review')).toBeInTheDocument();
  });

  it('clears all request-only SNMPv3 credentials after direct capture', async () => {
    captureWalkProfile.mockResolvedValue(review);
    render(<WalkProfileCreator />);

    fireEvent.click(screen.getByRole('button', { name: 'Capture from device' }));
    fireEvent.change(screen.getByLabelText('Target IP address'), {
      target: { value: '192.0.2.10' },
    });
    fireEvent.change(screen.getByLabelText('SNMP version'), { target: { value: '3' } });
    fireEvent.change(screen.getByTestId('walk-profile-username'), {
      target: { value: 'capture-user' },
    });
    fireEvent.click(screen.getByTestId('walk-profile-capture'));

    await waitFor(() => expect(captureWalkProfile).toHaveBeenCalled());
    await waitFor(() => expect(screen.getByTestId('walk-profile-username')).toHaveValue(''));
  });
});
