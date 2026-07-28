import {
  enterpriseScenarioRequest,
  isScenarioRequestValid,
  type ScenarioGenerateRequest,
} from '../../api/scenario-client';
import type { LibraryNetwork, Template } from '../../api/types';

/**
 * NewSimulationWizard step identifiers, in stepper order. Kept as a
 * const tuple (not an enum) so the stepper can derive index/label
 * arrays from the same source of truth.
 */
export const WIZARD_STEPS = [
  'template',
  'devices',
  'protocols',
  'review',
  'preflight',
  'finish',
] as const;
export type WizardStepId = (typeof WIZARD_STEPS)[number];

/**
 * Where the starting config comes from. 'empty' has no existing-UI
 * equivalent — it's a one-line addition (a blank devices: [] skeleton)
 * so the wizard doesn't force a template pick.
 */
export type WizardSource = 'template' | 'userConfig' | 'upload' | 'empty' | 'generated';

/**
 * WizardState is held locally in the container. Draft content and its
 * revision live beside this navigation/source state so authoring never
 * changes the daemon's active configuration.
 */
export interface WizardState {
  step: number;
  source: WizardSource | null;
  template: Template | null;
  userConfig: LibraryNetwork | null;
  uploadFile: File | null;
  fleetRequest: ScenarioGenerateRequest;
  selectedInterface: string;
  starting: boolean;
  saving: boolean;
}

export const initialWizardState: WizardState = {
  step: 0,
  source: null,
  template: null,
  userConfig: null,
  uploadFile: null,
  fleetRequest: enterpriseScenarioRequest(),
  selectedInterface: '',
  starting: false,
  saving: false,
};

/** Step 1 is complete once a source is picked and an interface chosen. */
export function isTemplateStepComplete(state: WizardState): boolean {
  if (!state.selectedInterface) return false;
  if (state.source === 'empty') return true;
  if (state.source === 'template') return state.template !== null;
  if (state.source === 'userConfig') return state.userConfig !== null;
  if (state.source === 'upload') return state.uploadFile !== null;
  if (state.source === 'generated') return isScenarioRequestValid(state.fleetRequest);
  return false;
}
