import { stringify as stringifyYaml } from 'yaml';
import type { Device } from '../../api/types';

/**
 * The value for an optional block: its populated fields, or null when there
 * are none.
 *
 * An empty block is emitted as a bare key -- `dhcp:` -- rather than `dhcp: {}`,
 * which is what the daemon reads as "present, defaulted".
 */
const block = (fields: Record<string, unknown>): Record<string, unknown> | null =>
  Object.keys(fields).length > 0 ? fields : null;

const baseDeviceFields = (device: Device): Record<string, unknown> => {
  const out: Record<string, unknown> = {
    hostname: device.hostname,
    mac: device.mac,
  };
  if (device.type) out.type = device.type;
  if (device.ip) out.ip = device.ip;
  if (device.ips && device.ips.length > 0) out.ips = device.ips;
  return out;
};

const interfaceFields = (device: Device): Record<string, unknown>[] | undefined => {
  if (!device.interfaceDetails || device.interfaceDetails.length === 0) {
    return undefined;
  }
  return device.interfaceDetails.map((iface) => {
    const out: Record<string, unknown> = { name: iface.name };
    if (iface.speed !== undefined && iface.speed > 0) out.speed = iface.speed;
    if (iface.duplex) out.duplex = iface.duplex;
    if (iface.adminStatus) out.adminStatus = iface.adminStatus;
    if (iface.operStatus) out.operStatus = iface.operStatus;
    if (iface.description) out.description = iface.description;
    return out;
  });
};

const snmpFields = (device: Device): Record<string, unknown> | null | undefined => {
  const snmp = device.snmpAgent;
  if (!snmp) return undefined;

  const out: Record<string, unknown> = {};
  if (snmp.community) out.community = snmp.community;
  if (snmp.sysName) out.sysName = snmp.sysName;
  if (snmp.walkFile) out.walkFile = snmp.walkFile;
  if (snmp.walkFiles && snmp.walkFiles.length > 0) out.walkFiles = snmp.walkFiles;
  if (snmp.addMibs && snmp.addMibs.length > 0) {
    out.addMibs = snmp.addMibs.map((mib) => ({
      oid: mib.oid,
      type: mib.type,
      value: mib.value,
    }));
  }
  return block(out);
};

const dhcpFields = (device: Device): Record<string, unknown> | null | undefined => {
  const dhcp = device.dhcp;
  if (!dhcp) return undefined;

  const out: Record<string, unknown> = {};
  if (dhcp.subnetMask) out.subnetMask = dhcp.subnetMask;
  if (dhcp.router) out.router = dhcp.router;
  if (dhcp.domainNameServer) out.domainNameServer = dhcp.domainNameServer;
  return block(out);
};

const dnsFields = (device: Device): Record<string, unknown> | null | undefined => {
  const dns = device.dns;
  if (!dns) return undefined;

  const out: Record<string, unknown> = {};
  if (dns.forwardRecords && dns.forwardRecords.length > 0) {
    out.forwardRecords = dns.forwardRecords.map((record) => ({
      name: record.name,
      ip: record.ip,
    }));
  }
  return block(out);
};

const trafficFields = (device: Device): Record<string, unknown> | undefined => {
  if (!device.traffic?.enabled) return undefined;

  const out: Record<string, unknown> = { enabled: true };
  if (device.traffic.arpAnnouncements?.enabled) {
    out.arpAnnouncements = { enabled: true };
  }
  return out;
};

/**
 * Assemble the device document in the order the preview has always shown.
 *
 * Key order is insertion order here, and `yaml` preserves it, so this reads
 * top-to-bottom as the rendered preview does.
 */
const buildDeviceDocument = (device: Device): Record<string, unknown> => {
  const out = baseDeviceFields(device);

  const interfaces = interfaceFields(device);
  if (interfaces) out.interfaces = interfaces;

  const snmp = snmpFields(device);
  if (snmp !== undefined) out.snmpAgent = snmp;

  if (device.lldp?.enabled) {
    out.lldp = {
      enabled: true,
      ...(device.lldp.systemDescription
        ? { systemDescription: device.lldp.systemDescription }
        : {}),
    };
  }
  if (device.cdp?.enabled) {
    out.cdp = {
      enabled: true,
      ...(device.cdp.platform ? { platform: device.cdp.platform } : {}),
    };
  }
  if (device.stp?.enabled) {
    out.stp = {
      enabled: true,
      ...(device.stp.bridgePriority !== undefined
        ? { bridgePriority: device.stp.bridgePriority }
        : {}),
    };
  }

  const dhcp = dhcpFields(device);
  if (dhcp !== undefined) out.dhcp = dhcp;

  const dns = dnsFields(device);
  if (dns !== undefined) out.dns = dns;

  if (device.http?.enabled) {
    out.http = {
      enabled: true,
      ...(device.http.serverName ? { serverName: device.http.serverName } : {}),
    };
  }
  if (device.ftp?.enabled) {
    out.ftp = {
      enabled: true,
      ...(device.ftp.welcomeBanner ? { welcomeBanner: device.ftp.welcomeBanner } : {}),
    };
  }
  if (device.netbios?.enabled) {
    out.netbios = {
      enabled: true,
      ...(device.netbios.name ? { name: device.netbios.name } : {}),
      ...(device.netbios.workgroup ? { workgroup: device.netbios.workgroup } : {}),
    };
  }

  const traffic = trafficFields(device);
  if (traffic) out.traffic = traffic;

  return out;
};

/**
 * Build a YAML preview string from a Device object.
 *
 * Serialised with the `yaml` package rather than hand-built, so a value
 * containing a quote, a backslash or a newline is escaped by the same code
 * that writes the loadable export. Concatenating it produced a preview that
 * would not parse -- `hostname: "a"b"` -- for values a user can type.
 */
export const buildYamlPreview = (device: Device): string => {
  try {
    // lineWidth: 0 disables folding. A wrapped scalar is still valid YAML but
    // reads as a mangled value in a preview pane.
    return stringifyYaml({ devices: [buildDeviceDocument(device)] }, { lineWidth: 0 }).trimEnd();
  } catch {
    return '# Error generating YAML preview';
  }
};
