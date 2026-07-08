import type { LibraryNetwork, Template } from '../../api/types';

/**
 * NewSimulationWizard step identifiers, in stepper order. Kept as a
 * const tuple (not an enum) so the stepper can derive index/label
 * arrays from the same source of truth.
 */
export const WIZARD_STEPS = ['template', 'devices', 'protocols', 'review', 'finish'] as const;
export type WizardStepId = (typeof WIZARD_STEPS)[number];

/**
 * Where the starting config comes from. 'empty' has no existing-UI
 * equivalent — it's a one-line addition (a blank devices: [] skeleton)
 * so the wizard doesn't force a template pick.
 */
export type WizardSource = 'template' | 'userConfig' | 'upload' | 'empty';

/**
 * WizardState is held locally (useState) in the container — it never
 * touches the global ui-store. Once `configPath` is set the picked
 * config has been materialised as a real, safe (non-built-in) file and
 * the simulation has been started against it, which is what unlocks
 * the existing device-editing surface (device CRUD only operates on
 * the daemon's currently-active config).
 */
export interface WizardState {
  step: number;
  source: WizardSource | null;
  template: Template | null;
  userConfig: LibraryNetwork | null;
  uploadFile: File | null;
  selectedInterface: string;
  configPath: string | null;
  starting: boolean;
  saving: boolean;
}

export const initialWizardState: WizardState = {
  step: 0,
  source: null,
  template: null,
  userConfig: null,
  uploadFile: null,
  selectedInterface: '',
  configPath: null,
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
  return false;
}
