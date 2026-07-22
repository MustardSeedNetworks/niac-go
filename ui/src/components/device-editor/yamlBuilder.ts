import type { Device } from '../../api/types';

const appendBaseDeviceLines = (lines: string[], device: Device) => {
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

const appendSnmpLines = (lines: string[], device: Device) => {
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
  if (device.snmpAgent.walkFiles && device.snmpAgent.walkFiles.length > 0) {
    lines.push('      walkFiles:');
    for (const walkFile of device.snmpAgent.walkFiles) {
      lines.push(`        - "${walkFile}"`);
    }
  }
  if (device.snmpAgent.addMibs && device.snmpAgent.addMibs.length > 0) {
    lines.push('      addMibs:');
    for (const mib of device.snmpAgent.addMibs) {
      lines.push(`        - oid: "${mib.oid}"`);
      lines.push(`          type: "${mib.type}"`);
      lines.push(`          value: "${mib.value}"`);
    }
  }
};

const appendInterfaceLines = (lines: string[], device: Device) => {
  if (!device.interfaceDetails || device.interfaceDetails.length === 0) {
    return;
  }

  lines.push('    interfaces:');
  for (const iface of device.interfaceDetails) {
    lines.push(`      - name: "${iface.name}"`);
    if (iface.speed !== undefined && iface.speed > 0) {
      lines.push(`        speed: ${iface.speed}`);
    }
    if (iface.duplex) {
      lines.push(`        duplex: "${iface.duplex}"`);
    }
    if (iface.adminStatus) {
      lines.push(`        adminStatus: "${iface.adminStatus}"`);
    }
    if (iface.operStatus) {
      lines.push(`        operStatus: "${iface.operStatus}"`);
    }
    if (iface.description) {
      lines.push(`        description: "${iface.description}"`);
    }
  }
};

const appendLldpLines = (lines: string[], device: Device) => {
  if (!device.lldp?.enabled) {
    return;
  }
  lines.push('    lldp:');
  lines.push('      enabled: true');
  if (device.lldp.systemDescription) {
    lines.push(`      systemDescription: "${device.lldp.systemDescription}"`);
  }
};

const appendCdpLines = (lines: string[], device: Device) => {
  if (!device.cdp?.enabled) {
    return;
  }
  lines.push('    cdp:');
  lines.push('      enabled: true');
  if (device.cdp.platform) {
    lines.push(`      platform: "${device.cdp.platform}"`);
  }
};

const appendStpLines = (lines: string[], device: Device) => {
  if (!device.stp?.enabled) {
    return;
  }
  lines.push('    stp:');
  lines.push('      enabled: true');
  if (device.stp.bridgePriority !== undefined) {
    lines.push(`      bridgePriority: ${device.stp.bridgePriority}`);
  }
};

const appendDhcpLines = (lines: string[], device: Device) => {
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
  if (device.dhcp.domainNameServer) {
    lines.push(`      domainNameServer: "${device.dhcp.domainNameServer}"`);
  }
};

const appendDnsLines = (lines: string[], device: Device) => {
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

const appendHttpLines = (lines: string[], device: Device) => {
  if (!device.http?.enabled) {
    return;
  }
  lines.push('    http:');
  lines.push('      enabled: true');
  if (device.http.serverName) {
    lines.push(`      serverName: "${device.http.serverName}"`);
  }
};

const appendFtpLines = (lines: string[], device: Device) => {
  if (!device.ftp?.enabled) {
    return;
  }
  lines.push('    ftp:');
  lines.push('      enabled: true');
  if (device.ftp.welcomeBanner) {
    lines.push(`      welcomeBanner: "${device.ftp.welcomeBanner}"`);
  }
};

const appendNetbiosLines = (lines: string[], device: Device) => {
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

const appendTrafficLines = (lines: string[], device: Device) => {
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
 * Build a YAML preview string from a Device object
 */
export const buildYamlPreview = (device: Device): string => {
  try {
    const lines: string[] = ['devices:'];
    appendBaseDeviceLines(lines, device);
    appendInterfaceLines(lines, device);
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
