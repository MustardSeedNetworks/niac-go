import { describe, expect, it } from 'vitest';
import { parseDraftBehaviorTimelines } from './behavior-timeline';

describe('parseDraftBehaviorTimelines', () => {
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
