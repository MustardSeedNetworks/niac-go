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

export type DraftBehaviorFault = { device: string; value: number } & (
  | { type: InterfaceFaultType; interface: string }
  | { type: DeviceFaultType; interface?: never }
);

export const isDeviceBehaviorFaultType = (type: string): type is DeviceFaultType =>
  deviceBehaviorFaultTypes.some((candidate) => candidate === type);

export const isResourceBehaviorFaultType = (type: string): type is ResourceFaultType =>
  resourceBehaviorFaultTypes.some((candidate) => candidate === type);

export const isInterfaceBehaviorFaultType = (type: string): type is InterfaceFaultType =>
  interfaceBehaviorFaultTypes.some((candidate) => candidate === type);

export const behaviorFaultMaximum = (type: DraftBehaviorFault['type']) =>
  type === 'captive_portal' ? 1 : type === 'latency' ? 60000 : 100;

export const validBehaviorFault = (fault: DraftBehaviorFault) =>
  Boolean(fault.device && (isDeviceBehaviorFaultType(fault.type) || fault.interface)) &&
  Number.isInteger(fault.value) &&
  fault.value >= 1 &&
  fault.value <= behaviorFaultMaximum(fault.type);
