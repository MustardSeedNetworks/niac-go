import { describe, expect, it } from 'vitest';
import {
  parseFleetDefaults,
  serializeCapturePlaybacks,
  serializeDiscoveryProtocols,
} from './fleet-defaults';

const config = `discovery_protocols:
  lldp:
    enabled: true
    interval: 30
  cdp:
    enabled: false

capture_playbacks:
  - file_name: office.pcap
    loop_time: 60
    scale_time: 0.5

devices: []
`;

describe('parseFleetDefaults', () => {
  it('reads the discovery defaults and the capture playback', () => {
    const model = parseFleetDefaults(config);

    expect(model.discoveryProtocols.lldp).toEqual({ enabled: true, interval: 30 });
    expect(model.discoveryProtocols.cdp).toEqual({ enabled: false, interval: undefined });
    expect(model.discoveryProtocols.edp).toBeUndefined();
    expect(model.capturePlaybacks).toEqual([
      { fileName: 'office.pcap', loopTime: 60, scaleTime: 0.5 },
    ]);
  });

  it('returns empty defaults for a config that does not parse', () => {
    expect(parseFleetDefaults('devices: [\n  - broken')).toEqual({
      discoveryProtocols: {},
      capturePlaybacks: [],
    });
  });
});

describe('serializeDiscoveryProtocols', () => {
  it('round-trips an enabled protocol', () => {
    const yaml = serializeDiscoveryProtocols({ lldp: { enabled: true, interval: 30 } });

    expect(yaml).toBe('discovery_protocols:\n  lldp:\n    enabled: true\n    interval: 30\n');
    expect(parseFleetDefaults(yaml).discoveryProtocols.lldp).toEqual({
      enabled: true,
      interval: 30,
    });
  });

  it('omits an absent interval rather than writing zero', () => {
    expect(serializeDiscoveryProtocols({ cdp: { enabled: true } })).toBe(
      'discovery_protocols:\n  cdp:\n    enabled: true\n',
    );
  });

  it('returns empty string when nothing is enabled, which the splice reads as removal', () => {
    expect(serializeDiscoveryProtocols({})).toBe('');
    expect(serializeDiscoveryProtocols({ lldp: { enabled: false } })).toBe('');
  });
});

describe('serializeCapturePlaybacks', () => {
  it('emits at most one entry, because the loader refuses more', () => {
    const yaml = serializeCapturePlaybacks([
      { fileName: 'first.pcap' },
      { fileName: 'second.pcap' },
    ]);

    expect(yaml).toBe('capture_playbacks:\n  - file_name: first.pcap\n');
    expect(yaml).not.toContain('second.pcap');
  });

  it('treats a blank file name as no playback at all', () => {
    expect(serializeCapturePlaybacks([{ fileName: '   ' }])).toBe('');
    expect(serializeCapturePlaybacks([])).toBe('');
  });

  it('round-trips loop and scale time', () => {
    const yaml = serializeCapturePlaybacks([
      { fileName: 'office.pcap', loopTime: 60, scaleTime: 0.5 },
    ]);
    expect(parseFleetDefaults(yaml).capturePlaybacks[0]).toEqual({
      fileName: 'office.pcap',
      loopTime: 60,
      scaleTime: 0.5,
    });
  });
});
