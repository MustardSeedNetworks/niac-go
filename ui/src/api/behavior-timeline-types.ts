import type { DraftBehaviorFault } from './behavior-fault-types';

export interface DraftBehaviorTraffic {
  device: string;
  interface: string;
  utilization: number;
}

export interface DraftBehaviorPhase {
  name: string;
  startOffsetMs: number;
  durationMs: number;
  reset: boolean;
  traffic: DraftBehaviorTraffic[];
  faults: DraftBehaviorFault[];
  actions: DraftBehaviorAction[];
}

export interface DraftBehaviorTimeline {
  name: string;
  startOffsetMs: number;
  repeatCount: number;
  phases: DraftBehaviorPhase[];
}

export interface DraftBehaviorAction {
  device: string;
  type: 'reboot' | 'stp_topology_change';
}

export const isBehaviorActionType = (type: string): type is DraftBehaviorAction['type'] =>
  type === 'reboot' || type === 'stp_topology_change';

export const validBehaviorActions = (actions: DraftBehaviorAction[]) =>
  actions.every(
    (action, index) =>
      Boolean(action.device) &&
      isBehaviorActionType(action.type) &&
      !actions
        .slice(0, index)
        .some((other) => other.device === action.device && other.type === action.type),
  );
