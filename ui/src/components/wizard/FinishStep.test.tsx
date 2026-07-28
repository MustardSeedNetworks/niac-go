/**
 * FinishStep.test.tsx
 *
 * The finish step confirms the saved draft used for the running simulation
 * and hands the operator to runtime monitoring.
 */
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { describe, expect, it } from 'vitest';
import '../../i18n';
import { FinishStep } from './FinishStep';

function renderStep() {
  return render(
    <MemoryRouter>
      <FinishStep draftName="campus-draft" />
    </MemoryRouter>,
  );
}

describe('FinishStep', () => {
  it('shows the saved draft used to start the simulation', () => {
    renderStep();
    expect(screen.getByTestId('wizard-finish-draft-name')).toHaveTextContent('campus-draft');
  });

  it('links to the existing Simulation page', () => {
    renderStep();
    expect(screen.getByTestId('wizard-goto-runtime').closest('a')).toHaveAttribute(
      'href',
      '/runtime',
    );
  });
});
