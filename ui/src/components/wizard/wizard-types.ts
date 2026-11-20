import {
  defaultScenarioRequest,
  isScenarioRequestValid,
  type ScenarioGenerateRequest,
} from '../../api/scenario-client';
import type {
  HistoryRecord,
  LibraryNetwork,
  NetworkInterface,
  SimulationStatus,
} from '../../api/types';

/**
 * NewSimulationWizard step identifiers, in stepper order. Kept as a
 * const tuple (not an enum) so the stepper can derive index/label
 * arrays from the same source of truth.
 *
 * `networks` sits after `devices` because addressing needs devices to exist
 * before it can put them on a network.
 */
export const WIZARD_STEPS = [
  'scenario',
  'devices',
  'networks',
  'protocols',
  'review',
  'preflight',
  'finish',
] as const;
export type WizardStepId = (typeof WIZARD_STEPS)[number];

/**
 * Where the starting config comes from. 'empty' has no existing-UI
 * equivalent — it's a one-line addition (a blank devices: [] skeleton)
 * so the wizard doesn't force a scenario pick.
 */
export type WizardSource = 'userConfig' | 'upload' | 'empty' | 'generated';

/**
 * WizardState is held locally in the container. Draft content and its
 * revision live beside this navigation/source state so authoring never
 * changes the daemon's active configuration.
 */
export interface WizardState {
  step: number;
  source: WizardSource | null;
  userConfig: LibraryNetwork | null;
  uploadFile: File | null;
  fleetRequest: ScenarioGenerateRequest;
  /**
   * The scenario pack `fleetRequest` came from, or null when the operator is
   * tuning the generator by hand. A pack and the hand-tuned fleet are two ways
   * of choosing the same starting point, so only one of them is ever the
   * selection; editing any generator field drops the pack (#2185).
   */
  fleetPackId: string | null;
  /**
   * The pack the generator was tuned away from, so the UI can say the fleet is
   * now custom rather than silently dropping the choice. Cleared by any other
   * choice of starting point.
   */
  customizedFromPackId: string | null;
  selectedInterface: string;
  starting: boolean;
  saving: boolean;
}

export const initialWizardState: WizardState = {
  step: 0,
  source: null,
  userConfig: null,
  uploadFile: null,
  fleetRequest: defaultScenarioRequest(),
  fleetPackId: null,
  customizedFromPackId: null,
  selectedInterface: '',
  starting: false,
  saving: false,
};

/** Step 1 is complete once a source is picked and an interface chosen. */
export function isStartingPointStepComplete(state: WizardState): boolean {
  if (!state.selectedInterface) return false;
  if (state.source === 'empty') return true;
  if (state.source === 'userConfig') return state.userConfig !== null;
  if (state.source === 'upload') return state.uploadFile !== null;
  if (state.source === 'generated') return isScenarioRequestValid(state.fleetRequest);
  return false;
}

/**
 * Start as-is runs an unmodified pack or library network, so it needs one of
 * those chosen. A hand-tuned fleet, an upload and a blank start are authored
 * content and go through the editing steps.
 */
export function canStartAsIs(state: WizardState): boolean {
  if (!isStartingPointStepComplete(state)) return false;
  if (state.source === 'userConfig') return true;
  return state.source === 'generated' && state.fleetPackId !== null;
}

const loopbackAddress = /^(127\.|::1$)/;

function isLoopbackInterface(iface: NetworkInterface): boolean {
  if (/^lo\d*$/.test(iface.name)) return true;
  return iface.addresses.length > 0 && iface.addresses.every((a) => loopbackAddress.test(a));
}

/**
 * Loopback carries no traffic an operator can observe from another host, so it
 * is hidden whenever a real interface exists. A host that offers nothing else
 * keeps it, so the wizard cannot dead-end.
 */
export function selectableInterfaces(interfaces: NetworkInterface[]): NetworkInterface[] {
  const real = interfaces.filter((iface) => !isLoopbackInterface(iface));
  return real.length > 0 ? real : interfaces;
}

/** The interface of the most recent run that is still on offer, or ''. */
export function lastUsedInterface(history: HistoryRecord[], offered: NetworkInterface[]): string {
  const names = new Set(offered.map((iface) => iface.name));
  let best = '';
  let bestAt = Number.NEGATIVE_INFINITY;
  for (const record of history) {
    const at = Date.parse(record.startedAt);
    if (!names.has(record.interface) || Number.isNaN(at) || at <= bestAt) continue;
    best = record.interface;
    bestAt = at;
  }
  return best;
}

/** Names of the running sessions already bound to `iface`. */
export function sessionsOnInterface(status: SimulationStatus | null, iface: string): string[] {
  if (!status || !iface) return [];
  const sessions = status.sessions?.length ? status.sessions : [status];
  return sessions
    .filter((session) => session.running && session.interface === iface)
    .map((session) => session.sessionId ?? session.configName ?? iface);
}
