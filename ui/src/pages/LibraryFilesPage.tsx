import { Database, FileBox, RefreshCw, RotateCcw, Search } from 'lucide-react';
import { type FC, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  fetchLibraryPcaps,
  fetchLibraryWalks,
  type LibraryFileEntry,
  revertWalk,
} from '../api/client';
import { useApiResource } from '../hooks/useApiResource';
import { useErrorToast } from '../hooks/useErrorToast';
import { Button } from '../ui/Button';
import { Card, CardContent } from '../ui/Card';
import { ConfirmModal } from '../ui/ConfirmModal';
import { InfoPopover } from '../ui/InfoPopover';
import { Tag } from '../ui/Tag';
import { H2, SmallText } from '../ui/Typography';

/**
 * LibraryFilesPage is the shared browser for the read-only library
 * subdirs (walks/, pcaps/). It exists once and is wrapped twice in
 * pageRegistry so the route system stays declarative.
 *
 * Per #548 PR 3 the page is intentionally minimal: list + search +
 * total stats. Upload + delete for binary content are deferred to a
 * later PR — the immediate need is for the picker integrations
 * (device editor's SNMP section, traffic/packets PCAP picker) to have
 * a single source of truth.
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
  const { t } = useTranslation('common');
  const { t: tHelp } = useTranslation('help');
  const { t: tPages } = useTranslation('pages');
  const fetcher = kind === 'walks' ? fetchLibraryWalks : fetchLibraryPcaps;
  const { data, loading, refetch, error } = useApiResource(fetcher, [], {
    intervalMs: 30000,
    errorToast: true,
  });
  const [search, setSearch] = useState('');
  const [revertTarget, setRevertTarget] = useState<string | null>(null);
  const [reverting, setReverting] = useState(false);
  const showError = useErrorToast();

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

  const columnCount = kind === 'walks' ? 5 : 4;
  const KindIcon = kind === 'walks' ? Database : FileBox;
  const title =
    kind === 'walks' ? tPages('libraryFiles.walksTitle') : tPages('libraryFiles.pcapsTitle');
  const kindLabel =
    kind === 'walks' ? tPages('libraryFiles.walksKind') : tPages('libraryFiles.pcapsKind');
  const emptyHint =
    kind === 'walks'
      ? tPages('libraryFiles.walksEmptyHint')
      : tPages('libraryFiles.pcapsEmptyHint');

  return (
    <div className="stack-xl">
      <Card className="border-surface-border bg-bg-surface/70">
        <CardContent>
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-comfortable">
            <div className="flex items-center gap-default">
              <div className="pad-xs rounded-lg bg-status-info/20">
                <KindIcon className="w-6 h-6 text-status-info" />
              </div>
              <div>
                <H2>{title}</H2>
                <SmallText className="text-text-muted">
                  {tPages('libraryFiles.fileCount', { count: entries.length })} ·{' '}
                  {humanBytes(totalBytes)}
                </SmallText>
              </div>
            </div>

            <div className="flex items-center gap-compact">
              <div className="relative">
                <Search className="absolute left-2 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-text-muted" />
                <input
                  type="search"
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  placeholder={tPages('libraryFiles.searchPlaceholder')}
                  aria-label={tPages('libraryFiles.filterAriaLabel')}
                  className="w-64 rounded-md border border-surface-border bg-bg-base/40 pl-7 pr-3 py-compact-md text-xs text-text-primary placeholder:text-text-muted focus:border-status-info/40 focus:outline-none focus:ring-1 focus:ring-status-info/30"
                />
              </div>
              <Button
                variant="ghost"
                size="sm"
                onClick={refetch}
                leftIcon={<RefreshCw className="w-4 h-4" />}
                disabled={loading}
                title={tPages('libraryFiles.refreshTitle')}
              >
                {tPages('libraryFiles.refreshButton')}
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card className="border-surface-border bg-bg-surface/70">
        <CardContent>
          {loading && entries.length === 0 ? (
            <div className="py-10 text-center">
              <SmallText className="text-text-muted">
                {tPages('libraryFiles.loadingKind', { kind: kindLabel })}
              </SmallText>
            </div>
          ) : error ? (
            <SmallText className="text-status-error">
              {tPages('libraryFiles.loadFailed', { error: error.message })}
            </SmallText>
          ) : entries.length === 0 ? (
            <div className="py-10 text-center">
              <SmallText className="text-text-muted">
                {tPages('libraryFiles.noneInstalled', { kind: kindLabel })}
              </SmallText>
              <p className="mt-inline text-xs text-text-muted">{emptyHint}</p>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead className="text-xs text-text-muted uppercase tracking-wide">
                  <tr className="border-b border-surface-border">
                    <th className="text-left py-row pr-4">{tPages('libraryFiles.nameHeader')}</th>
                    <th className="text-right py-row pr-4">{tPages('libraryFiles.sizeHeader')}</th>
                    <th className="text-left py-row pr-4">
                      <span className="inline-flex items-center gap-1">
                        {tPages('libraryFiles.sourceHeader')}
                        <InfoPopover
                          label={t('jargon.ariaLabel', { term: 'starter / bundle / user' })}
                          title="starter / bundle / user"
                        >
                          {tHelp('jargon.librarySource')}
                        </InfoPopover>
                      </span>
                    </th>
                    <th className="text-left py-row">{tPages('libraryFiles.modifiedHeader')}</th>
                    {kind === 'walks' && (
                      <th className="text-right py-row">{tPages('libraryFiles.actionsHeader')}</th>
                    )}
                  </tr>
                </thead>
                <tbody className="text-text-primary">
                  {filtered.map((entry) => (
                    <tr key={entry.name} className="border-b border-surface-border last:border-0">
                      <td className="py-row pr-4 font-mono text-xs">
                        <span className="inline-flex items-center gap-compact">
                          {entry.name}
                          {kind === 'walks' && entry.edited && (
                            <Tag colorScheme="yellow">{tPages('libraryFiles.editedTag')}</Tag>
                          )}
                        </span>
                      </td>
                      <td className="py-row pr-4 text-right tabular-nums">
                        {humanBytes(entry.sizeBytes)}
                      </td>
                      <td className="py-row pr-4">
                        <SourceBadge source={entry.source} />
                      </td>
                      <td className="py-row text-xs text-text-muted">
                        {new Date(entry.modifiedAt).toLocaleString()}
                      </td>
                      {kind === 'walks' && (
                        <td className="py-row text-right">
                          {entry.edited && (
                            <Button
                              variant="ghost"
                              size="xs"
                              leftIcon={<RotateCcw className="w-3.5 h-3.5" />}
                              onClick={() => setRevertTarget(entry.name)}
                              aria-label={tPages('libraryFiles.revertLabel', { name: entry.name })}
                              data-testid={`revert-walk-${entry.name}`}
                            >
                              {tPages('libraryFiles.revertButton')}
                            </Button>
                          )}
                        </td>
                      )}
                    </tr>
                  ))}
                  {search.trim() !== '' && filtered.length === 0 && (
                    <tr>
                      <td
                        colSpan={columnCount}
                        className="py-6 text-center text-xs text-text-muted"
                      >
                        {tPages('libraryFiles.noMatchEntries', { query: search })}
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>

      {kind === 'walks' && (
        <ConfirmModal
          isOpen={revertTarget !== null}
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
    </div>
  );
}

const SourceBadge: FC<{ source: LibraryFileEntry['source'] }> = ({ source }) => {
  const { t } = useTranslation('pages');
  const styles: Record<LibraryFileEntry['source'], string> = {
    starter: 'border-brand-primary/40 bg-brand-primary/10 text-brand-accent',
    bundle: 'border-status-info/40 bg-status-info/10 text-status-info',
    user: 'border-status-success/40 bg-status-success/10 text-status-success',
  };
  const labels: Record<LibraryFileEntry['source'], string> = {
    starter: t('libraryFiles.sourceStarter'),
    bundle: t('libraryFiles.sourceBundle'),
    user: t('libraryFiles.sourceUser'),
  };
  return (
    <span
      className={`inline-block rounded-full border px-cell py-0.5 text-[10px] font-medium capitalize ${styles[source]}`}
    >
      {labels[source]}
    </span>
  );
};

function humanBytes(n: number): string {
  if (n >= 1 << 30) return `${(n / (1 << 30)).toFixed(1)} GB`;
  if (n >= 1 << 20) return `${(n / (1 << 20)).toFixed(1)} MB`;
  if (n >= 1 << 10) return `${(n / (1 << 10)).toFixed(1)} KB`;
  return `${n} B`;
}

export const LibraryWalksPage: FC = () => <LibraryFilesView kind="walks" />;
export const LibraryPcapsPage: FC = () => <LibraryFilesView kind="pcaps" />;
