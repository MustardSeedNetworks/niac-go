import { fireEvent, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import { defaultScenarioRequest } from '../../api/scenario-client';
import '../../i18n';
import { FleetGeneratorCard } from './FleetGeneratorCard';

describe('FleetGeneratorCard', () => {
  it('exposes bounded repeat controls and reports edits', () => {
    const onChange = vi.fn();
    render(
      <FleetGeneratorCard
        request={defaultScenarioRequest()}
        selected={false}
        customizedFromPackId={null}
        onChange={onChange}
        onSelect={vi.fn()}
      />,
    );

    fireEvent.click(screen.getByTestId('fleet-customize'));
    const accessSwitches = screen.getByRole('spinbutton', {
      name: 'Access switches per site',
    });
    expect(accessSwitches).toHaveAttribute('min', '1');
    expect(accessSwitches).toHaveAttribute('max', '20');
    fireEvent.change(accessSwitches, { target: { value: '8' } });
    expect(onChange).toHaveBeenLastCalledWith(
      expect.objectContaining({ counts: expect.objectContaining({ accessSwitches: 8 }) }),
    );
  });

  it('keeps the site count within the supported range', async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(
      <FleetGeneratorCard
        request={defaultScenarioRequest()}
        selected
        customizedFromPackId={null}
        onChange={onChange}
        onSelect={vi.fn()}
      />,
    );

    await user.click(screen.getByTestId('fleet-customize'));
    expect(screen.getByRole('button', { name: 'Add site' })).toBeDisabled();
    await user.click(screen.getByRole('button', { name: 'Remove site' }));
    expect(onChange).toHaveBeenCalledWith(
      expect.objectContaining({
        sites: expect.arrayContaining([expect.objectContaining({ code: 'COS' })]),
      }),
    );
    expect(onChange.mock.calls[0]?.[0]?.sites).toHaveLength(3);
  });

  it('keeps redundant layers and endpoint totals valid', () => {
    const onChange = vi.fn();
    const request = defaultScenarioRequest();
    request.counts.workstationsPerAccess = 12;
    render(
      <FleetGeneratorCard
        request={request}
        selected
        customizedFromPackId={null}
        onChange={onChange}
        onSelect={vi.fn()}
      />,
    );

    fireEvent.click(screen.getByTestId('fleet-customize'));
    fireEvent.change(
      screen.getByRole('spinbutton', { name: 'Redundant WAN, firewall, and core pairs' }),
      { target: { value: '1' } },
    );
    expect(onChange).toHaveBeenLastCalledWith(
      expect.objectContaining({
        counts: expect.objectContaining({ siteWanRouters: 1, firewalls: 1, coreSwitches: 1 }),
      }),
    );

    fireEvent.change(screen.getByRole('spinbutton', { name: 'Access switches per site' }), {
      target: { value: '20' },
    });
    expect(onChange).toHaveBeenLastCalledWith(
      expect.objectContaining({
        counts: expect.objectContaining({ accessSwitches: 20, workstationsPerAccess: 9 }),
      }),
    );
    // Sixteen access switches share the site's 180 wired endpoints, so the
    // per-switch input stops at eleven, below the spare-port ceiling of 33.
    expect(
      screen.getByRole('spinbutton', { name: 'Wired workstations per access switch' }),
    ).toHaveAttribute('max', '11');
  });

  it('says so when the fleet was tuned away from a pack', () => {
    render(
      <FleetGeneratorCard
        request={defaultScenarioRequest()}
        selected
        customizedFromPackId="hospital"
        onChange={vi.fn()}
        onSelect={vi.fn()}
      />,
    );

    expect(screen.getByTestId('fleet-customized-notice')).toHaveTextContent(
      'Customized from Hospital network. This is now a custom fleet.',
    );
  });
});
