import { type FC, useCallback, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { startSimulation } from '../api/client';
import {
  createScenarioDraft,
  createScenarioDraftFromTemplate,
  deleteScenarioDraft,
  fetchLibraryNetworkContent,
  replaceScenarioDraft,
  type ScenarioDraft,
} from '../api/library-client';
import { generateScenario } from '../api/scenario-client';
import type { LibraryNetwork, SimulationRequest, Template } from '../api/types';
import { DevicesStep } from '../components/wizard/DevicesStep';
import { FinishStep } from '../components/wizard/FinishStep';
import { NetworksStep } from '../components/wizard/NetworksStep';
import { PreflightStep } from '../components/wizard/PreflightStep';
import { ProtocolsStep } from '../components/wizard/ProtocolsStep';
import { ReviewStep } from '../components/wizard/ReviewStep';
import { TemplateStep } from '../components/wizard/TemplateStep';
import { WizardStepper } from '../components/wizard/WizardStepper';
import {
  initialWizardState,
  isTemplateStepComplete,
  WIZARD_STEPS,
  type WizardState,
} from '../components/wizard/wizard-types';
import { useErrorToast } from '../hooks/useErrorToast';
import { useSimulationStatus } from '../hooks/useSimulationStatus';
import { Button } from '../ui/Button';
import { Card, CardContent } from '../ui/Card';
import { SmallText } from '../ui/Typography';
import { reportError } from '../utils/error-reporter';
import { fileToText } from '../utils/file';

// An empty start is genuinely empty. The placeholder host this used to seed
// was indistinguishable from a device the author had added, so it either
// shipped into the scenario or had to be found and deleted first.
const EMPTY_CONFIG_YAML = 'devices: []\n';

function newDraftName(now = new Date()) {
  return `scenario-${now.toISOString().replaceAll(/[-:.TZ]/g, '')}`;
}

/**
 * Steps that edit the draft's content and so must save before navigating.
 *
 * This was `=== 'devices'` when Devices was the only editing step. Leaving it
 * that way would have let the Networks and Protocols steps change the draft
 * and silently drop it on Next or Back.
 */
const CONTENT_EDITING_STEPS = new Set(['devices', 'networks', 'protocols']);

function selectedSourceKey(state: WizardState) {
  if (state.source === 'template' && state.template) return `template:${state.template.name}`;
  if (state.source === 'userConfig' && state.userConfig) return `network:${state.userConfig.name}`;
  if (state.source === 'upload' && state.uploadFile) {
    return `upload:${state.uploadFile.name}:${state.uploadFile.size}:${state.uploadFile.lastModified}`;
  }
  if (state.source === 'generated') return `generated:${JSON.stringify(state.fleetRequest)}`;
  return state.source === 'empty' ? 'empty' : null;
}

/**
 * NewSimulationWizard sequences draft authoring and runtime activation:
 *
 *   1. Template  — ConfigPicker (pick a template / saved config / upload / empty)
 *   2. Devices   — edit and revision-save the isolated draft
 *   3. Protocols — inspect identities and configured services
 *   4. Review    — review the exact saved draft revision
 *   5. Connection — preflight and start only after authoring is complete
 *   6. Finish    — hand off to runtime monitoring
 */
export const NewSimulationWizardPage: FC = () => {
  const { t } = useTranslation('pages');
  const { data: simStatus } = useSimulationStatus();
  const showError = useErrorToast();
  const [state, setState] = useState<WizardState>(initialWizardState);
  const [draft, setDraft] = useState<ScenarioDraft | null>(null);
  const [draftContent, setDraftContent] = useState('');
  const [draftDirty, setDraftDirty] = useState(false);
  const [draftSourceKey, setDraftSourceKey] = useState<string | null>(null);
  const [startedSessionId, setStartedSessionId] = useState<string | null>(null);

  /**
   * Best-effort removal of a draft the wizard is done with.
   *
   * A failure here is not the author's problem: the draft they are moving to
   * has already been created, or they have already cancelled. Surfacing it
   * would report an error for an action they did not take, so it is logged
   * and swallowed rather than raised.
   */
  const discardDraft = useCallback(async (target: ScenarioDraft) => {
    try {
      await deleteScenarioDraft(target.name, target.revision);
    } catch (err) {
      reportError(err, `discard-scenario-draft:${target.name}`);
    }
  }, []);

  const steps = WIZARD_STEPS.map((id) => ({
    id,
    label: t(`newSimWizard.steps.${id}`),
  }));

  const prepareConfig = useCallback(async () => {
    setState((s) => ({ ...s, starting: true }));
    try {
      const sourceKey = selectedSourceKey(state);
      if (draft && sourceKey === draftSourceKey) {
        setDraftContent(draft.content);
        setDraftDirty(false);
        setState((s) => ({ ...s, step: 1, starting: false }));
        return;
      }

      const draftName = newDraftName();
      let created: ScenarioDraft;

      if (state.source === 'template' && state.template) {
        created = await createScenarioDraftFromTemplate(draftName, state.template.name);
      } else if (state.source === 'userConfig' && state.userConfig) {
        const content = (await fetchLibraryNetworkContent(state.userConfig.name)).content;
        created = await createScenarioDraft(draftName, content);
      } else if (state.source === 'upload' && state.uploadFile) {
        created = await createScenarioDraft(draftName, await fileToText(state.uploadFile));
      } else if (state.source === 'empty') {
        created = await createScenarioDraft(draftName, EMPTY_CONFIG_YAML);
      } else if (state.source === 'generated') {
        const generated = await generateScenario(state.fleetRequest);
        created = await createScenarioDraft(draftName, generated.content);
      } else {
        throw new Error(t('newSimWizard.template.errorNoSource'));
      }

      // Changing the source mid-wizard used to leave the previous draft behind
      // in the library, one per change of mind.
      if (draft) await discardDraft(draft);

      setDraft(created);
      setDraftContent(created.content);
      setDraftDirty(false);
      setDraftSourceKey(sourceKey);
      setState((s) => ({ ...s, step: 1, starting: false }));
    } catch (err) {
      showError(err);
      setState((s) => ({ ...s, starting: false }));
    }
  }, [draft, draftSourceKey, discardDraft, state, showError, t]);

  const saveDraft = useCallback(async () => {
    if (!draft || !draftDirty) return true;
    setState((s) => ({ ...s, saving: true }));
    try {
      const saved = await replaceScenarioDraft(draft.name, draft.revision, draftContent);
      setDraft(saved);
      setDraftContent(saved.content);
      setDraftDirty(false);
      return true;
    } catch (err) {
      showError(err);
      return false;
    } finally {
      setState((s) => ({ ...s, saving: false }));
    }
  }, [draft, draftContent, draftDirty, showError]);

  const cancelWizard = useCallback(async () => {
    if (draft) await discardDraft(draft);
    setDraft(null);
    setDraftContent('');
    setDraftDirty(false);
    setDraftSourceKey(null);
    setStartedSessionId(null);
    setState(initialWizardState);
  }, [draft, discardDraft]);

  const goBack = useCallback(async () => {
    if (CONTENT_EDITING_STEPS.has(WIZARD_STEPS[state.step] ?? '') && !(await saveDraft())) return;
    setState((s) => ({ ...s, step: Math.max(0, s.step - 1) }));
  }, [saveDraft, state.step]);

  const startPreparedSimulation = useCallback(
    async (request: SimulationRequest) => {
      setState((s) => ({ ...s, starting: true }));
      try {
        const started = await startSimulation(request);
        setStartedSessionId(started.sessionId ?? null);
        setState((s) => ({ ...s, step: WIZARD_STEPS.indexOf('finish'), starting: false }));
      } catch (err) {
        showError(err);
        setState((s) => ({ ...s, starting: false }));
      }
    },
    [showError],
  );

  const goNext = useCallback(async () => {
    if (state.step === 0) {
      await prepareConfig();
      return;
    }
    if (CONTENT_EDITING_STEPS.has(WIZARD_STEPS[state.step] ?? '') && !(await saveDraft())) return;
    setState((s) => ({ ...s, step: Math.min(WIZARD_STEPS.length - 1, s.step + 1) }));
  }, [state.step, prepareConfig, saveDraft]);

  // Only offered once a draft exists: before that there is nothing to abandon
  // and the author can simply navigate away.
  const cancelButton = draft ? (
    <Button
      variant="outline"
      data-testid="wizard-cancel-button"
      disabled={state.saving || state.starting}
      onClick={() => void cancelWizard()}
    >
      {t('newSimWizard.cancelLabel')}
    </Button>
  ) : null;

  const currentStep = WIZARD_STEPS[state.step];
  const canProceed =
    state.step === 0 ? isTemplateStepComplete(state) && !state.starting : !state.saving;

  if (simStatus === null) {
    return (
      <Card className="border-status-warning/30 bg-status-warning/20">
        <CardContent className="stack">
          <SmallText className="text-status-warning">
            {t('runtime.daemonModeWarning')} {t('runtime.daemonModeInstructions')}
          </SmallText>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="stack-xl">
      <WizardStepper steps={steps} currentIndex={state.step} />

      <div data-testid="wizard-step-panel">
        {state.step === 0 && (
          <TemplateStep
            state={state}
            onSelectTemplate={(template: Template) =>
              setState((s) => ({
                ...s,
                source: 'template',
                template,
                userConfig: null,
                uploadFile: null,
              }))
            }
            onSelectUserConfig={(userConfig: LibraryNetwork) =>
              setState((s) => ({
                ...s,
                source: 'userConfig',
                userConfig,
                template: null,
                uploadFile: null,
              }))
            }
            onUpload={(file: File | null) =>
              setState((s) => ({
                ...s,
                source: file ? 'upload' : s.source,
                uploadFile: file,
                template: file ? null : s.template,
                userConfig: file ? null : s.userConfig,
              }))
            }
            onSelectEmpty={() =>
              setState((s) => ({
                ...s,
                source: 'empty',
                template: null,
                userConfig: null,
                uploadFile: null,
              }))
            }
            onSelectFleet={() =>
              setState((s) => ({
                ...s,
                source: 'generated',
                template: null,
                userConfig: null,
                uploadFile: null,
              }))
            }
            onFleetChange={(fleetRequest) => setState((s) => ({ ...s, fleetRequest }))}
            onInterfaceChange={(iface: string) =>
              setState((s) => ({ ...s, selectedInterface: iface }))
            }
          />
        )}
        {currentStep === 'devices' && draft && (
          <DevicesStep
            draftName={draft.name}
            draft={draft}
            content={draftContent}
            dirty={draftDirty}
            saving={state.saving}
            onChange={(content) => {
              setDraftContent(content);
              setDraftDirty(content !== draft.content);
            }}
            onSave={saveDraft}
            onDraftUpdate={(updated) => {
              setDraft(updated);
              setDraftContent(updated.content);
              setDraftDirty(false);
            }}
            onBusyChange={(saving) => setState((s) => ({ ...s, saving }))}
          />
        )}
        {currentStep === 'networks' && draft && (
          <NetworksStep
            content={draftContent}
            onChange={(content) => {
              setDraftContent(content);
              setDraftDirty(content !== draft.content);
            }}
          />
        )}
        {currentStep === 'protocols' && draft && (
          <ProtocolsStep
            content={draftContent}
            onChange={(content) => {
              setDraftContent(content);
              setDraftDirty(content !== draft.content);
            }}
          />
        )}
        {currentStep === 'review' && draft && (
          <ReviewStep name={draft.name} content={draft.content} />
        )}
        {currentStep === 'preflight' && draft && (
          <PreflightStep
            request={{ interface: state.selectedInterface, configData: draft.content }}
            onStart={(request) => void startPreparedSimulation(request)}
            starting={state.starting}
          />
        )}
        {currentStep === 'finish' && draft && (
          <FinishStep draftName={draft.name} sessionId={startedSessionId} />
        )}
      </div>

      {currentStep === 'preflight' && (
        <div className="flex justify-start gap-default">
          <Button
            variant="outline"
            data-testid="wizard-back-button"
            disabled={state.saving}
            onClick={() => void goBack()}
          >
            {t('newSimWizard.backLabel')}
          </Button>
          {cancelButton}
        </div>
      )}
      {currentStep !== 'preflight' && currentStep !== 'finish' && (
        <div className="flex justify-between gap-default">
          <div className="flex gap-default">
            <Button
              variant="outline"
              data-testid="wizard-back-button"
              disabled={state.step === 0 || state.saving}
              onClick={() => void goBack()}
            >
              {t('newSimWizard.backLabel')}
            </Button>
            {cancelButton}
          </div>
          <Button
            tone="violet"
            data-testid="wizard-next-button"
            disabled={!canProceed}
            onClick={() => void goNext()}
          >
            {state.step === 0 && state.starting
              ? t('newSimWizard.template.startingLabel')
              : currentStep === 'devices' && state.saving
                ? t('newSimWizard.devices.savingLabel')
                : t('newSimWizard.nextLabel')}
          </Button>
        </div>
      )}
    </div>
  );
};

export default NewSimulationWizardPage;
