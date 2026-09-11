import { render, screen } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import { describe, expect, it } from 'vitest';
import i18n from '../../i18n';
import { TierBand } from './TierBands';
import type { Tier } from './tiers';

function renderBands(tiers: Tier[]) {
  return render(
    <I18nextProvider i18n={i18n}>
      <div>
        {tiers.map((tier) => (
          <TierBand key={`${tier.label}-${tier.depth ?? 0}`} tier={tier} left={0} width={1000} />
        ))}
      </div>
    </I18nextProvider>,
  );
}

const CORE: Tier = { label: 'Core', y: 0, height: 300, deviceCount: 2 };
const ACCESS: Tier = { label: 'Access', y: 400, height: 300, deviceCount: 5 };
const DIST_1: Tier = { label: 'Distribution', depth: 1, y: 100, height: 300, deviceCount: 1 };
const DIST_2: Tier = { label: 'Distribution', depth: 2, y: 200, height: 300, deviceCount: 3 };

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

  // The defect (#2061): a six-rank graph drew four bands all reading
  // "Distribution", so nothing on screen said which layer you were looking at.
  it('distinguishes numbered distribution bands on screen', () => {
    renderBands([DIST_1, DIST_2]);

    const labels = screen
      .getAllByTestId('topology-tier-band')
      .map((band) => band.textContent?.replace(/\s+/g, ' ').trim());

    expect(labels[0]).toContain('Distribution 1');
    expect(labels[1]).toContain('Distribution 2');
    expect(new Set(labels).size).toBe(2);
  });

  it('leaves a lone distribution band unnumbered', () => {
    renderBands([{ label: 'Distribution', y: 0, height: 300, deviceCount: 2 }]);

    const text = screen.getByTestId('topology-tier-band').textContent ?? '';
    expect(text).toContain('Distribution');
    expect(text).not.toMatch(/Distribution\s*\d/);
  });

  it('exposes the band depth for assertions and selectors', () => {
    renderBands([CORE, DIST_2]);

    const bands = screen.getAllByTestId('topology-tier-band');
    expect(bands[0]).not.toHaveAttribute('data-tier-depth');
    expect(bands[1]).toHaveAttribute('data-tier-depth', '2');
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
