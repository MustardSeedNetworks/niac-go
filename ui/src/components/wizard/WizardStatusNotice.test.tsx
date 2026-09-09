import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import '../../i18n';
import { ApiError, NetworkError, TimeoutError } from '../../api/errors';
import { WizardStatusNotice } from './WizardStatusNotice';

describe('WizardStatusNotice', () => {
  it.each([
    { name: 'loading', loading: true, error: null, text: 'Checking simulation availability…' },
    {
      name: 'failed',
      loading: false,
      error: new ApiError('Access denied', 403),
      text: 'Access denied',
    },
    { name: 'unavailable', loading: false, error: new NetworkError(), text: 'Cannot reach NIAC' },
    { name: 'timeout', loading: false, error: new TimeoutError(), text: 'Cannot reach NIAC' },
    {
      name: 'wrong mode',
      loading: false,
      error: new ApiError('Not implemented', 501),
      text: 'Start NIAC in daemon mode',
    },
    { name: 'missing result', loading: false, error: null, text: 'Cannot reach NIAC' },
  ])('explains $name without showing the editor', ({ loading, error, text }) => {
    render(<WizardStatusNotice loading={loading} error={error} />);
    expect(screen.getByTestId('wizard-status-notice')).toHaveTextContent(text);
  });
});
