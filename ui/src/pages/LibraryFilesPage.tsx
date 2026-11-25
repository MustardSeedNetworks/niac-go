import { RefreshCw, RotateCcw, Sparkles } from 'lucide-react';
import { useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { type LibraryFileEntry, revertWalk, type WalkProvenance } from '../api/library-client';
import { GlossaryPopover } from '../components/GlossaryPopover';
import { ContentBundleUploader } from '../components/library/ContentBundleUploader';
import { RequireScope } from '../components/ui/RequireScope';
import { useLibraryResource } from '../hooks/usePageResources';
import { useToast } from '../hooks/useToast';
import { useWalkSanitize } from '../hooks/useWalkSanitize';
import { Alert } from '../shared/Alert';
import { CategoryChip } from '../shared/CategoryChip';
import { ConfirmDialog } from '../shared/ConfirmDialog';
import { DataTable, type DataTableColumn } from '../shared/DataTable';
import { EmptyState } from '../shared/EmptyState';
import { SearchInput } from '../shared/SearchInput';
import { Skeleton } from '../shared/Skeleton';
import { StatusPill } from '../shared/StatusPill';
import { ActionButton } from '../ui/ActionButton';
import { ContainedTable } from '../ui/ContainedTable';
import { useDialogLabels } from '../ui/dialogLabels';
import { useTableLabels } from '../ui/tableLabels';

/**
 * LibraryFilesPage is the shared browser for the read-only library
 * subdirs (walks/, pcaps/). It exists once and is wrapped twice in
 * pageRegistry so the route system stays declarative.
 *
 * Per #548 PR 3 the page started intentionally minimal: list + search +
 * total stats, with upload deferred. #897 L3b adds the content-bundle
 * install control (admin-scoped, gated by <RequireScope>) — the
 * air-gapped/manual-install path for the whole library (networks +
 * walks + pcaps at once) — above the file listing.
 *
 * Walks additionally carry a preserve-once "original" (see
 * library.PreserveOriginal server-side): once a walk has been
 * overwritten, its `edited` flag flips true and a Revert control
 * appears so it can be restored to the pristine copy.
 */

interface Props {
  kind: 'walks' | 'pcaps';
}

function LibraryFilesView({ kind }: Props) {
  const dialogLabels = useDialogLabels();
  const { t: tPages } = useTranslation('pages');
  const { t: tCommon } = useTranslation('common');
  const { data, loading, refetch, error } = useLibraryResource(kind, {
    intervalMs: 30000,
    errorToast: true,
  });
  const [search, setSearch] = useState('');
  const [revertTarget, setRevertTarget] = useState<string | null>(null);
  const [reverting, setReverting] = useState(false);
  const { showError } = useToast();
  const sanitizeState = useWalkSanitize(refetch);

  const entries = data ?? [];
  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase();
    if (!q) return entries;
    return entries.filter((e) => e.name.toLowerCase().includes(q));
  }, [entries, search]);

  const totalBytes = useMemo(() => filtered.reduce((acc, e) => acc + e.sizeBytes, 0), [filtered]);

  const handleConfirmRevert = async () => {
    if (!revertTarget || reverting) return;
    setReverting(true);
    try {
      await revertWalk(revertTarget);
      setRevertTarget(null);
      await refetch();
    } catch (err) {
      showError(err);
    } finally {
      setReverting(false);
    }
  };

  const kindLabel =
    kind === 'walks' ? tPages('libraryFiles.walksKind') : tPages('libraryFiles.pcapsKind');
  const title =
    kind === 'walks' ? tPages('libraryFiles.walksTitle') : tPages('libraryFiles.pcapsTitle');
  const emptyHint =
    kind === 'walks'
      ? tPages('libraryFiles.walksEmptyHint')
      : tPages('libraryFiles.pcapsEmptyHint');
  const labels = useTableLabels<LibraryFileEntry>(kindLabel, (entry) =>
    tPages('libraryFiles.selectRowAriaLabel', { name: entry.name }),
  );

  const columns: DataTableColumn<LibraryFileEntry>[] = [
    {
      key: 'name',
      header: tPages('libraryFiles.nameHeader'),
      sortValue: (entry) => entry.name,
      cell: (entry) => (
        <span className="inline-flex items-center gap-2 font-mono text-xs">
          {entry.name}
          {kind === 'walks' && entry.edited && (
            <StatusPill tone="warning" label={tPages('libraryFiles.editedTag')} />
          )}
        </span>
      ),
    },
    {
      key: 'size',
      header: tPages('libraryFiles.sizeHeader'),
      align: 'right',
      sortValue: (entry) => entry.sizeBytes,
      cell: (entry) => <span className="tabular-nums">{humanBytes(entry.sizeBytes)}</span>,
    },
    {
      key: 'source',
      header: tPages('libraryFiles.sourceHeader'),
      cell: (entry) => <SourceBadge source={entry.source} />,
    },
    ...(kind === 'walks'
      ? [
          {
            key: 'content',
            header: tPages('libraryFiles.contentHeader'),
            cell: (entry: LibraryFileEntry) => <ProvenanceBadge provenance={entry.provenance} />,
          },
        ]
      : []),
    {
      key: 'modified',
      header: tPages('libraryFiles.modifiedHeader'),
      sortValue: (entry) => entry.modifiedAt,
      cell: (entry) => (
        <span className="whitespace-nowrap text-xs text-text-muted">
          {new Date(entry.modifiedAt).toLocaleString()}
        </span>
      ),
    },
  ];

  const rowActions =
    kind === 'walks'
      ? (entry: LibraryFileEntry) => (
          <>
            <ActionButton
              action="edit"
              variant="ghost"
              size="sm"
              icon={<Sparkles className="h-3.5 w-3.5" />}
              onClick={() => sanitizeState.setSanitizeTarget(entry.name)}
              aria-label={tPages('libraryFiles.sanitizeLabel', { name: entry.name })}
              data-testid={`sanitize-walk-${entry.name}`}
            >
              {tPages('libraryFiles.sanitizeButton')}
            </ActionButton>
            {entry.edited && (
              <ActionButton
                action="edit"
                variant="ghost"
                size="sm"
                icon={<RotateCcw className="h-3.5 w-3.5" />}
                onClick={() => setRevertTarget(entry.name)}
                aria-label={tPages('libraryFiles.revertLabel', { name: entry.name })}
                data-testid={`revert-walk-${entry.name}`}
              >
                {tPages('libraryFiles.revertButton')}
              </ActionButton>
            )}
          </>
        )
      : undefined;

  return (
    <div className="stack-xl">
      <RequireScope min="admin">
        <ContentBundleUploader />
      </RequireScope>

      <div className="stack">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <p className="flex items-center gap-1 text-sm text-text-muted">
            {tPages('libraryFiles.fileCount', { count: entries.length })} · {humanBytes(totalBytes)}
            <GlossaryPopover term="librarySource" />
          </p>
          <div className="flex min-w-0 flex-1 flex-wrap items-center justify-end gap-2">
            {kind === 'walks' && sanitizeState.selected.size > 0 && (
              <>
                <span className="text-sm text-text-muted">
                  {tPages('libraryFiles.selectedCount', { count: sanitizeState.selected.size })}
                </span>
                <ActionButton
                  action="edit"
                  variant="secondary"
                  size="sm"
                  icon={<Sparkles className="h-4 w-4" />}
                  onClick={() => sanitizeState.setBatchConfirmOpen(true)}
                  data-testid="sanitize-selected-walks"
                >
                  {tPages('libraryFiles.sanitizeSelectedButton')}
                </ActionButton>
              </>
            )}
            <SearchInput
              value={search}
              onValueChange={setSearch}
              label={tPages('libraryFiles.filterAriaLabel')}
              clearLabel={tCommon('accessibility.clearSearch')}
              placeholder={tPages('libraryFiles.searchPlaceholder')}
              className="min-w-48 flex-1 sm:max-w-64"
            />
            <ActionButton
              variant="ghost"
              size="sm"
              onClick={refetch}
              icon={<RefreshCw className="h-4 w-4" />}
              disabled={loading}
              title={tPages('libraryFiles.refreshTitle')}
            >
              {tPages('libraryFiles.refreshButton')}
            </ActionButton>
          </div>
        </div>

        {loading && entries.length === 0 ? (
          <div role="status" aria-label={tPages('libraryFiles.loadingKind', { kind: kindLabel })}>
            <Skeleton className="h-40 w-full" />
          </div>
        ) : error ? (
          <Alert tone="error">{tPages('libraryFiles.loadFailed', { error: error.message })}</Alert>
        ) : entries.length === 0 ? (
          <EmptyState
            title={tPages('libraryFiles.noneInstalled', { kind: kindLabel })}
            hint={emptyHint}
          />
        ) : (
          <ContainedTable>
            <DataTable
              rows={filtered}
              columns={columns}
              getRowKey={(entry) => entry.name}
              caption={title}
              labels={labels}
              data-testid={`library-${kind}-table`}
              {...(kind === 'walks'
                ? {
                    selection: {
                      selectedKeys: sanitizeState.selected,
                      onChange: sanitizeState.setSelected,
                    },
                  }
                : {})}
              {...(rowActions ? { rowActions } : {})}
              empty={
                <EmptyState title={tPages('libraryFiles.noMatchEntries', { query: search })} />
              }
            />
          </ContainedTable>
        )}
      </div>

      {kind === 'walks' && (
        <ConfirmDialog
          {...dialogLabels}
          open={revertTarget !== null}
          onConfirm={() => void handleConfirmRevert()}
          onCancel={() => setRevertTarget(null)}
          title={tPages('libraryFiles.revertModalTitle')}
          message={
            <p>
              {tPages('libraryFiles.revertModalMessagePrefix')}{' '}
              <span className="font-mono">{revertTarget}</span>{' '}
              {tPages('libraryFiles.revertModalMessageSuffix')}
            </p>
          }
          confirmLabel={
            reverting ? tPages('libraryFiles.revertingButton') : tPages('libraryFiles.revertButton')
          }
        />
      )}

      {kind === 'walks' && (
        <ConfirmDialog
          {...dialogLabels}
          open={sanitizeState.sanitizeTarget !== null}
          onConfirm={() => void sanitizeState.confirmSanitize()}
          onCancel={() => sanitizeState.setSanitizeTarget(null)}
          title={tPages('libraryFiles.sanitizeModalTitle')}
          variant="primary"
          message={
            <p>
              {tPages('libraryFiles.sanitizeModalMessagePrefix')}{' '}
              <span className="font-mono">{sanitizeState.sanitizeTarget}</span>{' '}
              {tPages('libraryFiles.sanitizeModalMessageSuffix')}
            </p>
          }
          confirming={sanitizeState.sanitizing}
          confirmLabel={tPages('libraryFiles.sanitizeButton')}
        />
      )}

      {kind === 'walks' && (
        <ConfirmDialog
          {...dialogLabels}
          open={sanitizeState.batchConfirmOpen}
          onConfirm={() => void sanitizeState.confirmBatchSanitize()}
          onCancel={() => sanitizeState.setBatchConfirmOpen(false)}
          title={tPages('libraryFiles.sanitizeBatchModalTitle')}
          variant="primary"
          message={tPages('libraryFiles.sanitizeBatchModalMessage', {
            count: sanitizeState.selected.size,
          })}
          confirming={sanitizeState.batchSanitizing}
          confirmLabel={tPages('libraryFiles.sanitizeSelectedButton')}
        />
      )}
    </div>
  );
}

function SourceBadge({ source }: { source: LibraryFileEntry['source'] }) {
  const { t } = useTranslation('pages');
  const labels: Record<LibraryFileEntry['source'], string> = {
    starter: t('libraryFiles.sourceStarter'),
    bundle: t('libraryFiles.sourceBundle'),
    user: t('libraryFiles.sourceUser'),
  };
  return <CategoryChip label={labels[source]} />;
}

/**
 * Half the shipped walks are generated rather than captured, and the catalog
 * gave no way to tell — someone picking a vendor walk had no way to know
 * nothing had ever measured a device of that model.
 */
function ProvenanceBadge({ provenance }: { provenance: LibraryFileEntry['provenance'] }) {
  const { t } = useTranslation('pages');
  if (!provenance) {
    return <span className="text-xs text-text-muted">{t('libraryFiles.contentUnknown')}</span>;
  }
  const tones: Record<WalkProvenance, 'success' | 'warning'> = {
    captured: 'success',
    generated: 'warning',
  };
  const labels: Record<WalkProvenance, string> = {
    captured: t('libraryFiles.contentCaptured'),
    generated: t('libraryFiles.contentGenerated'),
  };
  return <StatusPill tone={tones[provenance]} label={labels[provenance]} />;
}

function humanBytes(n: number): string {
  if (n >= 1 << 30) return `${(n / (1 << 30)).toFixed(1)} GB`;
  if (n >= 1 << 20) return `${(n / (1 << 20)).toFixed(1)} MB`;
  if (n >= 1 << 10) return `${(n / (1 << 10)).toFixed(1)} KB`;
  return `${n} B`;
}

export function LibraryWalksPage() {
  return <LibraryFilesView kind="walks" />;
}
export function LibraryPcapsPage() {
  return <LibraryFilesView kind="pcaps" />;
}
