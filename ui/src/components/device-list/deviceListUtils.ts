import type { Device } from '../../api/types';

/**
 * Protocol detection rules for devices.
 * Each rule has a label and a function to check if the protocol is enabled.
 */
export const PROTOCOL_RULES = [
  { label: 'SNMP', isEnabled: (device: Device) => Boolean(device.snmpAgent) },
  {
    label: 'LLDP',
    isEnabled: (device: Device) => Boolean(device.lldp?.enabled),
  },
  { label: 'CDP', isEnabled: (device: Device) => Boolean(device.cdp?.enabled) },
  { label: 'STP', isEnabled: (device: Device) => Boolean(device.stp?.enabled) },
  { label: 'DHCP', isEnabled: (device: Device) => Boolean(device.dhcp) },
  { label: 'DNS', isEnabled: (device: Device) => Boolean(device.dns) },
  {
    label: 'HTTP',
    isEnabled: (device: Device) => Boolean(device.http?.enabled),
  },
  { label: 'FTP', isEnabled: (device: Device) => Boolean(device.ftp?.enabled) },
  {
    label: 'NetBIOS',
    isEnabled: (device: Device) => Boolean(device.netbios?.enabled),
  },
] as const;

/**
 * Gets the list of enabled protocol labels for a device.
 */
export const getDeviceProtocols = (device: Device): string[] => {
  const protos: string[] = [];
  for (const rule of PROTOCOL_RULES) {
    if (rule.isEnabled(device)) {
      protos.push(rule.label);
    }
  }
  return protos;
};

/**
 * Checks if a device matches a search query.
 * Searches hostname, MAC, IP addresses, and device type.
 */
export const matchesSearchQuery = (device: Device, query: string): boolean => {
  const normalized = query.trim().toLowerCase();
  if (!normalized) {
    return true;
  }
  return Boolean(
    device.hostname.toLowerCase().includes(normalized) ||
      device.mac.toLowerCase().includes(normalized) ||
      device.ip?.toLowerCase().includes(normalized) ||
      device.ips?.some((ip) => ip.toLowerCase().includes(normalized)) ||
      device.type?.toLowerCase().includes(normalized),
  );
};

/**
 * Checks if a device matches a protocol filter.
 */
export const matchesProtocolFilter = (device: Device, filter: string): boolean => {
  if (filter === 'all') {
    return true;
  }
  const rule = PROTOCOL_RULES.find((entry) => entry.label === filter);
  return rule ? rule.isEnabled(device) : false;
};
