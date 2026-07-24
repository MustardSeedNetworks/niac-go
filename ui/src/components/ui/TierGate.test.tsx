/**
 * TierGate.test.tsx — locks the license-gate behavior plus the #713
 * i18n wiring: when the active license lacks the feature the wrapper
 * renders the locked overlay with the *localized* upgrade tooltip;
 * when it has the feature (or is still loading) it renders children
 * untouched. The Spanish case proves the tooltip is resolved through
 * i18next and interpolated, not a hardcoded English string.
 */

import { render, screen } from '@testing-library/react';
import type { ReactElement } from 'react';
import { I18nextProvider } from 'react-i18next';
import { afterAll, beforeEach, describe, expect, it, vi } from 'vitest';
import i18n from '../../i18n';
import { TierGate } from './TierGate';

// TierGate reads the license through useLicense(); mock the context so
// the test controls hasFeature/loading without a provider or network.
const mockLicense = {
  hasFeature: vi.fn<(feature: string) => boolean>(),
  loading: false,
};
vi.mock('../../contexts/LicenseContext', () => ({
  useLicense: () => mockLicense,
}));

function renderGate(node: ReactElement): ReturnType<typeof render> {
  return render(<I18nextProvider i18n={i18n}>{node}</I18nextProvider>);
}

describe('TierGate', () => {
  beforeEach(async () => {
    await i18n.changeLanguage('en');
    mockLicense.loading = false;
    mockLicense.hasFeature = vi.fn<(feature: string) => boolean>(() => false);
  });

  afterAll(async () => {
    await i18n.changeLanguage('en');
  });

  it('locks the child and shows the localized upgrade tooltip when unlicensed', () => {
    renderGate(
      <TierGate feature="capture.export">
        <button type="button">Export</button>
      </TierGate>,
    );

    // Child stays visible so the user sees what they're missing.
    expect(screen.getByRole('button', { name: 'Export' })).toBeInTheDocument();

    const locked = screen.getByTestId('tier-gate-locked');
    expect(locked).toHaveAttribute('data-feature', 'capture.export');

    // Tooltip text and the overlay title both carry the localized hint.
    expect(screen.getByRole('tooltip')).toHaveTextContent('Requires the Pro tier');
    expect(locked.querySelector('[title]')).toHaveAttribute('title', 'Requires the Pro tier');
  });

  it('interpolates a custom requiredTier into the tooltip', () => {
    renderGate(
      <TierGate feature="capture.export" requiredTier="Pro">
        <button type="button">Export</button>
      </TierGate>,
    );
    expect(screen.getByRole('tooltip')).toHaveTextContent('Requires the Pro tier');
  });

  it('localizes the tooltip when the active language is Spanish', async () => {
    await i18n.changeLanguage('es');
    renderGate(
      <TierGate feature="capture.export">
        <button type="button">Exportar</button>
      </TierGate>,
    );
    expect(screen.getByRole('tooltip')).toHaveTextContent('Requiere el nivel Pro');
  });

  it('honors an explicit message over the localized default', () => {
    renderGate(
      <TierGate feature="capture.export" message="Ask your admin to enable this">
        <button type="button">Export</button>
      </TierGate>,
    );
    const tooltip = screen.getByRole('tooltip');
    expect(tooltip).toHaveTextContent('Ask your admin to enable this');
    expect(tooltip).not.toHaveTextContent('Requires the Pro tier');
  });

  it('renders children untouched when the feature is licensed', () => {
    mockLicense.hasFeature = vi.fn<(feature: string) => boolean>(() => true);
    renderGate(
      <TierGate feature="capture.export">
        <button type="button">Export</button>
      </TierGate>,
    );
    expect(screen.getByRole('button', { name: 'Export' })).toBeInTheDocument();
    expect(screen.queryByTestId('tier-gate-locked')).not.toBeInTheDocument();
  });

  it('renders children untouched while the license is still loading', () => {
    mockLicense.loading = true;
    mockLicense.hasFeature = vi.fn<(feature: string) => boolean>(() => false);
    renderGate(
      <TierGate feature="capture.export">
        <button type="button">Export</button>
      </TierGate>,
    );
    expect(screen.getByRole('button', { name: 'Export' })).toBeInTheDocument();
    expect(screen.queryByTestId('tier-gate-locked')).not.toBeInTheDocument();
  });
});
