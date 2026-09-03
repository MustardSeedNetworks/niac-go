/**
 * The cross-language half of the authored-device contract.
 *
 * The device editor's round trip is only an identity if both ends agree on the
 * document. `device-yaml.test.ts` proves the TS end parses back what it
 * stringified — one library, both directions, which cannot catch a disagreement
 * between two YAML implementations. The leg that can actually break is
 * TS-serialize → Go strict parse: MAC- and IP-shaped bare scalars, sexagesimal
 * lookalikes and empty containers are all places where the `yaml` package and
 * `gopkg.in/yaml.v3` could differ on quoting or type.
 *
 * So this test writes what the editor would POST as `rawYaml` for each clinic
 * device, and `internal/api/devices_authored_contract_test.go` loads those same
 * files through the real save path. One set of files, asserted from both sides.
 *
 * Regenerate deliberately with UPDATE_CONTRACT=1, then run the Go test — that
 * is the half which says the daemon agrees.
 */

import { readdirSync, readFileSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { describe, expect, it } from 'vitest';
import { parse as parseYaml } from 'yaml';
import type { AuthoredDevice } from '../components/device-editor/generated/authored-device.generated';
import { serializeAuthoredDevice } from './authored-device-yaml';

const FIXTURE_DIR = join(import.meta.dirname, '../../../internal/api/testdata/authored_devices');

const clinic = parseYaml(
  readFileSync(join(import.meta.dirname, '__fixtures__', 'clinic-scenario.yaml'), 'utf8'),
) as { devices: AuthoredDevice[] };

const update = process.env.UPDATE_CONTRACT === '1';

/**
 * Scalar shapes the clinic scenario happens not to contain.
 *
 * Every clinic MAC carries a hex letter, so none of them is ambiguous. A
 * digits-only MAC is: under YAML 1.1 it resolves as a base-60 integer, and the
 * two implementations do not have to agree about that. Same for values that
 * look boolean or numeric but are authored as strings. Named as a device so it
 * travels the identical path as a real one.
 */
const scalarShapes: AuthoredDevice = {
  name: 'shape-probe',
  type: 'switch',
  mac: '00:11:22:33:44:55',
  ips: ['10.0.0.7', '2001:db8::7'],
  vlan: 10,
  properties: {
    sexagesimal: '00:11:22',
    booleanish: 'on',
    leading_zero: '0700',
    version: '1.10',
    multiline: 'line one\nline two',
    quoted: 'has "quotes" and a: colon',
  },
  ttl: { ttl: 0 },
  babble: false,
};

const cases = [...clinic.devices, scalarShapes];

describe('authored device contract', () => {
  it.each(cases.map((device) => [device.name ?? '(unnamed)', device] as const))(
    'emits the fixture the daemon parses for %s',
    (name, device) => {
      const produced = `${serializeAuthoredDevice(device)}\n`;
      const fixture = join(FIXTURE_DIR, `${name}.yaml`);

      if (update) {
        writeFileSync(fixture, produced);
      }

      expect(produced).toBe(readFileSync(fixture, 'utf8'));
    },
  );

  it('leaves no stale fixture behind', () => {
    const byName = (a: string, b: string) => a.localeCompare(b);
    const expected = cases.map((device) => `${device.name}.yaml`).sort(byName);

    expect(readdirSync(FIXTURE_DIR).sort(byName)).toEqual(expected);
  });
});
