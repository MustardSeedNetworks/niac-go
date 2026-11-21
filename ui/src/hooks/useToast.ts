import { useCallback, useContext } from 'react';
import { useTranslation } from 'react-i18next';
import { isApiError } from '../api/errors';
import { ToastContext, type ToastContextValue } from '../ui/toastContext';
import { getErrorMessage } from '../utils/format';

export interface UseToastResult extends ToastContextValue {
  /**
   * The shared seam for page-level "the request failed" errors (mutation
   * catch blocks, imperative fetches outside useApiResource). Field-scoped
   * validation errors stay inline next to the field they describe.
   */
  showError: (err: unknown, title?: string) => void;
}

export function useToast(): UseToastResult {
  const context = useContext(ToastContext);
  if (!context) {
    throw new Error('useToast must be used within a ToastProvider');
  }
  const { addToast, removeToast } = context;
  const { t } = useTranslation('common');

  const showError = useCallback(
    (err: unknown, title?: string) => {
      const details = isApiError(err)
        ? err.details.map((detail) =>
            detail.line ? `line ${detail.line}: ${detail.issue}` : detail.issue,
          )
        : [];
      addToast({
        type: 'error',
        title: title ?? t('toast.requestFailedTitle'),
        message: getErrorMessage(err),
        details: details.length > 0 ? details : undefined,
      });
    },
    [addToast, t],
  );

  return { addToast, removeToast, showError };
}
