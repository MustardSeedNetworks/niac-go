import {
  defaultScenarioRequest,
  isScenarioRequestValid,
  type ScenarioGenerateRequest,
} from '../../api/scenario-client';
import type { BuiltinScenario, LibraryNetwork } from '../../api/types';

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
export type WizardSource = 'builtin' | 'userConfig' | 'upload' | 'empty' | 'generated';

/**
 * WizardState is held locally in the container. Draft content and its
 * revision live beside this navigation/source state so authoring never
 * changes the daemon's active configuration.
 */
export interface WizardState {
  step: number;
  source: WizardSource | null;
  builtin: BuiltinScenario | null;
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
  selectedInterface: string;
  starting: boolean;
  saving: boolean;
}

export const initialWizardState: WizardState = {
  step: 0,
  source: null,
  builtin: null,
  userConfig: null,
  uploadFile: null,
  fleetRequest: defaultScenarioRequest(),
  fleetPackId: null,
  selectedInterface: '',
  starting: false,
  saving: false,
};

/** Step 1 is complete once a source is picked and an interface chosen. */
export function isStartingPointStepComplete(state: WizardState): boolean {
  if (!state.selectedInterface) return false;
  if (state.source === 'empty') return true;
  if (state.source === 'builtin') return state.builtin !== null;
  if (state.source === 'userConfig') return state.userConfig !== null;
  if (state.source === 'upload') return state.uploadFile !== null;
  if (state.source === 'generated') return isScenarioRequestValid(state.fleetRequest);
  return false;
}
