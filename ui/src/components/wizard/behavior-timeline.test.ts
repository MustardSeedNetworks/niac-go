import { describe, expect, it } from 'vitest';
import { stringify } from 'yaml';
import { parseDraftBehaviorTimelines } from './behavior-timeline';

describe('parseDraftBehaviorTimelines', () => {
  it('preserves ordered one-shot operations without adding fault fields', () => {
    const actions = [
      { device: 'edge-1', type: 'stp_topology_change' },
      { device: 'edge-2', type: 'reboot' },
    ];
    const parsed = parseDraftBehaviorTimelines(
      stringify({
        behavior_timelines: [
          {
            name: 'Maintenance',
            repeat_count: 2,
            phases: [{ name: 'Changes', duration_ms: 1000, reset: true, actions }],
          },
        ],
      }),
    );
    expect(parsed[0]?.phases[0]?.actions).toEqual(actions);
  });
  it('round-trips all resource faults without dropping them or adding interfaces', () => {
    const faults = [
      { device: 'hospital-server-1', type: 'cpu_percent', value: 1 },
      { device: 'hospital-server-1', type: 'memory_percent', value: 50 },
      { device: 'hospital-server-1', type: 'disk_percent', value: 100 },
      { device: 'access-1', interface: 'eth0', type: 'link_down', value: 1 },
    ];
    const authored = (actions: typeof faults) =>
      stringify({
        behavior_timelines: [
          { name: 'Resource pressure', phases: [{ name: 'Busy', faults: actions }] },
        ],
      });
    const parsed = parseDraftBehaviorTimelines(authored(faults));
    expect(parsed[0]?.phases[0]?.faults).toEqual(faults);
    const reloaded = parseDraftBehaviorTimelines(authored(parsed[0]?.phases[0]?.faults ?? []));
    expect(reloaded).toEqual(parsed);
  });
  it('preserves an interface-scoped link-down fault', () => {
    const parsed = parseDraftBehaviorTimelines(`
behavior_timelines:
  - name: Link outage
    repeat_count: 1
    phases:
      - name: Down
        duration_ms: 1000
        faults: [{device: edge-1, interface: eth0, type: link_down, value: 1}]
`);
    expect(parsed[0]?.phases[0]?.faults).toEqual([
      { device: 'edge-1', interface: 'eth0', type: 'link_down', value: 1 },
    ]);
  });
  it.each(['dhcp_no_offer', 'dns_nxdomain', 'dns_timeout', 'latency'])(
    'preserves a device-scoped %s without inventing an interface',
    (type) => {
      const parsed = parseDraftBehaviorTimelines(`
behavior_timelines:
  - name: Service outage
    repeat_count: 1
    phases:
      - name: Fault
        duration_ms: 1000
        faults:
          - device: resolver-1
            type: ${type}
            value: ${type === 'latency' ? 60000 : 100}
`);
      expect(parsed[0]?.phases[0]?.faults).toEqual([
        { device: 'resolver-1', type, value: type === 'latency' ? 60000 : 100 },
      ]);
    },
  );
  it('reads saved behavior timelines from draft YAML', () => {
    expect(
      parseDraftBehaviorTimelines(`
behavior_timelines:
  - name: Morning rush
    start_offset_ms: 1000
    repeat_count: 2
    phases:
      - name: Congestion
        start_offset_ms: 0
        duration_ms: 30000
        reset: true
        traffic:
          - device: access-1
            interface: Gi1/0/1
            utilization: 82
        faults:
          - device: access-1
            interface: Gi1/0/2
            type: packet_discards
            value: 4
`),
    ).toEqual([
      {
        name: 'Morning rush',
        startOffsetMs: 1000,
        repeatCount: 2,
        phases: [
          {
            name: 'Congestion',
            startOffsetMs: 0,
            durationMs: 30000,
            reset: true,
            actions: [],
            traffic: [{ device: 'access-1', interface: 'Gi1/0/1', utilization: 82 }],
            faults: [
              {
                device: 'access-1',
                interface: 'Gi1/0/2',
                type: 'packet_discards',
                value: 4,
              },
            ],
          },
        ],
      },
    ]);
  });

  it('returns an empty list for invalid YAML or unsupported fault types', () => {
    expect(parseDraftBehaviorTimelines('[')).toEqual([]);
    expect(
      parseDraftBehaviorTimelines(`
behavior_timelines:
  - name: Invalid fault
    phases:
      - faults:
          - type: unavailable
`)[0]?.phases[0]?.faults,
    ).toEqual([]);
  });
});
