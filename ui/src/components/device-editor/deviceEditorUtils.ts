import type { Device } from '../../api/types';

/**
 * Creates an empty device with default values
 */
export const createEmptyDevice = (): Device => ({
  hostname: '',
  mac: '',
  type: 'switch',
  ip: '',
  ips: [],
});

/**
 * FIX #294, #295: Build raw YAML for the backend create/update API
 */
function appendRawSnmpLines(lines: string[], agent: NonNullable<Device['snmpAgent']>): void {
  lines.push('snmpAgent:');
  if (agent.community) lines.push(`  community: "${agent.community}"`);
  if (agent.sysName) lines.push(`  sysName: "${agent.sysName}"`);
  if (agent.sysDescr) lines.push(`  sysDescr: "${agent.sysDescr}"`);
  if (agent.sysContact) lines.push(`  sysContact: "${agent.sysContact}"`);
  if (agent.sysLocation) lines.push(`  sysLocation: "${agent.sysLocation}"`);
  if (agent.walkFile) lines.push(`  walkFile: "${agent.walkFile}"`);
  if (agent.walkFiles && agent.walkFiles.length > 0) {
    lines.push('  walkFiles:');
    for (const wf of agent.walkFiles) lines.push(`    - "${wf}"`);
  }
}

function appendNetworkProtocolLines(lines: string[], device: Device): void {
  if (device.lldp?.enabled) {
    lines.push('lldp:');
    lines.push('  enabled: true');
    if (device.lldp.systemDescription)
      lines.push(`  systemDescription: "${device.lldp.systemDescription}"`);
    if (device.lldp.ttl) lines.push(`  ttl: ${device.lldp.ttl}`);
  }
  if (device.cdp?.enabled) {
    lines.push('cdp:');
    lines.push('  enabled: true');
    if (device.cdp.platform) lines.push(`  platform: "${device.cdp.platform}"`);
    if (device.cdp.version) lines.push(`  version: ${device.cdp.version}`);
  }
  if (device.stp?.enabled) {
    lines.push('stp:');
    lines.push('  enabled: true');
    if (device.stp.bridgePriority !== undefined)
      lines.push(`  bridgePriority: ${device.stp.bridgePriority}`);
  }
}

function appendServiceLines(lines: string[], device: Device): void {
  if (device.dhcp) {
    lines.push('dhcp:');
    if (device.dhcp.subnetMask) lines.push(`  subnetMask: "${device.dhcp.subnetMask}"`);
    if (device.dhcp.router) lines.push(`  router: "${device.dhcp.router}"`);
    if (device.dhcp.dns && device.dhcp.dns.length > 0) {
      lines.push('  dns:');
      for (const d of device.dhcp.dns) lines.push(`    - "${d}"`);
    }
    if (device.dhcp.poolStart) lines.push(`  poolStart: "${device.dhcp.poolStart}"`);
    if (device.dhcp.poolEnd) lines.push(`  poolEnd: "${device.dhcp.poolEnd}"`);
  }
  if (device.dns) {
    lines.push('dns:');
    if (device.dns.forwardRecords && device.dns.forwardRecords.length > 0) {
      lines.push('  forwardRecords:');
      for (const r of device.dns.forwardRecords) {
        lines.push(`    - name: "${r.name}"`);
        lines.push(`      ip: "${r.ip}"`);
      }
    }
  }
  if (device.http?.enabled) {
    lines.push('http:');
    lines.push('  enabled: true');
    if (device.http.serverName) lines.push(`  serverName: "${device.http.serverName}"`);
  }
  if (device.ftp?.enabled) {
    lines.push('ftp:');
    lines.push('  enabled: true');
    if (device.ftp.welcomeBanner) lines.push(`  welcomeBanner: "${device.ftp.welcomeBanner}"`);
  }
  if (device.netbios?.enabled) {
    lines.push('netbios:');
    lines.push('  enabled: true');
    if (device.netbios.name) lines.push(`  name: "${device.netbios.name}"`);
    if (device.netbios.workgroup) lines.push(`  workgroup: "${device.netbios.workgroup}"`);
  }
  if (device.traffic?.enabled) {
    lines.push('traffic:');
    lines.push('  enabled: true');
  }
}

/**
 * Builds raw YAML for the backend create/update API
 */
export const buildDeviceRawYaml = (device: Device): string => {
  const lines: string[] = [];
  if (device.type) lines.push(`type: ${device.type}`);
  if (device.mac) lines.push(`mac: "${device.mac}"`);
  if (device.ip) lines.push(`ip: "${device.ip}"`);
  if (device.ips && device.ips.length > 0) {
    lines.push('ips:');
    for (const ip of device.ips) lines.push(`  - "${ip}"`);
  }
  if (device.vlan) lines.push(`vlan: ${device.vlan}`);
  if (device.snmpAgent) appendRawSnmpLines(lines, device.snmpAgent);
  appendNetworkProtocolLines(lines, device);
  appendServiceLines(lines, device);
  return lines.join('\n');
};

/**
 * Helper functions for building YAML preview with proper indentation
 */
const appendBaseDeviceLines = (lines: string[], device: Device): void => {
  lines.push(`  - hostname: "${device.hostname}"`);
  lines.push(`    mac: "${device.mac}"`);
  if (device.type) {
    lines.push(`    type: ${device.type}`);
  }
  if (device.ip) {
    lines.push(`    ip: "${device.ip}"`);
  }
  if (device.ips && device.ips.length > 0) {
    lines.push('    ips:');
    for (const ip of device.ips) {
      lines.push(`      - "${ip}"`);
    }
  }
};

const appendSnmpLines = (lines: string[], device: Device): void => {
  if (!device.snmpAgent) {
    return;
  }
  lines.push('    snmpAgent:');
  if (device.snmpAgent.community) {
    lines.push(`      community: "${device.snmpAgent.community}"`);
  }
  if (device.snmpAgent.sysName) {
    lines.push(`      sysName: "${device.snmpAgent.sysName}"`);
  }
  if (device.snmpAgent.walkFile) {
    lines.push(`      walkFile: "${device.snmpAgent.walkFile}"`);
  }
};

const appendLldpLines = (lines: string[], device: Device): void => {
  if (!device.lldp?.enabled) {
    return;
  }
  lines.push('    lldp:');
  lines.push('      enabled: true');
  if (device.lldp.systemDescription) {
    lines.push(`      systemDescription: "${device.lldp.systemDescription}"`);
  }
};

const appendCdpLines = (lines: string[], device: Device): void => {
  if (!device.cdp?.enabled) {
    return;
  }
  lines.push('    cdp:');
  lines.push('      enabled: true');
  if (device.cdp.platform) {
    lines.push(`      platform: "${device.cdp.platform}"`);
  }
};

const appendStpLines = (lines: string[], device: Device): void => {
  if (!device.stp?.enabled) {
    return;
  }
  lines.push('    stp:');
  lines.push('      enabled: true');
  if (device.stp.bridgePriority !== undefined) {
    lines.push(`      bridgePriority: ${device.stp.bridgePriority}`);
  }
};

const appendDhcpLines = (lines: string[], device: Device): void => {
  if (!device.dhcp) {
    return;
  }
  lines.push('    dhcp:');
  if (device.dhcp.subnetMask) {
    lines.push(`      subnetMask: "${device.dhcp.subnetMask}"`);
  }
  if (device.dhcp.router) {
    lines.push(`      router: "${device.dhcp.router}"`);
  }
  if (device.dhcp.dns && device.dhcp.dns.length > 0) {
    lines.push('      dns:');
    for (const d of device.dhcp.dns) {
      lines.push(`        - "${d}"`);
    }
  }
};

const appendDnsLines = (lines: string[], device: Device): void => {
  if (!device.dns) {
    return;
  }
  lines.push('    dns:');
  if (device.dns.forwardRecords && device.dns.forwardRecords.length > 0) {
    lines.push('      forwardRecords:');
    for (const record of device.dns.forwardRecords) {
      lines.push(`        - name: "${record.name}"`);
      lines.push(`          ip: "${record.ip}"`);
    }
  }
};

const appendHttpLines = (lines: string[], device: Device): void => {
  if (!device.http?.enabled) {
    return;
  }
  lines.push('    http:');
  lines.push('      enabled: true');
  if (device.http.serverName) {
    lines.push(`      serverName: "${device.http.serverName}"`);
  }
};

const appendFtpLines = (lines: string[], device: Device): void => {
  if (!device.ftp?.enabled) {
    return;
  }
  lines.push('    ftp:');
  lines.push('      enabled: true');
  if (device.ftp.welcomeBanner) {
    lines.push(`      welcomeBanner: "${device.ftp.welcomeBanner}"`);
  }
};

const appendNetbiosLines = (lines: string[], device: Device): void => {
  if (!device.netbios?.enabled) {
    return;
  }
  lines.push('    netbios:');
  lines.push('      enabled: true');
  if (device.netbios.name) {
    lines.push(`      name: "${device.netbios.name}"`);
  }
  if (device.netbios.workgroup) {
    lines.push(`      workgroup: "${device.netbios.workgroup}"`);
  }
};

const appendTrafficLines = (lines: string[], device: Device): void => {
  if (!device.traffic?.enabled) {
    return;
  }
  lines.push('    traffic:');
  lines.push('      enabled: true');
  if (device.traffic.arpAnnouncements?.enabled) {
    lines.push('      arpAnnouncements:');
    lines.push('        enabled: true');
  }
};

/**
 * Builds a YAML preview string for displaying device configuration
 */
export const buildYamlPreview = (device: Device): string => {
  try {
    const lines: string[] = ['devices:'];
    appendBaseDeviceLines(lines, device);
    appendSnmpLines(lines, device);
    appendLldpLines(lines, device);
    appendCdpLines(lines, device);
    appendStpLines(lines, device);
    appendDhcpLines(lines, device);
    appendDnsLines(lines, device);
    appendHttpLines(lines, device);
    appendFtpLines(lines, device);
    appendNetbiosLines(lines, device);
    appendTrafficLines(lines, device);
    return lines.join('\n');
  } catch {
    return '# Error generating YAML preview';
  }
};

/**
 * Message type for status feedback
 */
export interface StatusMessage {
  type: 'success' | 'error';
  text: string;
}
