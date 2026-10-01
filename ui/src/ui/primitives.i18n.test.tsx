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
import { Alert } from './Alert';
import { BaseCard } from './BaseCard';
import { ConnectionStatus } from './ConnectionStatus';
import { SearchInput } from './Input';
import { StatusBadge } from './StatusBadge';

const noop = () => {};

beforeEach(async () => {
  await i18n.changeLanguage('es');
});

afterEach(async () => {
  await i18n.changeLanguage('en');
});

describe('shared primitives in es', () => {
  it('names the dismiss buttons in Spanish', () => {
    render(
      <>
        <Alert status="info" onDismiss={noop}>
          x
        </Alert>
        <DeviceStatusMessage message={{ type: 'success', text: 'ok' }} onDismiss={noop} />
      </>,
    );

    expect(screen.getByRole('button', { name: 'Descartar alerta' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Descartar mensaje' })).toBeInTheDocument();
  });

  it('names every status badge in Spanish', () => {
    render(
      <>
        <StatusBadge status="success" />
        <StatusBadge status="warning" variant="dot" />
        <StatusBadge status="error" />
        <StatusBadge status="unknown" />
        <StatusBadge status="loading" />
      </>,
    );

    for (const name of [
      'Estado: correcto',
      'Estado: advertencia',
      'Estado: error',
      'Estado: desconocido',
      'Estado: cargando',
    ]) {
      expect(screen.getByRole('img', { name }), name).toBeInTheDocument();
    }
  });

  it('describes the connection in Spanish', () => {
    render(<ConnectionStatus status="disconnected" />);

    expect(screen.getByTestId('connection-status')).toHaveAccessibleName('Servidor inaccesible');
    expect(screen.getByText('Sin conexión')).toBeInTheDocument();
  });

  it('localizes the search placeholder and clear button', () => {
    render(<SearchInput value="tcp" onChange={noop} />);

    expect(screen.getByPlaceholderText('Buscar...')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Limpiar búsqueda' })).toBeInTheDocument();
  });

  it('localizes the default empty message', () => {
    render(
      <BaseCard<string> title="Card" data={null} getStatus={() => 'success'}>
        {(value) => value}
      </BaseCard>,
    );

    expect(screen.getByText('No hay datos disponibles')).toBeInTheDocument();
  });
});
