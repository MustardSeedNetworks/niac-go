import { Check } from 'lucide-react';
import type { FC } from 'react';
import { iconSizes } from '../../constants/sizes';
import type { WizardStepId } from './wizard-types';

interface WizardStepperProps {
  steps: readonly { id: WizardStepId; label: string }[];
  currentIndex: number;
}

/**
 * Horizontal step indicator for NewSimulationWizard. Purely presentational —
 * navigation happens via the container's Back/Next controls, not by
 * clicking a step directly, so a half-finished step can't be skipped to.
 */
export const WizardStepper: FC<WizardStepperProps> = ({ steps, currentIndex }) => (
  <ol data-testid="wizard-stepper" className="flex flex-wrap items-center gap-compact">
    {steps.map((step, index) => {
      const status = index < currentIndex ? 'done' : index === currentIndex ? 'active' : 'pending';
      return (
        <li key={step.id} className="flex items-center gap-compact">
          <div
            data-testid={`wizard-step-${step.id}`}
            data-status={status}
            className={`flex items-center gap-compact rounded-full border px-3 py-compact text-xs font-medium ${
              status === 'active'
                ? 'border-brand-accent bg-brand-primary/20 text-brand-accent'
                : status === 'done'
                  ? 'border-status-success/40 bg-status-success/10 text-status-success'
                  : 'border-surface-border bg-bg-surface/40 text-text-muted'
            }`}
          >
            {status === 'done' ? (
              <Check className={iconSizes.sm} />
            ) : (
              <span className="tabular-nums">{index + 1}</span>
            )}
            {step.label}
          </div>
          {index < steps.length - 1 && (
            <span className="h-px w-4 bg-surface-border" aria-hidden="true" />
          )}
        </li>
      );
    })}
  </ol>
);
