import { useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { useUIStore } from '../stores/ui-store';
import { getErrorMessage } from '../utils/format';

/**
 * useErrorToast — the shared seam for page-level "the request failed"
 * errors (mutation catch blocks, imperative fetches outside
 * useApiResource). Surfaces the error as a global toast via
 * ui/ToastContainer instead of a bespoke per-page banner.
 *
 * Field-scoped validation errors (form field messages, upload
 * validation) are NOT what this is for — those stay inline next to the
 * field they describe.
 */
export function useErrorToast() {
  const addNotification = useUIStore((s) => s.addNotification);
  const { t } = useTranslation('common');

  return useCallback(
    (err: unknown, title?: string) => {
      addNotification({
        type: 'error',
        title: title ?? t('toast.requestFailedTitle'),
        message: getErrorMessage(err),
      });
    },
    [addNotification, t],
  );
}
