import * as v from 'valibot';

export const interfaceBehaviorFaultTypes = [
  'fcs_errors',
  'packet_discards',
  'interface_errors',
  'high_utilization',
  'link_down',
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
  | { type: 'duplicate_dhcp_offer'; address: string; value?: never; interface?: never }
  | ({ value: number; address?: never } & (
      | { type: InterfaceFaultType; interface: string }
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
  type: Exclude<DraftBehaviorFault['type'], 'duplicate_dhcp_offer'>,
) => (type === 'captive_portal' ? 1 : type === 'latency' ? 60000 : 100);

const faultIPv4 = v.pipe(v.string(), v.ipv4());
export const validFaultAddress = (address: string) => {
  if (!v.safeParse(faultIPv4, address).success) return false;
  const first = Number(address.split('.')[0]);
  return (first < 224 || first > 239) && address !== '0.0.0.0' && address !== '255.255.255.255';
};

export const validBehaviorFault = (fault: DraftBehaviorFault) => {
  if (!fault.device) return false;
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
