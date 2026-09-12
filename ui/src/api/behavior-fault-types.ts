import * as v from 'valibot';

export const interfaceBehaviorFaultTypes = [
  'fcs_errors',
  'packet_discards',
  'interface_errors',
  'high_utilization',
  'link_down',
  'poe_loss',
  'duplicate_ip',
  'bad_mask',
] as const;
export const resourceBehaviorFaultTypes = [
  'cpu_percent',
  'memory_percent',
  'disk_percent',
] as const;
export const deviceBehaviorFaultTypes = [
  'duplicate_dhcp_offer',
  'captive_portal',
  'dhcp_no_offer',
  'dns_nxdomain',
  'dns_timeout',
  'latency',
  ...resourceBehaviorFaultTypes,
] as const;

type InterfaceFaultType = (typeof interfaceBehaviorFaultTypes)[number];
type DeviceFaultType = (typeof deviceBehaviorFaultTypes)[number];
type ResourceFaultType = (typeof resourceBehaviorFaultTypes)[number];

export type DraftBehaviorFault = { device: string } & (
  | { type: 'bad_mask'; prefixBits: number; interface: string; value?: never; address?: never }
  | {
      type: 'duplicate_dhcp_offer';
      address: string;
      value?: never;
      interface?: never;
      prefixBits?: never;
    }
  | { type: 'duplicate_ip'; address: string; value?: never; interface: string; prefixBits?: never }
  | ({ value: number; address?: never; prefixBits?: never } & (
      | { type: Exclude<InterfaceFaultType, 'duplicate_ip' | 'bad_mask'>; interface: string }
      | { type: Exclude<DeviceFaultType, 'duplicate_dhcp_offer'>; interface?: never }
    ))
);

export const isDeviceBehaviorFaultType = (type: string): type is DeviceFaultType =>
  deviceBehaviorFaultTypes.some((candidate) => candidate === type);

export const isResourceBehaviorFaultType = (type: string): type is ResourceFaultType =>
  resourceBehaviorFaultTypes.some((candidate) => candidate === type);

export const isInterfaceBehaviorFaultType = (type: string): type is InterfaceFaultType =>
  interfaceBehaviorFaultTypes.some((candidate) => candidate === type);

export const behaviorFaultMaximum = (
  type: Exclude<DraftBehaviorFault['type'], 'duplicate_dhcp_offer' | 'duplicate_ip' | 'bad_mask'>,
) => (type === 'captive_portal' ? 1 : type === 'latency' ? 60000 : 100);

const faultIPv4 = v.pipe(v.string(), v.ipv4());
export const validFaultAddress = (address: string) => {
  if (!v.safeParse(faultIPv4, address).success) return false;
  const first = Number(address.split('.')[0]);
  return (first < 224 || first > 239) && address !== '0.0.0.0' && address !== '255.255.255.255';
};

export const validBehaviorFault = (fault: DraftBehaviorFault) => {
  if (!fault.device) return false;
  if (fault.type === 'bad_mask')
    return (
      fault.value === undefined &&
      fault.address === undefined &&
      Boolean(fault.interface) &&
      Number.isInteger(fault.prefixBits) &&
      fault.prefixBits >= 0 &&
      fault.prefixBits <= 32
    );
  if (fault.type === 'duplicate_ip')
    return (
      fault.value === undefined && Boolean(fault.interface) && validFaultAddress(fault.address)
    );
  if (fault.type === 'duplicate_dhcp_offer')
    return (
      fault.value === undefined && fault.interface === undefined && validFaultAddress(fault.address)
    );
  return (
    fault.address === undefined &&
    Boolean(isDeviceBehaviorFaultType(fault.type) || fault.interface) &&
    Number.isInteger(fault.value) &&
    fault.value >= 1 &&
    fault.value <= behaviorFaultMaximum(fault.type)
  );
};
