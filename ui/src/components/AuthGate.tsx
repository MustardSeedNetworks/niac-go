import { type FormEvent, type ReactElement, type ReactNode, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { ApiError } from '../api/errors';
import {
  AUTH_FAILURE_EVENT,
  clearRuntimeAPIToken,
  setRuntimeAPIToken,
  validateRuntimeAuthentication,
} from '../api/requestCore';

interface AuthGateProps {
  children: ReactNode;
}

type AuthState = 'checking' | 'prompt' | 'authenticated';

export function AuthGate({ children }: AuthGateProps): ReactElement {
  const { t } = useTranslation('common');
  const [state, setState] = useState<AuthState>('checking');
  const [token, setToken] = useState('');
  const [error, setError] = useState('');

  useEffect(() => {
    const requireAuthentication = () => {
      clearRuntimeAPIToken();
      setState('prompt');
      setError(t('auth.sessionExpired'));
    };
    window.addEventListener(AUTH_FAILURE_EVENT, requireAuthentication);
    return () => window.removeEventListener(AUTH_FAILURE_EVENT, requireAuthentication);
  }, [t]);

  useEffect(() => {
    validateRuntimeAuthentication()
      .then(() => setState('authenticated'))
      .catch((err: unknown) => {
        setState('prompt');
        if (!(err instanceof ApiError && err.status === 401)) {
          setError(t('auth.connectionFailed'));
        }
      });
  }, [t]);

  const connect = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const candidate = token.trim();
    if (!candidate) return;

    setError('');
    setRuntimeAPIToken(candidate);
    try {
      await validateRuntimeAuthentication();
      setToken('');
      setState('authenticated');
    } catch (err: unknown) {
      if (err instanceof ApiError && err.status === 401) {
        clearRuntimeAPIToken();
        setError(t('auth.invalidToken'));
      } else {
        setError(t('auth.connectionFailed'));
      }
      setState('prompt');
    }
  };

  if (state === 'authenticated') {
    return <>{children}</>;
  }

  if (state === 'checking') {
    return (
      <main className="grid min-h-screen place-items-center bg-surface-base text-text-primary">
        <p role="status">{t('auth.checking')}</p>
      </main>
    );
  }

  return (
    <main className="grid min-h-screen place-items-center bg-surface-base px-4 text-text-primary">
      <form
        onSubmit={connect}
        className="w-full max-w-md rounded-xl border border-surface-border bg-surface-raised p-8 shadow-xl"
      >
        <h1 className="font-display text-2xl font-semibold">{t('auth.title')}</h1>
        <p className="mt-2 text-sm text-text-secondary">{t('auth.description')}</p>
        <label htmlFor="api-token" className="mt-6 block text-sm font-medium">
          {t('auth.tokenLabel')}
        </label>
        <input
          id="api-token"
          data-testid="api-token-input"
          type="password"
          autoComplete="off"
          spellCheck={false}
          value={token}
          onChange={(event) => setToken(event.target.value)}
          className="mt-2 w-full rounded-lg border border-surface-border bg-surface-sunken px-3 py-2 font-mono text-sm outline-none focus:border-brand-primary"
        />
        {error && (
          <p role="alert" data-testid="auth-gate-error" className="mt-3 text-sm text-status-error">
            {error}
          </p>
        )}
        <button
          type="submit"
          data-testid="auth-gate-connect"
          disabled={!token.trim()}
          className="mt-6 w-full rounded-lg bg-brand-primary px-4 py-2 font-medium text-text-inverse disabled:opacity-50"
        >
          {t('auth.connect')}
        </button>
        <p className="mt-4 text-xs text-text-muted">{t('auth.memoryOnly')}</p>
      </form>
    </main>
  );
}
