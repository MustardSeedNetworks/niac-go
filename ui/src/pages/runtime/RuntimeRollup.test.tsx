/**
 * RuntimeRollup.test.tsx — pins the state derivation, which is the whole
 * point of the component.
 *
 * The case that matters is "no daemon": the page must not answer the question
 * "is this run healthy" with a row of zeros when there is no run to measure.
 * A rollup that renders 0 devices / 0 uptime says "nothing is wrong" at
 * exactly the moment nobody can tell.
 */
import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { SimulationStatus } from '../../api/types';
import '../../i18n';
import { RuntimeRollup } from './RuntimeRollup';

const running: SimulationStatus = {
  running: true,
  interface: 'eth0',
  deviceCount: 3,
  uptimeSeconds: 42,
};

function renderRollup(props: Partial<{ simStatus: SimulationStatus | null; loading: boolean }>) {
  return render(
    <RuntimeRollup simStatus={props.simStatus ?? null} loading={props.loading ?? false} />,
  );
}

function state(container: HTMLElement): string | null {
  return container.querySelector('[data-state]')?.getAttribute('data-state') ?? null;
}

describe('RuntimeRollup', () => {
  it('reports no data, not all-clear, while the first poll is in flight', () => {
    const { container } = renderRollup({ simStatus: null, loading: true });
    expect(state(container)).toBe('unknown');
    expect(screen.getByText('Checking the daemon')).toBeInTheDocument();
  });

  it('treats a missing daemon as unmeasurable and prints em dashes, not zeros', () => {
    const { container } = renderRollup({ simStatus: null, loading: false });
    expect(state(container)).toBe('unknown');
    expect(screen.getByText('NIAC is not running in daemon mode')).toBeInTheDocument();
    // All three figures refuse to render a value.
    expect(screen.getAllByText('—')).toHaveLength(3);
    expect(screen.queryByText('0')).not.toBeInTheDocument();
  });

  it("uses the daemon's own reason as the headline when a run is degraded", () => {
    const { container } = renderRollup({
      simStatus: { ...running, degraded: true, degradedReason: 'Shared trunk veth is down' },
    });
    expect(state(container)).toBe('warn');
    expect(screen.getByText('Shared trunk veth is down')).toBeInTheDocument();
  });

  it('falls back to a stated reason when the daemon reports degraded without one', () => {
    const { container } = renderRollup({ simStatus: { ...running, degraded: true } });
    expect(state(container)).toBe('warn');
    expect(
      screen.getByText('The simulation is running but cannot exchange frames'),
    ).toBeInTheDocument();
  });

  it('names the interface and device count while a run is healthy', () => {
    const { container } = renderRollup({ simStatus: running });
    expect(state(container)).toBe('ok');
    expect(screen.getByText('Simulating 3 devices on eth0')).toBeInTheDocument();
    expect(screen.getByText('42s')).toBeInTheDocument();
  });

  it('treats an idle daemon as calm rather than a failure', () => {
    const { container } = renderRollup({
      simStatus: { running: false, deviceCount: 0, uptimeSeconds: 0 },
    });
    expect(state(container)).toBe('ok');
    expect(screen.getByText('No simulation is running')).toBeInTheDocument();
  });
});
