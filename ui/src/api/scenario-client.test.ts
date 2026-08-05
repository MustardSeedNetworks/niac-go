import { beforeEach, describe, expect, it, vi } from 'vitest';

const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);

describe('scenario generator client', () => {
  beforeEach(() => {
    vi.resetModules();
    mockFetch.mockReset();
  });

  it('submits the enterprise repeat controls to the protected generator', async () => {
    mockFetch
      .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ token: 'csrf-token' }) })
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve({ content: 'devices: []\n', manifest: { device_count: 0 } }),
      });

    const { enterpriseScenarioRequest, generateScenario } = await import('./scenario-client');
    const request = enterpriseScenarioRequest();
    const response = await generateScenario(request);

    expect(response.manifest.deviceCount).toBe(0);
    expect(mockFetch.mock.calls[1][0]).toContain('/api/v1/scenario/generate');
    const payload = JSON.parse(mockFetch.mock.calls[1][1]?.body as string);
    expect(payload).toMatchObject({
      counts: { accessSwitches: 16, accessPointsPerAccess: 2 },
      attachmentName: 'cyberscope',
      endpointProfile: 'enterprise',
    });
    expect(payload.sites).toHaveLength(4);
    expect(payload.sites[0]).toMatchObject({ code: 'COS', octet: 240 });
  });

  it('rejects cross-field combinations the generator cannot build', async () => {
    const { enterpriseScenarioRequest, isScenarioRequestValid } = await import('./scenario-client');
    const request = enterpriseScenarioRequest();
    expect(isScenarioRequestValid(request)).toBe(true);

    request.counts.accessSwitches = 20;
    expect(isScenarioRequestValid(request)).toBe(false);
    request.counts.workstationsPerAccess = 3;
    expect(isScenarioRequestValid(request)).toBe(true);

    request.counts.firewalls = 1;
    expect(isScenarioRequestValid(request)).toBe(false);
    request.counts.siteWanRouters = 1;
    request.counts.coreSwitches = 1;
    expect(isScenarioRequestValid(request)).toBe(true);

    request.sites[0].octet = 254;
    expect(isScenarioRequestValid(request)).toBe(false);

    Object.assign(request, { endpointProfile: 'invalid' });
    expect(isScenarioRequestValid(request)).toBe(false);
  });

  it('matches the generator integer and UTF-8 byte limits', async () => {
    const { enterpriseScenarioRequest, isScenarioRequestValid } = await import('./scenario-client');
    const request = enterpriseScenarioRequest();

    request.counts.accessSwitches = 1.5;
    expect(isScenarioRequestValid(request)).toBe(false);
    request.counts.accessSwitches = 16;

    request.snmpCommunity = 'é'.repeat(128);
    expect(isScenarioRequestValid(request)).toBe(false);
    request.snmpCommunity = 'NetAllyDemo';

    request.attachmentName = 'é'.repeat(33);
    expect(isScenarioRequestValid(request)).toBe(false);
    request.attachmentName = 'cyberscope';

    request.sites[0].location = 'é'.repeat(65);
    expect(isScenarioRequestValid(request)).toBe(false);
  });
});
