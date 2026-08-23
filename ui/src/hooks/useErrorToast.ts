import { useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { isApiError } from '../api/errors';
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
      // Surface the server's details[] rather than only the generic message.
      // The API says exactly what is wrong ("line 65: field enabled not found
      // in type converter.DhcpServer"); dropping it left users with nothing to
      // act on (D3).
      const details = isApiError(err)
        ? err.details.map((detail) =>
            detail.line ? `line ${detail.line}: ${detail.issue}` : detail.issue,
          )
        : undefined;

      addNotification({
        type: 'error',
        title: title ?? t('toast.requestFailedTitle'),
        message: getErrorMessage(err),
        details: details && details.length > 0 ? details : undefined,
      });
    },
    [addNotification, t],
  );
}
