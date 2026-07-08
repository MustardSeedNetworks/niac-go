import { type FC, useCallback, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { applyTemplate, startSimulation, uploadUserConfig } from '../api/client';
import type { Template, UserConfig } from '../api/types';
import { DevicesStep } from '../components/wizard/DevicesStep';
import { FinishStep } from '../components/wizard/FinishStep';
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
import { fileToText } from '../utils/file';

// Placeholder skeleton for the "start empty" source — the smallest valid
// config the daemon will load. Not a reimplementation of `niac config
// generate`; it's just the seed content for a config the wizard's
// existing-component steps then build up.
const EMPTY_CONFIG_YAML = 'devices: []\n';

/**
 * NewSimulationWizard — a thin orchestration layer over existing pages
 * and endpoints (issue #958). It does not introduce any new backend route
 * or reimplement device/config editing; it sequences:
 *
 *   1. Template  — ConfigPicker (pick a template / saved config / upload / empty)
 *   2. Devices   — the existing Device Library page (device CRUD)
 *   3. Protocols — the existing DeviceTable's Protocols column, read-only
 *   4. Review    — the existing SelectedNetworkPreview
 *   5. Finish    — the existing config-save endpoint, then hand off to
 *                  the existing Simulation page
 *
 * Device CRUD in this app only operates on the daemon's single active
 * config (`s.configPath()` server-side) — there's no "stage a draft
 * config" endpoint. So step 1's Next action materialises the picked
 * source as a real (non-built-in) config file and starts the simulation
 * against it, which is what unlocks steps 2-3's editing. See the PR
 * description for the full rationale.
 */
export const NewSimulationWizardPage: FC = () => {
  const { t } = useTranslation('pages');
  const { data: simStatus } = useSimulationStatus();
  const showError = useErrorToast();
  const [state, setState] = useState<WizardState>(initialWizardState);

  const steps = WIZARD_STEPS.map((id) => ({
    id,
    label: t(`newSimWizard.steps.${id}`),
  }));

  const goBack = useCallback(() => {
    setState((s) => ({ ...s, step: Math.max(0, s.step - 1) }));
  }, []);

  const materializeAndStart = useCallback(async () => {
    setState((s) => ({ ...s, starting: true }));
    try {
      let configPath: string | undefined;
      let configData: string | undefined;

      if (state.source === 'template' && state.template) {
        // Copy the template into a new saved config first (existing
        // "Save As" endpoint) instead of passing templateName straight to
        // startSimulation — the latter points the daemon's active config
        // path at the built-in template file itself, so any device edit
        // in step 2 would overwrite the shipped template on disk.
        const applied = await applyTemplate({ templateName: state.template.name });
        configPath = applied.configPath;
      } else if (state.source === 'userConfig' && state.userConfig) {
        configPath = state.userConfig.path;
      } else if (state.source === 'upload' && state.uploadFile) {
        configData = await fileToText(state.uploadFile);
      } else if (state.source === 'empty') {
        const created = await uploadUserConfig({
          name: `new-simulation-${Date.now()}`,
          content: EMPTY_CONFIG_YAML,
        });
        configPath = created.path;
      } else {
        throw new Error(t('newSimWizard.template.errorNoSource'));
      }

      await startSimulation({
        interface: state.selectedInterface,
        configPath,
        configData,
      });

      setState((s) => ({ ...s, step: 1, configPath: configPath ?? null, starting: false }));
    } catch (err) {
      showError(err);
      setState((s) => ({ ...s, starting: false }));
    }
  }, [state, showError, t]);

  const goNext = useCallback(() => {
    if (state.step === 0) {
      void materializeAndStart();
      return;
    }
    setState((s) => ({ ...s, step: Math.min(WIZARD_STEPS.length - 1, s.step + 1) }));
  }, [state.step, materializeAndStart]);

  const canProceed = state.step === 0 ? isTemplateStepComplete(state) && !state.starting : true;

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
            onSelectUserConfig={(userConfig: UserConfig) =>
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
            onInterfaceChange={(iface: string) =>
              setState((s) => ({ ...s, selectedInterface: iface }))
            }
          />
        )}
        {state.step === 1 && <DevicesStep />}
        {state.step === 2 && <ProtocolsStep />}
        {state.step === 3 && <ReviewStep />}
        {state.step === 4 && <FinishStep />}
      </div>

      {state.step < WIZARD_STEPS.length - 1 && (
        <div className="flex justify-between gap-default">
          <Button
            variant="outline"
            data-testid="wizard-back-button"
            disabled={state.step === 0}
            onClick={goBack}
          >
            {t('newSimWizard.backLabel')}
          </Button>
          <Button
            tone="violet"
            data-testid="wizard-next-button"
            disabled={!canProceed}
            onClick={goNext}
          >
            {state.step === 0 && state.starting
              ? t('newSimWizard.template.startingLabel')
              : t('newSimWizard.nextLabel')}
          </Button>
        </div>
      )}
      {state.step === WIZARD_STEPS.length - 1 && (
        <div className="flex justify-start">
          <Button variant="outline" data-testid="wizard-back-button" onClick={goBack}>
            {t('newSimWizard.backLabel')}
          </Button>
        </div>
      )}
    </div>
  );
};

export default NewSimulationWizardPage;
