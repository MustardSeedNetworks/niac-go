import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import type { ReactNode } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { ApiError } from '../api/errors';

const { mockValidate, mockSetRuntimeAPIToken, mockClearRuntimeAPIToken, mockT } = vi.hoisted(
  () => ({
    mockValidate: vi.fn(),
    mockSetRuntimeAPIToken: vi.fn(),
    mockClearRuntimeAPIToken: vi.fn(),
    mockT: (key: string) => key,
  }),
);

vi.mock('../api/requestCore', () => ({
  AUTH_FAILURE_EVENT: 'niac:authentication-required',
  AUTH_DISCONNECT_EVENT: 'niac:disconnect',
  clearRuntimeAPIToken: mockClearRuntimeAPIToken,
  setRuntimeAPIToken: mockSetRuntimeAPIToken,
  validateRuntimeAuthentication: mockValidate,
}));

vi.mock('react-i18next', () => ({
  useTranslation: () => ({ t: mockT }),
}));

import { AuthGate } from './AuthGate';

const renderGate = (children: ReactNode = <div>protected content</div>) =>
  render(<AuthGate>{children}</AuthGate>);

describe('AuthGate', () => {
  beforeEach(() => {
    mockValidate.mockReset();
    mockSetRuntimeAPIToken.mockReset();
    mockClearRuntimeAPIToken.mockReset();
  });

  it('passes through when the daemon permits an unauthenticated loopback session', async () => {
    mockValidate.mockResolvedValueOnce(undefined);
    renderGate();
    expect(await screen.findByText('protected content')).toBeVisible();
  });

  it('identifies the product on the sign-in screen', async () => {
    mockValidate.mockRejectedValueOnce(new ApiError('Unauthorized', 401));
    renderGate();

    expect(await screen.findByRole('heading', { level: 1 })).toHaveTextContent('app.title');
    expect(screen.getByTestId('product-mark')).toBeVisible();
    expect(screen.getByText('app.tagline')).toBeVisible();
    expect(screen.getByRole('heading', { level: 2 })).toHaveTextContent('auth.title');
  });

  it('prompts on 401 and validates a bearer token before rendering the app', async () => {
    mockValidate
      .mockRejectedValueOnce(new ApiError('Unauthorized', 401))
      .mockResolvedValueOnce({ scope: 'read-write' });
    renderGate();

    const input = await screen.findByLabelText('auth.tokenLabel');
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
    fireEvent.change(input, { target: { value: 'entered-token' } });
    fireEvent.click(screen.getByRole('button', { name: 'auth.connect' }));

    await waitFor(() => {
      expect(mockSetRuntimeAPIToken).toHaveBeenCalledWith('entered-token');
    });
    expect(await screen.findByText('protected content')).toBeVisible();
  });

  it('clears a rejected token and keeps the prompt visible', async () => {
    mockValidate.mockRejectedValue(new ApiError('Unauthorized', 401));
    renderGate();

    const input = await screen.findByLabelText('auth.tokenLabel');
    fireEvent.change(input, { target: { value: 'bad-token' } });
    fireEvent.click(screen.getByRole('button', { name: 'auth.connect' }));

    expect(await screen.findByRole('alert')).toHaveTextContent('auth.invalidToken');
    expect(mockClearRuntimeAPIToken).toHaveBeenCalled();
    expect(screen.queryByText('protected content')).not.toBeInTheDocument();
  });

  it('reports a connection failure without discarding the candidate token', async () => {
    mockValidate
      .mockRejectedValueOnce(new ApiError('Unauthorized', 401))
      .mockRejectedValueOnce(new TypeError('Failed to fetch'));
    renderGate();

    const input = await screen.findByLabelText('auth.tokenLabel');
    fireEvent.change(input, { target: { value: 'valid-token' } });
    fireEvent.click(screen.getByRole('button', { name: 'auth.connect' }));

    expect(await screen.findByRole('alert')).toHaveTextContent('auth.connectionFailed');
    expect(mockClearRuntimeAPIToken).not.toHaveBeenCalled();
    expect(input).toHaveValue('valid-token');
  });

  it('returns to the connect prompt, with no error, when the operator disconnects', async () => {
    mockValidate.mockResolvedValueOnce(undefined);
    renderGate();
    expect(await screen.findByText('protected content')).toBeVisible();

    act(() => {
      window.dispatchEvent(new Event('niac:disconnect'));
    });

    expect(await screen.findByLabelText('auth.tokenLabel')).toBeVisible();
    expect(screen.queryByText('protected content')).not.toBeInTheDocument();
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
  });
});
