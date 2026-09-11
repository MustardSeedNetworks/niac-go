import { describe, expect, it } from 'vitest';
import type { AttachmentPolicy } from '../../api/fabric-types';
import { bindingOptions } from './attachment-options';

const policies: AttachmentPolicy[] = [
  { interface: 'eth0', mode: 'access', accessVlan: 200 },
  { interface: 'eth0', mode: 'trunk', allowedVlans: [210, 200] },
  { interface: 'eth1', mode: 'direct' },
];

describe('bindingOptions', () => {
  it('offers only the modes the operator approved for that interface', () => {
    expect(bindingOptions(policies, 'eth0').modes).toEqual(['access', 'trunk']);
    expect(bindingOptions(policies, 'eth1').modes).toEqual(['direct']);
  });

  it('offers the access VLAN and the trunk set separately, in VLAN order', () => {
    const options = bindingOptions(policies, 'eth0');

    expect(options.vlansByMode.access).toEqual([200]);
    expect(options.vlansByMode.trunk).toEqual([200, 210]);
  });

  // Direct mode carries no VLAN at all — the compiler rejects a direct binding
  // that names one (CodeInvalidAccessVLAN), so the picker must not offer one.
  it('offers no VLAN for direct mode', () => {
    expect(bindingOptions(policies, 'eth1').vlansByMode.direct).toEqual([]);
  });

  it('offers nothing for an interface no policy names', () => {
    expect(bindingOptions(policies, 'eth9').modes).toEqual([]);
  });
});
