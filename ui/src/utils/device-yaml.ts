/**
 * The one place a Device becomes the daemon's YAML shape.
 *
 * Two writers used to build this independently — the device-editor preview and
 * the YAML export — and both got it wrong, in different ways. The daemon's
 * schema (`internal/converter/types.go`) is snake_case and decodes with
 * `KnownFields(true)`, so an unknown key is a hard parse error, not a field
 * quietly dropped. A single wrong key made the whole document unloadable.
 *
 * The mapping below is the contract with that Go struct. It is pinned by
 * `internal/converter/testdata/ui_device_contract.yaml`, which this module
 * produces and a Go test parses — so a drift on either side fails a test.
 */

import type { Device } from '../api/device-config-types';

/**
 * UI device types mapped onto the daemon's `oneof` validation set.
 *
 * `unknown` is deliberately absent: the daemon does not accept it, so emitting
 * it fails validation. Omitting the key lets the rest of the device load.
 */
const DAEMON_DEVICE_TYPE: Record<string, string> = {
  router: 'router',
  switch: 'switch',
  access_point: 'access-point',
  firewall: 'firewall',
  server: 'server',
  workstation: 'workstation',
  iot: 'iot',
};

/**
 * A block's populated fields, or null when it has none.
 *
 * An empty block is emitted as a bare key — `dhcp:` — which unmarshals to a
 * nil pointer, the daemon's "present, defaulted".
 */
const block = (fields: Record<string, unknown>): Record<string, unknown> | null =>
  Object.keys(fields).length > 0 ? fields : null;

/** Copy `key` from `src` under `as` when it holds a non-empty value. */
const put = (out: Record<string, unknown>, as: string, value: unknown): void => {
  if (value === undefined || value === null || value === '') return;
  if (Array.isArray(value) && value.length === 0) return;
  out[as] = value;
};

const interfaces = (device: Device): Record<string, unknown>[] | undefined => {
  if (!device.interfaceDetails || device.interfaceDetails.length === 0) return undefined;

  return device.interfaceDetails.map((iface) => {
    const out: Record<string, unknown> = { name: iface.name };
    if (iface.speed !== undefined && iface.speed > 0) out.speed = iface.speed;
    put(out, 'duplex', iface.duplex === 'auto' ? '' : iface.duplex);
    put(out, 'admin_status', iface.adminStatus);
    put(out, 'oper_status', iface.operStatus);
    put(out, 'description', iface.description);
    return out;
  });
};

const snmpAgent = (device: Device): Record<string, unknown> | null | undefined => {
  const snmp = device.snmpAgent;
  if (!snmp) return undefined;

  const out: Record<string, unknown> = {};
  put(out, 'community', snmp.community);
  put(out, 'sysname', snmp.sysName);
  put(out, 'walk_file', snmp.walkFile);
  put(out, 'walk_files', snmp.walkFiles);
  if (snmp.addMibs && snmp.addMibs.length > 0) {
    out.add_mibs = snmp.addMibs.map((mib) => ({
      oid: mib.oid,
      type: mib.type,
      value: mib.value,
    }));
  }
  return block(out);
};

const dhcp = (device: Device): Record<string, unknown> | null | undefined => {
  if (!device.dhcp) return undefined;

  const out: Record<string, unknown> = {};
  put(out, 'subnet_mask', device.dhcp.subnetMask);
  put(out, 'router', device.dhcp.router);
  put(out, 'domain_name_server', device.dhcp.domainNameServer);
  return block(out);
};

const dns = (device: Device): Record<string, unknown> | null | undefined => {
  if (!device.dns) return undefined;

  const out: Record<string, unknown> = {};
  if (device.dns.forwardRecords && device.dns.forwardRecords.length > 0) {
    out.forward_records = device.dns.forwardRecords.map((record) => ({
      name: record.name,
      ip: record.ip,
    }));
  }
  return block(out);
};

/**
 * Convert a Device into the daemon's YAML device shape.
 *
 * Only fields the daemon's `Device` struct declares are emitted. Notably
 * absent, and deliberately so:
 *
 *   - `traffic` — there is no device-level traffic concept at all; the UI's
 *     device editor never exposed this field to begin with. The only
 *     `traffic` in the daemon's schema belongs to a behaviour phase, which
 *     is a different concept modelled elsewhere.
 *   - `ip` — the daemon has `ips` only, so a single address is folded in.
 *   - `protocols` — server-computed, not configuration.
 */
export const toDaemonDevice = (device: Device): Record<string, unknown> => {
  const out: Record<string, unknown> = {};

  put(out, 'name', device.hostname);
  if (device.type) put(out, 'type', DAEMON_DEVICE_TYPE[device.type]);
  put(out, 'mac', device.mac);

  // The daemon has no scalar `ip`; a single address is the first entry of
  // `ips`. Deduped so a device carrying both does not repeat it.
  const addresses = [...(device.ip ? [device.ip] : []), ...(device.ips ?? [])];
  put(out, 'ips', [...new Set(addresses)]);

  if (device.vlan !== undefined) put(out, 'vlan', device.vlan);
  if (device.babble) out.babble = true;
  put(out, 'map_to_ip', device.mapToIp);

  const snmp = snmpAgent(device);
  if (snmp !== undefined) out.snmp_agent = snmp;

  const dhcpBlock = dhcp(device);
  if (dhcpBlock !== undefined) out.dhcp = dhcpBlock;

  const dnsBlock = dns(device);
  if (dnsBlock !== undefined) out.dns = dnsBlock;

  if (device.lldp?.enabled) {
    const lldp: Record<string, unknown> = { enabled: true };
    put(lldp, 'system_description', device.lldp.systemDescription);
    out.lldp = lldp;
  }
  if (device.cdp?.enabled) {
    const cdp: Record<string, unknown> = { enabled: true };
    put(cdp, 'platform', device.cdp.platform);
    out.cdp = cdp;
  }
  if (device.stp?.enabled) {
    const stp: Record<string, unknown> = { enabled: true };
    if (device.stp.bridgePriority !== undefined) stp.bridge_priority = device.stp.bridgePriority;
    out.stp = stp;
  }
  if (device.http?.enabled) {
    const http: Record<string, unknown> = { enabled: true };
    put(http, 'server_name', device.http.serverName);
    out.http = http;
  }
  if (device.ftp?.enabled) {
    const ftp: Record<string, unknown> = { enabled: true };
    put(ftp, 'welcome_banner', device.ftp.welcomeBanner);
    out.ftp = ftp;
  }
  if (device.netbios?.enabled) {
    const netbios: Record<string, unknown> = { enabled: true };
    put(netbios, 'name', device.netbios.name);
    put(netbios, 'workgroup', device.netbios.workgroup);
    out.netbios = netbios;
  }

  const ifaces = interfaces(device);
  if (ifaces) out.interfaces = ifaces;

  return out;
};
