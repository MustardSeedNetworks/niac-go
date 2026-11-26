import { useCallback, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { sanitizeWalk, sanitizeWalksBatch } from '../api/library-client';
import { useToast } from './useToast';

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
  const { addToast, showError } = useToast();

  const [sanitizeTarget, setSanitizeTarget] = useState<string | null>(null);
  const [sanitizing, setSanitizing] = useState(false);

  const [selected, setSelected] = useState<ReadonlySet<string>>(new Set());
  const [batchConfirmOpen, setBatchConfirmOpen] = useState(false);
  const [batchSanitizing, setBatchSanitizing] = useState(false);

  const clearSelection = useCallback(() => setSelected(new Set()), []);

  const confirmSanitize = useCallback(async () => {
    if (!sanitizeTarget || sanitizing) return;
    setSanitizing(true);
    try {
      const name = sanitizeTarget;
      await sanitizeWalk(name);
      addToast({
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
  }, [sanitizeTarget, sanitizing, addToast, t, refetch, showError]);

  const confirmBatchSanitize = useCallback(async () => {
    if (selected.size === 0 || batchSanitizing) return;
    setBatchSanitizing(true);
    try {
      const names = Array.from(selected);
      const resp = await sanitizeWalksBatch(names);
      addToast({
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
  }, [selected, batchSanitizing, addToast, t, refetch, showError, clearSelection]);

  return {
    sanitizeTarget,
    setSanitizeTarget,
    sanitizing,
    confirmSanitize,
    selected,
    setSelected,
    clearSelection,
    batchConfirmOpen,
    setBatchConfirmOpen,
    batchSanitizing,
    confirmBatchSanitize,
  };
}
