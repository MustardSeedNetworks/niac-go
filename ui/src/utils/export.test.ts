/**
 * Tests for the device export helpers.
 *
 * These assert the produced payloads, not just that a download fired: the
 * export path is the one place a device's protocol blobs are reshaped for the
 * daemon, so a wrong key or a dropped optional field is a silent data defect.
 */

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { parse as parseYaml } from 'yaml';
import type { Device } from '../api/device-config-types';
import { exportDevicesAsCSV, exportDevicesAsJSON, exportDevicesAsYAML } from './export';

/** Captures what downloadFile() handed to the browser. */
interface Download {
  content: Promise<string>;
  filename: string;
  mimeType: string;
}

let downloads: Download[];
let createdUrls: string[];
let revokedUrls: string[];
let clicked: number;

beforeEach(() => {
  downloads = [];
  createdUrls = [];
  revokedUrls = [];
  clicked = 0;

  vi.stubGlobal('URL', {
    ...URL,
    createObjectURL: (blob: Blob) => {
      const url = `blob:mock/${createdUrls.length}`;
      createdUrls.push(url);
      // Recorded synchronously so the click() below can attach the filename to
      // this same entry; the content is read lazily by lastDownload().
      downloads.push({ content: blob.text(), filename: '', mimeType: blob.type });
      return url;
    },
    revokeObjectURL: (url: string) => {
      revokedUrls.push(url);
    },
  });

  // Anchor.click() would navigate in jsdom; record it instead.
  vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (
    this: HTMLAnchorElement,
  ) {
    clicked += 1;
    const pending = downloads[downloads.length - 1];
    if (pending) pending.filename = this.download;
  });
});

afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

/** The most recent download, with its blob contents resolved. */
async function lastDownload(): Promise<{ content: string; filename: string; mimeType: string }> {
  const d = downloads[downloads.length - 1];
  if (!d) throw new Error('no download was triggered');
  return { ...d, content: await d.content };
}

/** Indexing helper: the suite's tsconfig treats every index access as possibly undefined. */
function at<T>(items: T[], index: number): T {
  const item = items[index];
  if (item === undefined) throw new Error(`no element at index ${index}`);
  return item;
}

const minimal: Device = { hostname: 'sw1', mac: '00:11:22:33:44:55' };

const full: Device = {
  hostname: 'core-1',
  mac: 'aa:bb:cc:dd:ee:ff',
  ip: '10.0.0.1',
  ips: ['10.0.0.1', '10.0.0.2'],
  type: 'switch' as Device['type'],
  snmpAgent: { enabled: true } as Device['snmpAgent'],
  lldp: { enabled: true } as Device['lldp'],
  cdp: { enabled: true } as Device['cdp'],
  stp: { enabled: true } as Device['stp'],
  dhcp: { enabled: true } as Device['dhcp'],
  dns: { enabled: true } as Device['dns'],
  http: { enabled: true } as Device['http'],
  ftp: { enabled: true } as Device['ftp'],
  netbios: { enabled: true } as Device['netbios'],
};

describe('exportDevicesAsYAML', () => {
  it('emits only the fields the device actually has', async () => {
    exportDevicesAsYAML([minimal]);

    const doc = parseYaml((await lastDownload()).content) as {
      devices: Record<string, unknown>[];
    };
    expect(doc.devices).toHaveLength(1);
    // name and mac are the only populated fields; nothing optional leaks through.
    expect(Object.keys(at(doc.devices, 0)).sort()).toEqual(['mac', 'name']);
  });

  it('maps every protocol blob onto the key the daemon expects', async () => {
    exportDevicesAsYAML([full]);

    const doc = parseYaml((await lastDownload()).content) as {
      devices: Record<string, unknown>[];
    };
    const out = at(doc.devices, 0);
    // Every key here is checked against internal/converter/types.go. This
    // test previously pinned `netbios_status`, which the daemon does not
    // declare -- and because it decodes with KnownFields(true), that one key
    // made the whole exported file unloadable.
    expect(out.name).toBe('core-1');
    expect(out.snmp_agent).toBeDefined();
    expect(out.netbios).toBeDefined();
    expect(out.ips).toEqual(['10.0.0.1', '10.0.0.2']);
    expect(out).not.toHaveProperty('hostname');
    expect(out).not.toHaveProperty('netbios_status');
  });

  it('omits an empty ips array rather than emitting an empty list', async () => {
    exportDevicesAsYAML([{ ...minimal, ips: [] }]);

    const doc = parseYaml((await lastDownload()).content) as {
      devices: Record<string, unknown>[];
    };
    expect(at(doc.devices, 0)).not.toHaveProperty('ips');
  });

  it('defaults the filename and honours an override', async () => {
    exportDevicesAsYAML([minimal]);
    expect((await lastDownload()).filename).toBe('devices.yaml');

    exportDevicesAsYAML([minimal], 'custom.yaml');
    expect((await lastDownload()).filename).toBe('custom.yaml');
  });
});

describe('exportDevicesAsJSON', () => {
  it('round-trips the devices unchanged', async () => {
    exportDevicesAsJSON([full]);

    const download = await lastDownload();
    expect(JSON.parse(download.content)).toEqual([full]);
    expect(download.mimeType).toBe('application/json');
    expect(download.filename).toBe('devices.json');
  });
});

describe('exportDevicesAsCSV', () => {
  it('lists only the protocols that are enabled', async () => {
    exportDevicesAsCSV([full]);

    const lines = (await lastDownload()).content.split('\n');
    const header = at(lines, 0);
    const row = at(lines, 1);
    expect(header).toBe('hostname,type,mac,ip,ips,protocols');
    // Every protocol on `full` is enabled, in declaration order.
    expect(row).toContain('SNMP; LLDP; CDP; STP; DHCP; DNS; HTTP; FTP; NetBIOS');
  });

  it('treats a present-but-disabled protocol as absent', async () => {
    exportDevicesAsCSV([
      {
        ...minimal,
        lldp: { enabled: false } as Device['lldp'],
        cdp: { enabled: true } as Device['cdp'],
      },
    ]);

    const row = at((await lastDownload()).content.split('\n'), 1);
    expect(row).toContain('CDP');
    expect(row).not.toContain('LLDP');
  });

  it('counts snmpAgent, dhcp and dns by presence, not by an enabled flag', async () => {
    // These three have no `enabled` gate in getDeviceProtocolList — presence
    // alone is the signal, unlike the others.
    exportDevicesAsCSV([
      {
        ...minimal,
        snmpAgent: {} as Device['snmpAgent'],
        dhcp: {} as Device['dhcp'],
        dns: {} as Device['dns'],
      },
    ]);

    const row = at((await lastDownload()).content.split('\n'), 1);
    expect(row).toContain('SNMP; DHCP; DNS');
  });

  it('quotes fields containing a comma, a quote or a newline', async () => {
    exportDevicesAsCSV([{ hostname: 'a,b', mac: 'has "quotes"', ip: 'line\nbreak' }]);

    const row = (await lastDownload()).content.split('\n').slice(1).join('\n');
    expect(row).toContain('"a,b"');
    // Embedded quotes are doubled, per RFC 4180.
    expect(row).toContain('"has ""quotes"""');
    expect(row).toContain('"line\nbreak"');
  });

  it('leaves a field with no special characters unquoted', async () => {
    exportDevicesAsCSV([minimal]);

    const row = at((await lastDownload()).content.split('\n'), 1);
    expect(row.startsWith('sw1,')).toBe(true);
  });

  it('renders absent optional fields as empty columns', async () => {
    exportDevicesAsCSV([minimal]);

    const row = at((await lastDownload()).content.split('\n'), 1);
    // type, ip, ips and protocols are all absent on a minimal device.
    expect(row).toBe('sw1,,00:11:22:33:44:55,,,');
  });

  it('emits a header even when there are no devices', async () => {
    exportDevicesAsCSV([]);

    expect((await lastDownload()).content).toBe('hostname,type,mac,ip,ips,protocols');
  });
});

describe('download plumbing', () => {
  it('revokes the object URL it created', async () => {
    exportDevicesAsJSON([minimal]);
    await lastDownload();

    expect(clicked).toBe(1);
    expect(revokedUrls).toEqual(createdUrls);
  });

  it('leaves no anchor behind in the document', async () => {
    exportDevicesAsJSON([minimal]);
    await lastDownload();

    expect(document.querySelectorAll('a')).toHaveLength(0);
  });
});
