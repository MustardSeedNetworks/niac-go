import { expect, test } from '@playwright/test';

const review = {
  walkName: 'captured/office.walk',
  profile: {
    role: 'captured-switch',
    deviceType: 'switch',
    vendor: 'cisco',
    model: 'Catalyst 9300-48P',
    platform: 'Cisco IOS XE',
    software: '17.15',
    sysObjectId: '1.3.6.1.4.1.9.1.2238',
    interfaceCount: 1,
    supportedSnmpData: ['interfaces', 'system'],
    interfaces: [
      {
        name: 'GigabitEthernet1/0/1',
        type: 'ethernet',
        speed: 1_000_000_000,
        adminStatus: 'up',
        operStatus: 'up',
      },
    ],
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
      total_interfaces: 1,
      physical_interfaces: 1,
      logical_interfaces: 0,
      total_neighbors: 0,
    },
  },
};

test('imports, reviews, and creates a reusable walk profile', async ({ page }) => {
  await page.route('**/api/v1/library/walks', (route) => route.fulfill({ json: [] }));
  await page.route('**/api/v1/walk/import', async (route) => {
    const request = route.request().postDataJSON() as { name: string; content: string };
    expect(request.name).toBe('office.walk');
    expect(request.content).toContain('sysName');
    await route.fulfill({ json: review });
  });
  await page.route('**/api/v1/scenario/profiles/captured', async (route) => {
    const request = route.request().postDataJSON() as { role: string; walkName: string };
    expect(request.role).toBe('office-access');
    expect(request.walkName).toBe('captured/office.walk');
    await route.fulfill({ status: 201, json: { ...review.profile, role: request.role } });
  });

  await page.goto('/walk-analyzer');
  await expect(page.getByText('Online', { exact: true })).toBeVisible();
  await expect(page.getByTestId('walk-profile-creator')).toBeVisible();
  const walkFile = page.getByTestId('walk-profile-file');
  await walkFile.setInputFiles('e2e/fixtures/office.snmpwalk');
  await expect(page.getByTestId('walk-profile-import')).toBeEnabled();
  await page.getByTestId('walk-profile-import').click();
  await expect(page.getByTestId('walk-profile-review')).toBeVisible();
  await page.getByLabel('Profile role').fill('office-access');
  await page.getByTestId('walk-profile-create').click();
  await expect(page.getByRole('status')).toContainText('office-access');
});
