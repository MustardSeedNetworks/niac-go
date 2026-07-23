/**
 * LicensePage.test.tsx — Phase 5g: license features render human-readable
 * labels + descriptions sourced from the backend registry instead of raw
 * feature IDs (e.g. "bgp"), and locked (ungranted) features are visually
 * distinguished with an upgrade hint.
 */
import { render, screen } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import { describe, expect, it, vi } from 'vitest';
import type { LicenseFeature, LicenseStatus } from '../contexts/LicenseContext';
import i18n from '../i18n';
import { LicensePage } from './LicensePage';

const mockLicense: {
  status: LicenseStatus | null;
  loading: boolean;
  error: string | null;
  refresh: () => Promise<void>;
} = {
  status: null,
  loading: false,
  error: null,
  refresh: vi.fn(),
};

vi.mock('../contexts/LicenseContext', async () => {
  const actual = await vi.importActual<typeof import('../contexts/LicenseContext')>(
    '../contexts/LicenseContext',
  );
  return {
    ...actual,
    useLicense: () => mockLicense,
  };
});

function renderPage(): ReturnType<typeof render> {
  return render(
    <I18nextProvider i18n={i18n}>
      <LicensePage />
    </I18nextProvider>,
  );
}

const bgpFeature: LicenseFeature = {
  id: 'bgp',
  label: 'BGP Routing',
  description: 'Simulate BGP peering and route advertisement between devices.',
  granted: true,
};

const restApiFeature: LicenseFeature = {
  id: 'rest_api',
  label: 'REST API',
  description: 'Drive NIAC programmatically via the full REST API surface.',
  granted: false,
};

function baseStatus(features: LicenseFeature[]): LicenseStatus {
  return {
    tier: 'Pro',
    isActivated: true,
    isTrialMode: false,
    trialDaysRemaining: 0,
    features,
    licenseEnforced: true,
  };
}

describe('LicensePage', () => {
  it('states the device scale contract', () => {
    mockLicense.status = baseStatus([]);
    mockLicense.loading = false;
    mockLicense.error = null;

    renderPage();

    expect(
      screen.getByText(
        'Free: up to 10 simulated devices; Pro removes tier soft cap; absolute ceiling 1000.',
      ),
    ).toBeInTheDocument();
  });

  it('renders a granted feature by its human-readable label and description, not the raw ID', () => {
    mockLicense.status = baseStatus([bgpFeature]);
    mockLicense.loading = false;
    mockLicense.error = null;

    renderPage();

    expect(screen.getByText('BGP Routing')).toBeInTheDocument();
    expect(
      screen.getByText('Simulate BGP peering and route advertisement between devices.'),
    ).toBeInTheDocument();
    expect(screen.queryByText('bgp')).not.toBeInTheDocument();
  });

  it('marks an ungranted feature as locked with an upgrade hint', () => {
    mockLicense.status = baseStatus([restApiFeature]);
    mockLicense.loading = false;
    mockLicense.error = null;

    renderPage();

    expect(screen.getByText('REST API')).toBeInTheDocument();
    expect(screen.getByText('Requires upgrade')).toBeInTheDocument();
  });
});
