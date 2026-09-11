import type { AttachmentMode, AttachmentPolicy } from '../../api/fabric-types';

/**
 * Binding choices derived from the operator's approved policies for one host
 * interface. The daemon refuses any start outside them, so the picker offers
 * these instead of free input (AP-0).
 */
export interface BindingOptions {
  modes: AttachmentMode[];
  vlansByMode: Record<AttachmentMode, number[]>;
}

const emptyOptions = (): BindingOptions => ({
  modes: [],
  vlansByMode: { direct: [], access: [], trunk: [] },
});

/**
 * bindingOptions narrows the policy list to one interface. Modes keep the
 * operator's declared order so the picker reads the way the policy flags were
 * written. A trunk policy approves a VLAN set; every other mode approves one.
 */
export const bindingOptions = (
  policies: readonly AttachmentPolicy[],
  interfaceName: string,
): BindingOptions => {
  const options = emptyOptions();
  for (const policy of policies) {
    if (policy.interface !== interfaceName) continue;
    if (!options.modes.includes(policy.mode)) options.modes.push(policy.mode);
    const vlans = policy.mode === 'trunk' ? (policy.allowedVlans ?? []) : [policy.accessVlan ?? 0];
    for (const vlan of vlans) {
      if (vlan > 0 && !options.vlansByMode[policy.mode].includes(vlan)) {
        options.vlansByMode[policy.mode].push(vlan);
      }
    }
  }
  for (const mode of options.modes) {
    options.vlansByMode[mode].sort((a, b) => a - b);
  }
  return options;
};
