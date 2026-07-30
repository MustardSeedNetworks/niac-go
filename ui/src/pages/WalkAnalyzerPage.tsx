import { type FC, type ReactNode, useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { analyzeWalk } from '../api/client';
import { fetchLibraryWalks, type LibraryFileEntry } from '../api/library-client';
import type { WalkAnalyzeResponse } from '../api/types';
import { WalkProfileCreator } from '../components/walk/WalkProfileCreator';
import { Card, CardContent, CardRow, CardValue } from '../ui/Card';
import { formatBitsPerSecond } from '../utils/format';

const STATUS_BADGE: Record<string, string> = {
  up: 'bg-status-success/20 text-status-success ring-status-success/40',
  down: 'bg-status-error/20 text-status-error ring-status-error/40',
  testing: 'bg-status-warning/20 text-status-warning ring-status-warning/40',
};

const PROTOCOL_BADGE: Record<string, string> = {
  lldp: 'bg-status-info/20 text-status-info ring-status-info/40',
  cdp: 'bg-brand-primary/20 text-brand-accent ring-brand-primary/40',
};

const FALLBACK_BADGE = 'bg-bg-muted/20 text-text-secondary ring-surface-border/40';

function renderStatusBadge(status: string | undefined): ReactNode {
  if (!status) {
    return <span className="text-text-muted">—</span>;
  }
  const cls = STATUS_BADGE[status.toLowerCase()] ?? FALLBACK_BADGE;
  return (
    <span className={`rounded px-cell py-0.5 text-[10px] font-medium ring-1 ${cls}`}>{status}</span>
  );
}

function renderProtocolBadge(protocol: string): ReactNode {
  const cls = PROTOCOL_BADGE[protocol.toLowerCase()] ?? FALLBACK_BADGE;
  return (
    <span className={`rounded px-cell py-0.5 text-[10px] font-medium ring-1 ${cls}`}>
      {protocol.toUpperCase()}
    </span>
  );
}

export const WalkAnalyzerPage: FC = () => {
  const { t } = useTranslation('pages');
  const [files, setFiles] = useState<LibraryFileEntry[]>([]);
  const [filesError, setFilesError] = useState<string | null>(null);
  const [filesLoading, setFilesLoading] = useState(true);

  const [selectedFile, setSelectedFile] = useState<string>('');
  const [customPath, setCustomPath] = useState<string>('');

  const [response, setResponse] = useState<WalkAnalyzeResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState<'idle' | 'analyzing'>('idle');

  // Hydrate the file dropdown from /api/v1/library/walks — the same
  // sandboxed walks directory the walk validator's picker uses, so
  // library-relative names like "cisco/c3900.walk" resolve unchanged
  // server-side.
  useEffect(() => {
    let cancelled = false;
    setFilesLoading(true);
    fetchLibraryWalks()
      .then((entries) => {
        if (cancelled) return;
        setFiles(entries);
        setFilesError(null);
        if (entries.length > 0 && !selectedFile) {
          setSelectedFile(entries[0].name);
        }
      })
      .catch((err: Error) => {
        if (cancelled) return;
        setFilesError(err.message);
      })
      .finally(() => {
        if (!cancelled) setFilesLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [selectedFile]);

  const targetPath = customPath.trim() || selectedFile;
  const result = response?.result;
  const interfaces = result?.interfaces ?? [];
  const neighbors = result?.neighbors ?? [];

  const run = useCallback(async () => {
    if (!targetPath) {
      setError('Pick a walk file or enter a path first.');
      return;
    }
    setError(null);
    setBusy('analyzing');
    try {
      const analyzed = await analyzeWalk(targetPath);
      setResponse(analyzed);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy('idle');
    }
  }, [targetPath]);

  return (
    <div className="stack-xl">
      <WalkProfileCreator />
      <Card className="border-surface-border bg-bg-surface/70">
        <CardContent className="stack-lg">
          <header>
            <h1 className="text-2xl font-semibold text-text-primary">
              {t('walkAnalyzer.pageTitle')}
            </h1>
            <p className="text-sm text-text-muted">
              Same engine as <code>niac analyze-walk</code>. Parses SNMPv2-MIB, IF-MIB/ifXTable,
              LLDP-MIB, and CISCO-CDP-MIB into device identity, interface inventory, and neighbor
              adjacencies.
            </p>
          </header>

          <div className="grid gap-comfortable md:grid-cols-2">
            <label className="block text-sm">
              <span className="text-text-secondary">From walks/ directory</span>
              <select
                value={selectedFile}
                onChange={(e) => setSelectedFile(e.target.value)}
                disabled={filesLoading || files.length === 0}
                title="Hydrated from /api/v1/library/walks (the sandboxed walks directory). Use the absolute-path field to analyze a walk outside this directory."
                data-testid="walk-analyzer-picker"
                className="mt-tight w-full rounded border border-surface-border bg-bg-base/60 px-3 py-row text-sm text-text-primary focus:border-status-info focus:outline-none disabled:opacity-50"
              >
                {filesLoading && <option>Loading…</option>}
                {!filesLoading && files.length === 0 && (
                  <option>{t('walkAnalyzer.noWalksFound')}</option>
                )}
                {files.map((f) => (
                  <option key={f.name} value={f.name}>
                    {f.name} ({Math.round(f.sizeBytes / 1024)} KB)
                  </option>
                ))}
              </select>
              {filesError && (
                <span className="mt-tight block text-xs text-status-error">{filesError}</span>
              )}
            </label>

            <label className="block text-sm">
              <span className="text-text-secondary">{t('walkAnalyzer.pastePathLabel')}</span>
              <input
                type="text"
                value={customPath}
                onChange={(e) => setCustomPath(e.target.value)}
                placeholder="/srv/niac/walks/cisco-c9300.walk"
                title="Absolute path to a walk file. Takes precedence over the dropdown selection. The path is bounded server-side; ../ traversal is rejected."
                data-testid="walk-analyzer-path-input"
                className="mt-tight w-full rounded border border-surface-border bg-bg-base/60 px-3 py-row font-mono text-xs text-text-primary placeholder:text-text-muted focus:border-status-info focus:outline-none"
              />
            </label>
          </div>

          <div className="flex flex-wrap items-center gap-default">
            <button
              type="button"
              onClick={() => void run()}
              disabled={busy !== 'idle' || !targetPath}
              title="Parses the walk file into device identity, interfaces, and neighbors. Read-only — never modifies the file."
              data-testid="walk-analyzer-analyze-button"
              className="rounded bg-status-info/20 px-3 py-compact-md text-sm font-medium text-status-info ring-1 ring-status-info/40 hover:bg-status-info/30 disabled:opacity-50"
            >
              {busy === 'analyzing' ? 'Analyzing…' : 'Analyze'}
            </button>
            {error && (
              <span className="text-sm text-status-error" role="alert">
                {error}
              </span>
            )}
          </div>
        </CardContent>
      </Card>

      {result && (
        <div className="stack-xl" data-testid="walk-analyzer-results">
          <Card className="border-surface-border bg-bg-surface/70">
            <CardContent className="stack">
              <h2 className="heading-3 text-text-primary">Device</h2>
              <CardRow label="sysName" value={result.device.sysname || '—'} mono />
              <CardRow label="sysDescr" value={result.device.sysdescr || '—'} />
              <CardRow label="sysObjectID" value={result.device.sysobjectid || '—'} mono />
              {result.device.syscontact && (
                <CardRow label="sysContact" value={result.device.syscontact} />
              )}
              {result.device.syslocation && (
                <CardRow label="sysLocation" value={result.device.syslocation} />
              )}
            </CardContent>
          </Card>

          <Card className="border-surface-border bg-bg-surface/70">
            <CardContent className="stack">
              <h2 className="heading-3 text-text-primary">Statistics</h2>
              <div className="grid grid-cols-2 gap-comfortable sm:grid-cols-4">
                <CardValue label="Interfaces" value={result.statistics.totalInterfaces} />
                <CardValue label="Physical" value={result.statistics.physicalInterfaces} />
                <CardValue label="Logical" value={result.statistics.logicalInterfaces} />
                <CardValue label="Neighbors" value={result.statistics.totalNeighbors} />
              </div>
            </CardContent>
          </Card>

          <Card
            className="border-surface-border bg-bg-surface/70"
            data-testid="walk-analyzer-interfaces-table"
          >
            <CardContent className="stack-lg">
              <h2 className="heading-3 text-text-primary">Interfaces</h2>
              {interfaces.length === 0 ? (
                <p className="text-sm text-text-muted">{t('walkAnalyzer.noInterfacesFound')}</p>
              ) : (
                <table className="w-full text-sm">
                  <thead className="bg-bg-base/40 text-left text-xs uppercase tracking-wider text-text-muted">
                    <tr>
                      <th className="px-3 py-row w-16">Index</th>
                      <th className="px-3 py-row">Name</th>
                      <th className="px-3 py-row">Type</th>
                      <th className="px-3 py-row">Speed</th>
                      <th className="px-3 py-row">Admin</th>
                      <th className="px-3 py-row">Oper</th>
                      <th className="px-3 py-row">MAC</th>
                      <th className="px-3 py-row">Description</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-knob/5">
                    {interfaces.map((iface, idx) => (
                      <tr
                        key={`${iface.index}-${idx}`}
                        className="text-text-primary hover:bg-bg-base/40"
                      >
                        <td className="px-3 py-row font-mono text-xs text-text-muted">
                          {iface.index}
                        </td>
                        <td className="px-3 py-row">{iface.name || '—'}</td>
                        <td className="px-3 py-row text-text-muted">{iface.type || '—'}</td>
                        <td className="px-3 py-row font-mono text-xs">
                          {formatBitsPerSecond(iface.speed)}
                        </td>
                        <td className="px-3 py-row">{renderStatusBadge(iface.adminStatus)}</td>
                        <td className="px-3 py-row">{renderStatusBadge(iface.operStatus)}</td>
                        <td className="px-3 py-row font-mono text-xs text-text-muted">
                          {iface.macAddress || '—'}
                        </td>
                        <td className="px-3 py-row text-text-muted">{iface.description || '—'}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </CardContent>
          </Card>

          <Card
            className="border-surface-border bg-bg-surface/70"
            data-testid="walk-analyzer-neighbors-table"
          >
            <CardContent className="stack-lg">
              <h2 className="heading-3 text-text-primary">Neighbors</h2>
              {neighbors.length === 0 ? (
                <p className="text-sm text-text-muted">{t('walkAnalyzer.noNeighborsFound')}</p>
              ) : (
                <table className="w-full text-sm">
                  <thead className="bg-bg-base/40 text-left text-xs uppercase tracking-wider text-text-muted">
                    <tr>
                      <th className="px-3 py-row">Local Interface</th>
                      <th className="px-3 py-row w-24">Protocol</th>
                      <th className="px-3 py-row">Remote Device</th>
                      <th className="px-3 py-row">Remote Interface</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-knob/5">
                    {neighbors.map((neighbor, idx) => (
                      <tr
                        key={`${neighbor.localInterface}-${neighbor.remoteDevice}-${idx}`}
                        className="text-text-primary hover:bg-bg-base/40"
                      >
                        <td className="px-3 py-row font-mono text-xs">
                          {neighbor.localInterface || '—'}
                        </td>
                        <td className="px-3 py-row">{renderProtocolBadge(neighbor.protocol)}</td>
                        <td className="px-3 py-row">{neighbor.remoteDevice || '—'}</td>
                        <td className="px-3 py-row font-mono text-xs text-text-muted">
                          {neighbor.remoteInterface || '—'}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </CardContent>
          </Card>
        </div>
      )}
    </div>
  );
};
