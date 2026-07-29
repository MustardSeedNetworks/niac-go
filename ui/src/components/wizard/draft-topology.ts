import { parse } from 'yaml';
import type { DeviceSummary, TopologyLink } from '../../api/types';

export interface DraftInterface {
  name: string;
  type: string;
  speed: number;
  occupied: boolean;
}

export interface DraftTopologyModel {
  devices: DeviceSummary[];
  links: TopologyLink[];
  interfaces: Record<string, DraftInterface[]>;
  positions: Record<string, { x: number; y: number }>;
  segmentByDevice: Record<string, number>;
  segmented: boolean;
  configBacked: boolean;
}

type UnknownRecord = Record<string, unknown>;

const asRecord = (value: unknown): UnknownRecord | null =>
  typeof value === 'object' && value !== null && !Array.isArray(value)
    ? (value as UnknownRecord)
    : null;
const asString = (value: unknown) => (typeof value === 'string' ? value : '');
const asNumber = (value: unknown) =>
  typeof value === 'number' && Number.isFinite(value) ? value : 0;
const asBoolean = (value: unknown) => value === true;
const asStrings = (value: unknown) =>
  Array.isArray(value) ? value.filter((item): item is string => typeof item === 'string') : [];
const asNumbers = (value: unknown) =>
  Array.isArray(value) ? value.filter((item): item is number => Number.isInteger(item)) : [];

function records(value: unknown): UnknownRecord[] {
  if (!Array.isArray(value)) throw new Error('Draft devices must be an array');
  return value.map((device) => {
    const record = asRecord(device);
    if (!record) throw new Error('Draft device entries must be objects');
    return record;
  });
}

function deviceRecords(document: UnknownRecord): {
  records: UnknownRecord[];
  segmentByDevice: Record<string, number>;
  segmented: boolean;
  configBacked: boolean;
} {
  if ('segments' in document && document.segments !== null && !Array.isArray(document.segments))
    throw new Error('Draft segments must be an array');
  if (Array.isArray(document.segments) && document.segments.length > 0) {
    const segmentByDevice: Record<string, number> = {};
    const configBacked = document.segments.some(
      (segment) => asString(asRecord(segment)?.config) !== '',
    );
    return {
      records: document.segments.flatMap((segment, segmentIndex) => {
        const devices = asRecord(segment)?.devices;
        const segmentDevices = Array.isArray(devices) ? records(devices) : [];
        for (const device of segmentDevices) {
          const name = asString(device.name);
          if (name) segmentByDevice[name] = segmentIndex;
        }
        return segmentDevices;
      }),
      segmentByDevice,
      segmented: true,
      configBacked,
    };
  }
  if ('devices' in document) {
    if (document.devices === null)
      return { records: [], segmentByDevice: {}, segmented: false, configBacked: false };
    if (!Array.isArray(document.devices)) throw new Error('Draft devices must be an array');
    return {
      records: records(document.devices),
      segmentByDevice: {},
      segmented: false,
      configBacked: false,
    };
  }
  return { records: [], segmentByDevice: {}, segmented: false, configBacked: false };
}

function endpointKey(device: string, iface: string, remoteDevice: string, remoteInterface: string) {
  return [`${device}|${iface}`, `${remoteDevice}|${remoteInterface}`].sort().join('|');
}

function directedEndpointKey(
  device: string,
  iface: string,
  remoteDevice: string,
  remoteInterface: string,
) {
  return `${device}|${iface}>${remoteDevice}|${remoteInterface}`;
}

function deviceProtocols(device: UnknownRecord) {
  const fields = [
    ['snmp_agent', 'SNMP'],
    ['lldp', 'LLDP'],
    ['cdp', 'CDP'],
    ['dhcp', 'DHCP'],
    ['dns', 'DNS'],
    ['http', 'HTTP'],
    ['ssh', 'SSH'],
  ] as const;
  return fields.filter(([field]) => asRecord(device[field]) !== null).map(([, label]) => label);
}

function declaredTopology(devices: UnknownRecord[]) {
  const links = new Set<string>();
  const trunkByDirection = new Map<string, UnknownRecord>();
  const occupied = new Set<string>();
  for (const device of devices) {
    const name = asString(device.name);
    const deviceTrunks = Array.isArray(device.trunk_ports) ? device.trunk_ports : [];
    for (const trunkValue of deviceTrunks) {
      const trunk = asRecord(trunkValue);
      if (!trunk) continue;
      const localInterface = asString(trunk.interface);
      const remoteDevice = asString(trunk.remote_device);
      const remoteInterface = asString(trunk.remote_interface);
      const key = directedEndpointKey(name, localInterface, remoteDevice, remoteInterface);
      links.add(key);
      trunkByDirection.set(key, trunk);
      occupied.add(`${name}|${localInterface}`);
      occupied.add(`${remoteDevice}|${remoteInterface}`);
    }
  }
  return { links, trunks: trunkByDirection, occupied };
}

function sameLinkProperties(left: UnknownRecord, right: UnknownRecord) {
  const leftVLANs = [...asNumbers(left.vlans)].sort((a, b) => a - b);
  const rightVLANs = [...asNumbers(right.vlans)].sort((a, b) => a - b);
  return (
    leftVLANs.length === rightVLANs.length &&
    leftVLANs.every((vlan, index) => vlan === rightVLANs[index]) &&
    asNumber(left.native_vlan) === asNumber(right.native_vlan) &&
    asBoolean(left.fdb_only) === asBoolean(right.fdb_only)
  );
}

export function parseDraftTopology(content: string): DraftTopologyModel {
  const document = asRecord(parse(content));
  if (!document) throw new Error('Draft YAML must contain an object');
  const { records, segmentByDevice, segmented, configBacked } = deviceRecords(document);
  const devices: DeviceSummary[] = [];
  const links: TopologyLink[] = [];
  const interfaces: Record<string, DraftInterface[]> = {};
  const positions: Record<string, { x: number; y: number }> = {};
  const seenLinks = new Set<string>();
  const {
    links: declaredLinks,
    trunks: declaredTrunks,
    occupied: occupiedEndpoints,
  } = declaredTopology(records);

  for (const device of records) {
    const name = asString(device.name);
    if (!name) throw new Error('Every draft device requires a name');
    const properties = asRecord(device.properties) ?? {};
    const rawInterfaces = Array.isArray(device.interfaces) ? device.interfaces : [];
    const rawTrunks = Array.isArray(device.trunk_ports) ? device.trunk_ports : [];
    const interfaceRecords = rawInterfaces
      .map(asRecord)
      .filter((iface): iface is UnknownRecord => iface !== null);
    const interfaceByName = new Map(interfaceRecords.map((iface) => [asString(iface.name), iface]));

    devices.push({
      name,
      type: asString(device.type) || 'unknown',
      ips: asStrings(device.ips),
      protocols: deviceProtocols(device),
      vendor: asString(device.vendor),
      model: asString(properties.model),
      properties: Object.fromEntries(
        Object.entries(properties).filter(
          (entry): entry is [string, string] => typeof entry[1] === 'string',
        ),
      ),
    });
    interfaces[name] = interfaceRecords.map((iface) => ({
      name: asString(iface.name),
      type: asString(iface.type),
      speed: asNumber(iface.speed),
      occupied: occupiedEndpoints.has(`${name}|${asString(iface.name)}`),
    }));
    const x = Number.parseFloat(asString(properties.topology_x));
    const y = Number.parseFloat(asString(properties.topology_y));
    if (Number.isFinite(x) && Number.isFinite(y)) positions[name] = { x, y };

    for (const trunkValue of rawTrunks) {
      const trunk = asRecord(trunkValue);
      if (!trunk) continue;
      const localInterface = asString(trunk.interface);
      const remoteDevice = asString(trunk.remote_device);
      const remoteInterface = asString(trunk.remote_interface);
      if (!localInterface || !remoteDevice || !remoteInterface) continue;
      const key = endpointKey(name, localInterface, remoteDevice, remoteInterface);
      if (seenLinks.has(key)) continue;
      seenLinks.add(key);
      const iface = interfaceByName.get(localInterface);
      const vlans = asNumbers(trunk.vlans);
      links.push({
        source: name,
        target: remoteDevice,
        label: `${localInterface} ↔ ${remoteInterface}`,
        sourceInterface: localInterface,
        targetInterface: remoteInterface,
        linkType: vlans.length === 1 ? 'access' : 'trunk',
        vlans,
        nativeVlan: asNumber(trunk.native_vlan),
        fdbOnly: asBoolean(trunk.fdb_only),
        reciprocal: (() => {
          const reverseKey = directedEndpointKey(
            remoteDevice,
            remoteInterface,
            name,
            localInterface,
          );
          const reverse = declaredTrunks.get(reverseKey);
          return declaredLinks.has(reverseKey) && reverse !== undefined
            ? sameLinkProperties(trunk, reverse)
            : false;
        })(),
        speed: iface ? String(asNumber(iface.speed)) : undefined,
        duplex: iface ? asString(iface.duplex) : undefined,
        status: iface ? asString(iface.oper_status) : undefined,
        utilizationPercent: iface
          ? Math.max(asNumber(iface.in_utilization), asNumber(iface.out_utilization))
          : undefined,
      });
    }
  }
  return { devices, links, interfaces, positions, segmentByDevice, segmented, configBacked };
}
