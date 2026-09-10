import { parse } from 'yaml';
import {
  isDeviceBehaviorFaultType,
  isInterfaceBehaviorFaultType,
} from '../../api/behavior-fault-types';
import { isBehaviorActionType } from '../../api/behavior-timeline-types';
import type {
  DraftBehaviorAction,
  DraftBehaviorFault,
  DraftBehaviorPhase,
  DraftBehaviorTimeline,
  DraftBehaviorTraffic,
} from '../../api/library-client';

type UnknownRecord = Record<string, unknown>;

function record(value: unknown): UnknownRecord | null {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
    ? (value as UnknownRecord)
    : null;
}

function text(value: unknown) {
  return typeof value === 'string' ? value : '';
}

function integer(value: unknown) {
  return typeof value === 'number' && Number.isInteger(value) ? value : 0;
}

function traffic(value: unknown): DraftBehaviorTraffic | null {
  const item = record(value);
  if (!item) return null;
  return {
    device: text(item.device),
    interface: text(item.interface),
    utilization: integer(item.utilization),
  };
}

function fault(value: unknown): DraftBehaviorFault | null {
  const item = record(value);
  if (!item) return null;
  const type = text(item.type);
  const fields = { device: text(item.device), value: integer(item.value) };
  if (isDeviceBehaviorFaultType(type)) return { ...fields, type };
  if (isInterfaceBehaviorFaultType(type))
    return { ...fields, type, interface: text(item.interface) };
  return null;
}

function phase(value: unknown): DraftBehaviorPhase | null {
  const item = record(value);
  if (!item) return null;
  return {
    name: text(item.name),
    startOffsetMs: integer(item.start_offset_ms),
    durationMs: integer(item.duration_ms),
    reset: item.reset === true,
    actions: Array.isArray(item.actions)
      ? item.actions.map(operation).filter((entry): entry is DraftBehaviorAction => entry !== null)
      : [],
    traffic: Array.isArray(item.traffic)
      ? item.traffic.map(traffic).filter((entry): entry is DraftBehaviorTraffic => entry !== null)
      : [],
    faults: Array.isArray(item.faults)
      ? item.faults.map(fault).filter((entry): entry is DraftBehaviorFault => entry !== null)
      : [],
  };
}

function operation(value: unknown): DraftBehaviorAction | null {
  const item = record(value);
  if (!item) return null;
  const type = text(item.type);
  return isBehaviorActionType(type) ? { device: text(item.device), type } : null;
}

function timeline(value: unknown): DraftBehaviorTimeline | null {
  const item = record(value);
  if (!item || !Array.isArray(item.phases)) return null;
  return {
    name: text(item.name),
    startOffsetMs: integer(item.start_offset_ms),
    repeatCount: integer(item.repeat_count),
    phases: item.phases.map(phase).filter((entry): entry is DraftBehaviorPhase => entry !== null),
  };
}

export function parseDraftBehaviorTimelines(content: string): DraftBehaviorTimeline[] {
  try {
    const root = record(parse(content) as unknown);
    if (!root || !Array.isArray(root.behavior_timelines)) return [];
    return root.behavior_timelines
      .map(timeline)
      .filter((entry): entry is DraftBehaviorTimeline => entry !== null);
  } catch {
    return [];
  }
}
