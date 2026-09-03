/**
 * ADR-0007 wire-casing contract.
 *
 * The API speaks camelCase in both directions, with no exceptions
 * (docs/adr/0007-json-wire-casing-convention.md). Every request struct in
 * internal/api decodes with DisallowUnknownFields(), so a single mis-cased key
 * is a hard 400 — not a warning, not a partial decode.
 *
 * These tests assert the *outgoing key names* for the payload shapes the app
 * actually sends. A fixture built from a Go struct cannot catch this class of
 * bug, because the defect lives purely in the JSON key spelling. Regression
 * D11: `requestJson` snake_cased every payload, which silently broke device
 * create/clone/update, error injection, PCAP replay and alert saving — six
 * features, one transform.
 */

import { beforeEach, describe, expect, it, vi } from 'vitest';

const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);

/**
 * Mutating requests fetch a CSRF token first, so every case here needs two
 * resolutions: the token, then the real call.
 */
function armFetch() {
  mockFetch.mockReset();
  mockFetch
    .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ token: 'csrf-token' }) })
    .mockResolvedValue({
      ok: true,
      status: 200,
      headers: { get: () => 'application/json' },
      text: async () => '{}',
      json: async () => ({}),
    });
}

/** Keys sent on the wire for the most recent fetch call, recursively. */
function sentKeys(): string[] {
  const body = mockFetch.mock.calls.at(-1)?.[1]?.body;
  if (typeof body !== 'string') return [];
  const out: string[] = [];
  const walk = (v: unknown): void => {
    if (Array.isArray(v)) {
      v.forEach(walk);
      return;
    }
    if (v && typeof v === 'object') {
      for (const [k, entry] of Object.entries(v)) {
        out.push(k);
        walk(entry);
      }
    }
  };
  walk(JSON.parse(body));
  return out;
}

describe('ADR-0007: outgoing payloads are camelCase', () => {
  beforeEach(() => {
    armFetch();
  });

  // The device's own fields moved into `rawYaml`, where snake_case is correct
  // — that string is the daemon's YAML, parsed by the config loader, not JSON.
  // The envelope around it is still a JSON DTO and still camelCase.
  it('createDevice sends a camelCase envelope around the authored YAML', async () => {
    const { createDevice } = await import('./client');
    await createDevice(
      'UITEST-RTR-01',
      'name: UITEST-RTR-01\ninterfaces:\n  - name: Gi0/0\n    admin_status: up\n',
    ).catch(() => undefined);

    const keys = sentKeys();
    expect(keys).toEqual(['hostname', 'rawYaml']);
    expect(keys).not.toContain('raw_yaml');
  });

  it('cloneDevice sends newHostname, not new_hostname', async () => {
    const client = await import('./client');
    const clone = (client as Record<string, unknown>).cloneDevice as (
      h: string,
      p: { newHostname: string },
    ) => Promise<unknown>;
    await clone('LAB-EDGE-R1', { newHostname: 'UITEST-CLONE-01' }).catch(() => undefined);

    expect(sentKeys()).toContain('newHostname');
    expect(sentKeys()).not.toContain('new_hostname');
  });

  it('startReplay sends loopMs, not loop_ms', async () => {
    const client = await import('./client');
    const start = (client as Record<string, unknown>).startReplay as (
      p: unknown,
    ) => Promise<unknown>;
    await start({ file: 'capture.pcap', loopMs: 0, scale: 1 }).catch(() => undefined);

    expect(sentKeys()).toContain('loopMs');
    expect(sentKeys()).not.toContain('loop_ms');
  });

  it('updateAlerts sends packetsThreshold and webhookUrl', async () => {
    const client = await import('./client');
    const update = (client as Record<string, unknown>).updateAlerts as (
      p: unknown,
    ) => Promise<unknown>;
    await update({ packetsThreshold: 987654, webhookUrl: 'https://example.test/hook' }).catch(
      () => undefined,
    );

    const keys = sentKeys();
    expect(keys).toContain('packetsThreshold');
    expect(keys).toContain('webhookUrl');
    expect(keys).not.toContain('packets_threshold');
    expect(keys).not.toContain('webhook_url');
  });

  it('draft topology mutations keep macSuffix / sysObjectId / profileRole camelCase', async () => {
    const lib = await import('./library-client');
    const mutate = (lib as Record<string, unknown>).mutateScenarioDraftTopology as (
      n: string,
      r: string,
      m: unknown,
    ) => Promise<unknown>;
    await mutate('draft-1', 'rev-1', {
      operation: 'add_device',
      device: {
        name: 'SW-1',
        type: 'switch',
        macSuffix: 12,
        sysObjectId: '1.3.6.1.4.1.9.1.1',
        profileRole: 'access',
      },
    }).catch(() => undefined);

    const keys = sentKeys();
    expect(keys).toContain('macSuffix');
    expect(keys).toContain('sysObjectId');
    expect(keys).toContain('profileRole');
    expect(keys).not.toContain('mac_suffix');
    expect(keys).not.toContain('sys_object_id');
    expect(keys).not.toContain('profile_role');
  });
});
