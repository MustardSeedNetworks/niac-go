import { useCallback, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { sanitizeWalk, sanitizeWalksBatch } from '../api/client';
import { useUIStore } from '../stores/ui-store';
import { useErrorToast } from './useErrorToast';

/**
 * useWalkSanitize — sanitize state + handlers for the library walks view
 * (LibraryFilesPage). Mirrors the page's existing revert wiring (confirm
 * modal -> POST -> toast -> refetch) for both a single-row sanitize and a
 * selection-driven batch sanitize, backed by POST
 * /api/v1/library/walks/sanitize and .../sanitize-batch.
 *
 * Both routes preserve each walk's pristine original (idempotent), so a
 * successful sanitize always leaves the existing Revert control available
 * — this hook only triggers the refetch that makes that indicator light
 * up, it doesn't duplicate it.
 */
export function useWalkSanitize(refetch: () => void | Promise<void>) {
  const { t } = useTranslation('pages');
  const addNotification = useUIStore((s) => s.addNotification);
  const showError = useErrorToast();

  const [sanitizeTarget, setSanitizeTarget] = useState<string | null>(null);
  const [sanitizing, setSanitizing] = useState(false);

  const [selected, setSelected] = useState<ReadonlySet<string>>(new Set());
  const [batchConfirmOpen, setBatchConfirmOpen] = useState(false);
  const [batchSanitizing, setBatchSanitizing] = useState(false);

  const toggleRow = useCallback((key: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(key)) {
        next.delete(key);
      } else {
        next.add(key);
      }
      return next;
    });
  }, []);

  const toggleAll = useCallback((allKeys: string[]) => {
    setSelected((prev) => (prev.size === allKeys.length ? new Set() : new Set(allKeys)));
  }, []);

  const clearSelection = useCallback(() => setSelected(new Set()), []);

  const confirmSanitize = useCallback(async () => {
    if (!sanitizeTarget || sanitizing) return;
    setSanitizing(true);
    try {
      const name = sanitizeTarget;
      await sanitizeWalk(name);
      addNotification({
        type: 'success',
        title: t('libraryFiles.sanitizeSuccessTitle'),
        message: t('libraryFiles.sanitizeSuccessMessage', { name }),
      });
      setSanitizeTarget(null);
      await refetch();
    } catch (err) {
      showError(err);
    } finally {
      setSanitizing(false);
    }
  }, [sanitizeTarget, sanitizing, addNotification, t, refetch, showError]);

  const confirmBatchSanitize = useCallback(async () => {
    if (selected.size === 0 || batchSanitizing) return;
    setBatchSanitizing(true);
    try {
      const names = Array.from(selected);
      const resp = await sanitizeWalksBatch(names);
      addNotification({
        type: resp.failed > 0 ? 'warning' : 'success',
        title:
          resp.failed > 0
            ? t('libraryFiles.sanitizeBatchPartialTitle')
            : t('libraryFiles.sanitizeBatchSuccessTitle'),
        message: t('libraryFiles.sanitizeBatchSummary', {
          sanitized: resp.sanitized,
          failed: resp.failed,
        }),
      });
      setBatchConfirmOpen(false);
      clearSelection();
      await refetch();
    } catch (err) {
      showError(err);
    } finally {
      setBatchSanitizing(false);
    }
  }, [selected, batchSanitizing, addNotification, t, refetch, showError, clearSelection]);

  return {
    sanitizeTarget,
    setSanitizeTarget,
    sanitizing,
    confirmSanitize,
    selected,
    toggleRow,
    toggleAll,
    clearSelection,
    batchConfirmOpen,
    setBatchConfirmOpen,
    batchSanitizing,
    confirmBatchSanitize,
  };
}
