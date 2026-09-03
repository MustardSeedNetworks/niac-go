import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { describe, expect, it } from 'vitest';
import { parse as parseYaml } from 'yaml';
import type { AuthoredDevice } from '../components/device-editor/generated/authored-device.generated';
import { DEVICE_SECTIONS } from '../components/device-editor/generated/sections.generated';
import { parseAuthoredDevice, serializeAuthoredDevice } from './authored-device-yaml';

/**
 * P1b-2's acceptance: the device editor must be able to author everything the
 * daemon can run, and prove it by round-tripping the clinic scenario with zero
 * diff. The editor's document model is the authored YAML itself, so this test
 * is what makes that claim checkable rather than asserted.
 */

const clinic = parseYaml(
  readFileSync(join(__dirname, '__fixtures__', 'clinic-scenario.yaml'), 'utf8'),
) as { devices: AuthoredDevice[] };

describe('authored device YAML', () => {
  it.each(clinic.devices.map((device) => [device.name ?? '(unnamed)', device] as const))(
    'round-trips %s with zero diff',
    (_name, device) => {
      const roundTripped = parseAuthoredDevice(serializeAuthoredDevice(device));

      expect(roundTripped).toEqual(device);
    },
  );

  it('renders every authored field of every clinic device from the manifest', () => {
    const known = new Set(
      DEVICE_SECTIONS.flatMap((s) => (s.key === 'device' ? s.fields.map((f) => f.name) : [s.key])),
    );
    // The identity controls stay hand-bound and are deliberately absent from
    // the generated manifest; everything else must be a section or a field.
    const handBound = new Set(['name', 'type', 'mac', 'ips']);

    const unrenderable = clinic.devices.flatMap((device) =>
      Object.keys(device).filter((key) => !known.has(key) && !handBound.has(key)),
    );

    expect(unrenderable).toEqual([]);
  });

  it('prunes cleared values rather than writing them as empty', () => {
    const serialized = serializeAuthoredDevice({
      name: 'edge-01',
      vendor: '',
      vlan: undefined,
      // `false` and `0` are values an author chose, not absences.
      babble: false,
      ttl: { ttl: 0 },
    } as AuthoredDevice);

    expect(parseAuthoredDevice(serialized)).toEqual({
      name: 'edge-01',
      babble: false,
      ttl: { ttl: 0 },
    });
  });
});
