import { Database, FileBox, RefreshCw, Search } from 'lucide-react';
import { type FC, useMemo, useState } from 'react';
import { fetchLibraryPcaps, fetchLibraryWalks, type LibraryFileEntry } from '../api/client';
import { useApiResource } from '../hooks/useApiResource';
import { Button } from '../ui/Button';
import { Card, CardContent } from '../ui/Card';
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
 */

interface Props {
  kind: 'walks' | 'pcaps';
}

function LibraryFilesView({ kind }: Props) {
  const fetcher = kind === 'walks' ? fetchLibraryWalks : fetchLibraryPcaps;
  const { data, loading, refetch, error } = useApiResource(fetcher, [], { intervalMs: 30000 });
  const [search, setSearch] = useState('');

  const entries = data ?? [];
  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase();
    if (!q) return entries;
    return entries.filter((e) => e.name.toLowerCase().includes(q));
  }, [entries, search]);

  const totalBytes = useMemo(() => filtered.reduce((acc, e) => acc + e.sizeBytes, 0), [filtered]);

  const KindIcon = kind === 'walks' ? Database : FileBox;
  const title = kind === 'walks' ? 'SNMP Walks' : 'PCAP Captures';
  const emptyHint =
    kind === 'walks'
      ? 'Drop walk files into ~/.niac/library/walks/, or run `niac content install` to fetch the published bundle.'
      : 'Drop pcap files into ~/.niac/library/pcaps/, or run `niac content install` to fetch the published bundle.';

  return (
    <div className="space-y-6">
      <Card className="border-white/5 bg-bg-surface/70">
        <CardContent>
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
            <div className="flex items-center gap-3">
              <div className="p-2 rounded-lg bg-status-info/20">
                <KindIcon className="w-6 h-6 text-status-info" />
              </div>
              <div>
                <H2>{title}</H2>
                <SmallText className="text-text-muted">
                  {entries.length} {entries.length === 1 ? 'file' : 'files'} ·{' '}
                  {humanBytes(totalBytes)}
                </SmallText>
              </div>
            </div>

            <div className="flex items-center gap-2">
              <div className="relative">
                <Search className="absolute left-2 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-text-muted" />
                <input
                  type="search"
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  placeholder="Search by name…"
                  aria-label="Filter library entries by name"
                  className="w-64 rounded-md border border-white/10 bg-bg-base/40 pl-7 pr-3 py-1.5 text-xs text-text-primary placeholder:text-text-muted focus:border-status-info/40 focus:outline-none focus:ring-1 focus:ring-cyan-500/30"
                />
              </div>
              <Button
                variant="ghost"
                size="sm"
                onClick={refetch}
                leftIcon={<RefreshCw className="w-4 h-4" />}
                disabled={loading}
                title="Re-fetch the file listing from the daemon's library directory"
              >
                Refresh
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card className="border-white/5 bg-bg-surface/70">
        <CardContent>
          {error ? (
            <SmallText className="text-status-error">
              Failed to load library: {error.message}
            </SmallText>
          ) : entries.length === 0 ? (
            <div className="py-10 text-center">
              <SmallText className="text-text-muted">No {kind} installed yet.</SmallText>
              <p className="mt-2 text-xs text-text-muted">{emptyHint}</p>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead className="text-xs text-text-muted uppercase tracking-wide">
                  <tr className="border-b border-white/10">
                    <th className="text-left py-2 pr-4">Name</th>
                    <th className="text-right py-2 pr-4">Size</th>
                    <th className="text-left py-2 pr-4">Source</th>
                    <th className="text-left py-2">Modified</th>
                  </tr>
                </thead>
                <tbody className="text-text-primary">
                  {filtered.map((entry) => (
                    <tr key={entry.name} className="border-b border-white/5 last:border-0">
                      <td className="py-2 pr-4 font-mono text-xs">{entry.name}</td>
                      <td className="py-2 pr-4 text-right tabular-nums">
                        {humanBytes(entry.sizeBytes)}
                      </td>
                      <td className="py-2 pr-4">
                        <SourceBadge source={entry.source} />
                      </td>
                      <td className="py-2 text-xs text-text-muted">
                        {new Date(entry.modifiedAt).toLocaleString()}
                      </td>
                    </tr>
                  ))}
                  {search.trim() !== '' && filtered.length === 0 && (
                    <tr>
                      <td colSpan={4} className="py-6 text-center text-xs text-text-muted">
                        No entries match "{search}"
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

const SourceBadge: FC<{ source: LibraryFileEntry['source'] }> = ({ source }) => {
  const styles: Record<LibraryFileEntry['source'], string> = {
    starter: 'border-brand-500/40 bg-brand-500/10 text-brand-200',
    bundle: 'border-status-info/40 bg-status-info/10 text-status-info',
    user: 'border-status-success/40 bg-status-success/10 text-status-success',
  };
  return (
    <span
      className={`inline-block rounded-full border px-2 py-0.5 text-[10px] font-medium capitalize ${styles[source]}`}
    >
      {source}
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
