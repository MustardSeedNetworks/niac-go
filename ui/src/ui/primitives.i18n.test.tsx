/**
 * primitives.i18n.test.tsx — niac-go#2403: the shared primitives carried
 * English default aria-labels and status labels, so every page built on them
 * read English to a screen reader in Spanish. These render each one under
 * `es` and read the accessible names an operator actually hears.
 */
import { render, screen } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { DeviceStatusMessage } from '../components/device-list/DeviceStatusMessage';
import i18n from '../i18n';
import { ConnectionStatus } from './ConnectionStatus';

const noop = () => {};

beforeEach(async () => {
  await i18n.changeLanguage('es');
});

afterEach(async () => {
  await i18n.changeLanguage('en');
});

describe('shared primitives in es', () => {
  it('names the dismiss button in Spanish', () => {
    render(<DeviceStatusMessage message={{ type: 'success', text: 'ok' }} onDismiss={noop} />);

    expect(screen.getByRole('button', { name: 'Descartar mensaje' })).toBeInTheDocument();
  });

  it('describes the connection in Spanish', () => {
    render(<ConnectionStatus status="disconnected" />);

    // Visible word is "Sin conexión", which the longer cue does not contain.
    expect(screen.getByTestId('connection-status')).toHaveAccessibleName(
      'Servidor inaccesible Sin conexión',
    );
    expect(screen.getByText('Sin conexión')).toBeInTheDocument();
  });
});
