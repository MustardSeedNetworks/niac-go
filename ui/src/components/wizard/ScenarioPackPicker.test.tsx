import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defaultScenarioRequest, type ScenarioPack } from '../../api/scenario-client';
import i18n from '../../i18n';
import { renderWithResources as render } from '../../test/renderWithResources';
import { ScenarioPackPicker } from './ScenarioPackPicker';

const fetchScenarioPacks = vi.hoisted(() => vi.fn());

vi.mock('../../api/scenario-client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../api/scenario-client')>();
  return { ...actual, fetchScenarioPacks };
});

const hospital: ScenarioPack = {
  id: 'hospital',
  version: '1.2.0',
  manifestVersion: 4,
  name: 'Hospital network',
  description: 'Acute-care and ambulatory sites.',
  mapPurpose: 'presentation',
  request: { ...defaultScenarioRequest(), domain: 'care.example' },
  manifest: {
    deviceCount: 75,
    networkCount: 12,
    linkCount: 88,
    deviceNamesSha256: 'devices',
    networksSha256: 'networks',
    linksSha256: 'links',
    interfacesSha256: 'interfaces',
  },
};

const enterpriseScale: ScenarioPack = {
  ...hospital,
  id: 'enterprise-scale',
  name: 'Enterprise scale reference',
  description: 'Stress workload.',
  mapPurpose: 'stress',
  manifest: { ...hospital.manifest, deviceCount: 531, linkCount: 634 },
};

describe('ScenarioPackPicker', () => {
  beforeEach(async () => {
    fetchScenarioPacks.mockReset().mockResolvedValue([hospital]);
    await i18n.changeLanguage('en');
  });

  it('loads a versioned pack into the editable composer', async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(<ScenarioPackPicker request={defaultScenarioRequest()} onChange={onChange} />);

    await user.click(await screen.findByTestId('scenario-pack-hospital'));

    expect(onChange).toHaveBeenCalledWith(hospital.request);
    // The counts come from the pack, never from the prose: the four sites and
    // 128 radios are the default request's own shape (16 access switches x 2
    // radios x 4 sites), which is why a description may not restate them.
    expect(
      screen.getByText(/Version 1.2.0 · 4 sites · 75 devices · 128 access points · 88 links/),
    ).toBeVisible();
  });

  it('renders localized pack metadata', async () => {
    await i18n.changeLanguage('es');

    render(<ScenarioPackPicker request={defaultScenarioRequest()} onChange={vi.fn()} />);

    expect(await screen.findByText('Red hospitalaria')).toBeVisible();
    expect(screen.getByText(/Centro médico con acceso cableado resiliente/)).toBeVisible();
  });

  it('separates presentation maps from scale workloads', async () => {
    fetchScenarioPacks.mockResolvedValueOnce([enterpriseScale, hospital]);

    render(<ScenarioPackPicker request={defaultScenarioRequest()} onChange={vi.fn()} />);

    expect(await screen.findByText('Presentation-ready maps')).toBeVisible();
    expect(screen.getByText('Scale testing')).toBeVisible();
    expect(screen.getByTestId('scenario-pack-hospital')).toBeVisible();
    expect(screen.getByTestId('scenario-pack-enterprise-scale')).toBeVisible();
  });
});
