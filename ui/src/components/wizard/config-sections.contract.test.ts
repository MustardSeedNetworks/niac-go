/**
 * The cross-language half of the config-section contract.
 *
 * The wizard's Networks step and fleet-defaults editor compose YAML in
 * TypeScript that only the Go loader ever has to accept. A round trip through
 * one YAML library proves nothing about that: the leg that can break is
 * TS-serialize -> Go strict parse, and the strict decoder rejects a key this
 * side spells differently.
 *
 * This test asserts the UI still produces the committed fixture; a Go test in
 * internal/converter asserts the daemon still parses it.
 *
 * Regenerate deliberately with UPDATE_CONTRACT=1 when the mapping changes on
 * purpose -- then run the Go test, which is what says the daemon agrees.
 */

import { readFileSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { describe, expect, it } from 'vitest';
import { spliceConfigSection } from '../../utils/config-section';
import { setDeviceAddress } from './device-addressing';
import { serializeCapturePlaybacks, serializeDiscoveryProtocols } from './fleet-defaults';
import { serializeAttachments, serializeNetworks } from './network-addressing';

const FIXTURE = join(
  import.meta.dirname,
  '../../../../internal/converter/testdata/ui_config_contract.yaml',
);

/** Every section the wizard composes, so a key the Go side never checks does
 * not exist. Deliberately maximal rather than realistic. */
function buildContractConfig(): string {
  let config = `devices:
  - name: contract-rtr-01
    type: router
    mac: "00:11:22:33:44:55"
`;

  config = spliceConfigSection(
    config,
    'networks',
    serializeNetworks([
      { name: 'contract-lan', subnet: '10.20.0.0/24' },
      { name: 'contract-mgmt', subnet: '10.20.99.0/24', virtualVlan: 99 },
    ]),
  );
  config = spliceConfigSection(
    config,
    'attachments',
    serializeAttachments([{ name: 'tester', connect: 'contract-lan' }]),
  );
  config = spliceConfigSection(
    config,
    'discovery_protocols',
    serializeDiscoveryProtocols({
      lldp: { enabled: true, interval: 30 },
      cdp: { enabled: true },
    }),
  );
  config = spliceConfigSection(
    config,
    'capture_playbacks',
    serializeCapturePlaybacks([{ fileName: 'contract.pcap', loopTime: 60, scaleTime: 0.5 }]),
  );

  return setDeviceAddress(config, 'contract-rtr-01', 'contract-lan', '10.20.0.1/24');
}

describe('config-section contract', () => {
  it('still produces the fixture the daemon parses', () => {
    const produced = buildContractConfig();

    if (process.env.UPDATE_CONTRACT === '1') {
      writeFileSync(FIXTURE, produced);
    }

    expect(produced).toBe(readFileSync(FIXTURE, 'utf8'));
  });
});
