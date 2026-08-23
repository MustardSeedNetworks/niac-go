import type { Device } from '../../api/types';

/**
 * Protocols a device speaks, as reported by the API.
 *
 * This used to re-derive the list by probing per-protocol sub-objects
 * (`snmpAgent`, `lldp`, …), but `GET /api/v1/config/devices` returns a summary
 * shape that carries none of them — the union of keys across a real response is
 * hostname · interfaceDetails · interfaces · ips · mac · protocols · type · vlan.
 * Every predicate was therefore `Boolean(undefined)`: the column read "No
 * protocols" for every device, the CSV export shipped an empty column, and the
 * filter dropdown had a single option and could never match (D9).
 *
 * The server already computes this. Order is the server's; it is meaningful and
 * consistent across devices, so it is not re-sorted here.
 */
export const getDeviceProtocols = (device: Device): string[] =>
  Array.isArray(device.protocols) ? device.protocols : [];

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

export const matchesProtocolFilter = (device: Device, filter: string): boolean => {
  if (filter === 'all') {
    return true;
  }

  return getDeviceProtocols(device).some(
    (protocol) => protocol.toUpperCase() === filter.toUpperCase(),
  );
};
