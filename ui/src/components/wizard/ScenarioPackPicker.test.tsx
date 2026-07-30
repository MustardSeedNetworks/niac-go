import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { enterpriseScenarioRequest, type ScenarioPack } from '../../api/scenario-client';
import i18n from '../../i18n';
import { ScenarioPackPicker } from './ScenarioPackPicker';

const fetchScenarioPacks = vi.hoisted(() => vi.fn());
const hasFeature = vi.hoisted(() => vi.fn(() => true));

vi.mock('../../api/scenario-client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../api/scenario-client')>();
  return { ...actual, fetchScenarioPacks };
});

vi.mock('../../contexts/LicenseContext', () => ({
  useLicense: () => ({ hasFeature, loading: false }),
}));

const hospital: ScenarioPack = {
  id: 'hospital',
  version: '1.0.0',
  manifestVersion: 1,
  name: 'Hospital network',
  description: 'Acute-care and ambulatory sites.',
  request: { ...enterpriseScenarioRequest(), domain: 'care.example' },
  manifest: {
    deviceCount: 351,
    networkCount: 30,
    linkCount: 416,
    deviceNamesSha256: 'devices',
    networksSha256: 'networks',
    linksSha256: 'links',
  },
};

describe('ScenarioPackPicker', () => {
  beforeEach(async () => {
    fetchScenarioPacks.mockReset().mockResolvedValue([hospital]);
    hasFeature.mockReset().mockReturnValue(true);
    await i18n.changeLanguage('en');
  });

  it('loads a versioned pack into the editable composer', async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(<ScenarioPackPicker request={enterpriseScenarioRequest()} onChange={onChange} />);

    await user.click(await screen.findByTestId('scenario-pack-hospital'));

    expect(onChange).toHaveBeenCalledWith(hospital.request);
    expect(screen.getByText(/Version 1.0.0 · 351 devices · 416 links/)).toBeVisible();
  });

  it('does not request Pro-only packs without the entitlement', () => {
    hasFeature.mockReturnValueOnce(false);

    render(<ScenarioPackPicker request={enterpriseScenarioRequest()} onChange={vi.fn()} />);

    expect(fetchScenarioPacks).not.toHaveBeenCalled();
    expect(screen.queryByTestId('scenario-pack-hospital')).not.toBeInTheDocument();
  });

  it('renders localized pack metadata', async () => {
    await i18n.changeLanguage('es');

    render(<ScenarioPackPicker request={enterpriseScenarioRequest()} onChange={vi.fn()} />);

    expect(await screen.findByText('Red hospitalaria')).toBeVisible();
    expect(screen.getByText(/Sitios de cuidados intensivos/)).toBeVisible();
  });
});
