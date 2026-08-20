import { beforeEach, describe, expect, it, vi } from 'vitest';

const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);

describe('simulation preflight client', () => {
  beforeEach(() => {
    mockFetch.mockReset();
  });

  it('posts the deployment binding and returns the typed safety report', async () => {
    mockFetch
      .mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ token: 'csrf-token' }),
      })
      .mockResolvedValueOnce({
        ok: true,
        json: () =>
          Promise.resolve({
            safe: true,
            topology: {
              binding: {
                attachment: 'tester',
                interface: 'eth0',
                mode: 'access',
                physicalVlan: 2,
                network: 'lab-access',
                wireTagged: false,
              },
              networks: [],
              interfaces: [],
              routes: [],
              dhcpScopes: [],
            },
            diagnostics: [],
          }),
      });

    const { preflightSimulation } = await import('./client');
    const report = await preflightSimulation({
      interface: 'eth0',
      configData: 'devices: []',
      attachment: 'tester',
      attachmentMode: 'access',
      accessVlan: 2,
    });

    expect(report.safe).toBe(true);
    expect(report.topology.binding.physicalVlan).toBe(2);
    expect(mockFetch.mock.calls[1]?.[0]).toContain('/api/v1/simulation/preflight');
    const body = JSON.parse(mockFetch.mock.calls[1]?.[1].body as string);
    expect(body).toMatchObject({
      configData: 'devices: []',
      attachmentMode: 'access',
      accessVlan: 2,
    });
    expect(body).not.toHaveProperty('config_data');
    expect(body).not.toHaveProperty('dedicated');
  });

  it('posts simulation starts with the API camel-case contract', async () => {
    mockFetch
      .mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ token: 'csrf-token' }),
      })
      .mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ running: true, deviceCount: 1, packets: 0 }),
      });

    const { startSimulation } = await import('./client');
    await startSimulation({
      interface: 'eth0',
      configData: 'devices: []',
      attachment: 'tester',
      attachmentMode: 'access',
      accessVlan: 200,
    });

    const body = JSON.parse(mockFetch.mock.calls.at(-1)?.[1].body as string);
    expect(body).toMatchObject({
      configData: 'devices: []',
      attachmentMode: 'access',
      accessVlan: 200,
    });
    expect(body).not.toHaveProperty('config_data');
  });
});
