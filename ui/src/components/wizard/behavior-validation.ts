import { validBehaviorFault } from '../../api/behavior-fault-types';
import {
  type DraftBehaviorTimeline,
  validBehaviorActions,
} from '../../api/behavior-timeline-types';

export const validBehaviorTimelines = (timelines: DraftBehaviorTimeline[]) =>
  timelines.every(
    (timeline) =>
      timeline.name.trim() &&
      timeline.repeatCount >= 1 &&
      timeline.repeatCount <= 1000 &&
      timeline.startOffsetMs >= 0 &&
      timeline.phases.length > 0 &&
      timeline.phases.every(
        (phase) =>
          phase.name.trim() &&
          phase.startOffsetMs >= 0 &&
          phase.durationMs > 0 &&
          phase.traffic.length + phase.faults.length + phase.actions.length > 0 &&
          phase.traffic.every(
            (action) =>
              action.device &&
              action.interface &&
              action.utilization >= 1 &&
              action.utilization <= 100,
          ) &&
          phase.faults.every(validBehaviorFault) &&
          validBehaviorActions(phase.actions),
      ),
  );
