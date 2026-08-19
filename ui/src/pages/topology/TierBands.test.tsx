import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { TierBand } from './TierBands';
import type { Tier } from './tiers';

function renderBands(tiers: Tier[]) {
  return render(
    <div>
      {tiers.map((tier) => (
        <TierBand key={tier.label} tier={tier} left={0} width={1000} />
      ))}
    </div>,
  );
}

const CORE: Tier = { label: 'Core', y: 0, height: 300, deviceCount: 2 };
const ACCESS: Tier = { label: 'Access', y: 400, height: 300, deviceCount: 5 };

describe('TierBands', () => {
  it('renders one labelled band per tier', () => {
    renderBands([CORE, ACCESS]);

    const bands = screen.getAllByTestId('topology-tier-band');
    expect(bands).toHaveLength(2);
    expect(bands[0]).toHaveAttribute('data-tier', 'Core');
    expect(bands[1]).toHaveAttribute('data-tier', 'Access');
  });

  it('shows each band device count as a figure', () => {
    renderBands([CORE]);

    const count = screen.getByText('2');
    expect(count).toHaveClass('figure');
  });

  // Bands are chrome. A band that absorbed a click would steal it from the
  // device card underneath, and one announced to a screen reader would read
  // as data the daemon never reported.
  it('is inert and hidden from assistive technology', () => {
    renderBands([CORE]);

    const band = screen.getByTestId('topology-tier-band');
    expect(band).toHaveAttribute('aria-hidden', 'true');
    expect(band).toHaveClass('pointer-events-none');
  });

  it('renders nothing when the layout derived no tiers', () => {
    const { container } = renderBands([]);
    expect(container.querySelectorAll('[data-testid="topology-tier-band"]')).toHaveLength(0);
  });

  it('places each band at the canvas position the layout derived', () => {
    renderBands([ACCESS]);

    expect(screen.getByTestId('topology-tier-band')).toHaveStyle({
      transform: 'translate(0px, 400px)',
      height: '300px',
    });
  });
});
