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
                accessVlan: 2,
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
    expect(report.topology.binding.accessVlan).toBe(2);
    expect(mockFetch.mock.calls[1][0]).toContain('/api/v1/simulation/preflight');
  });
});
