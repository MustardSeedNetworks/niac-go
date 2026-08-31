import { stringify as stringifyYaml } from 'yaml';
import type { Device } from '../api/types';
import { toDaemonDevice } from './device-yaml';

/**
 * Export devices as a NIAC-loadable YAML config and trigger file download.
 * The structure matches what the daemon expects under `devices:` so the
 * downloaded file can be fed straight back into Simulation → Pick.
 */
export function exportDevicesAsYAML(devices: Device[], filename = 'devices.yaml'): void {
  // Same mapper as the device-editor preview. This function used to build the
  // document itself and emitted `netbios_status` plus camelCase sub-fields, so
  // the file it produced could not be loaded back at all -- the daemon decodes
  // with KnownFields(true), which makes one unknown key fatal.
  const doc = { devices: devices.map(toDaemonDevice) };
  const yaml = stringifyYaml(doc);
  downloadFile(yaml, filename, 'application/x-yaml');
}

/**
 * Export devices as JSON and trigger file download.
 *
 * For data analysis / reporting only — JSON isn't a loadable NIAC
 * config format. Use exportDevicesAsYAML for something runnable.
 */
export function exportDevicesAsJSON(devices: Device[], filename = 'devices.json'): void {
  const json = JSON.stringify(devices, null, 2);
  downloadFile(json, filename, 'application/json');
}

/**
 * Export devices as CSV and trigger file download.
 *
 * For data analysis / reporting only — CSV isn't a loadable NIAC config
 * format. Use exportDevicesAsYAML for something runnable.
 */
export function exportDevicesAsCSV(devices: Device[], filename = 'devices.csv'): void {
  const headers = ['hostname', 'type', 'mac', 'ip', 'ips', 'protocols'];
  const rows = devices.map((device) => {
    const protocols = getDeviceProtocolList(device).join('; ');
    const ips = device.ips?.join('; ') ?? '';
    return [
      escapeCsvField(device.hostname),
      escapeCsvField(device.type ?? ''),
      escapeCsvField(device.mac),
      escapeCsvField(device.ip ?? ''),
      escapeCsvField(ips),
      escapeCsvField(protocols),
    ].join(',');
  });

  const csv = [headers.join(','), ...rows].join('\n');
  downloadFile(csv, filename, 'text/csv');
}

function getDeviceProtocolList(device: Device): string[] {
  const protocols: string[] = [];
  if (device.snmpAgent) protocols.push('SNMP');
  if (device.lldp?.enabled) protocols.push('LLDP');
  if (device.cdp?.enabled) protocols.push('CDP');
  if (device.stp?.enabled) protocols.push('STP');
  if (device.dhcp) protocols.push('DHCP');
  if (device.dns) protocols.push('DNS');
  if (device.http?.enabled) protocols.push('HTTP');
  if (device.ftp?.enabled) protocols.push('FTP');
  if (device.netbios?.enabled) protocols.push('NetBIOS');
  return protocols;
}

function escapeCsvField(field: string): string {
  if (field.includes(',') || field.includes('"') || field.includes('\n')) {
    return `"${field.replace(/"/g, '""')}"`;
  }
  return field;
}

function downloadFile(content: string, filename: string, mimeType: string): void {
  const blob = new Blob([content], { type: mimeType });
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
}
