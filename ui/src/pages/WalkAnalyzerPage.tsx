import { type FC, type ReactNode, useCallback, useEffect, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';
import { analyzeWalk } from '../api/client';
import { fetchLibraryWalks, type LibraryFileEntry } from '../api/library-client';
import type { WalkAnalyzeResponse } from '../api/types';
import { WalkProfileCreator } from '../components/walk/WalkProfileCreator';
import { Button } from '../ui/Button';
import { Card, CardContent, CardRow, CardValue } from '../ui/Card';
import { DataTable, type DataTableColumn } from '../ui/DataTable';
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
        const firstEntry = entries[0];
        if (firstEntry && !selectedFile) {
          setSelectedFile(firstEntry.name);
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

  const interfaceColumns: DataTableColumn<(typeof interfaces)[number]>[] = [
    {
      key: 'index',
      header: 'Index',
      headerClassName: 'w-16',
      cellClassName: 'font-mono text-xs text-text-muted',
      cell: (iface) => iface.index,
    },
    { key: 'name', header: 'Name', cell: (iface) => iface.name || '—' },
    {
      key: 'type',
      header: 'Type',
      cellClassName: 'text-text-muted',
      cell: (iface) => iface.type || '—',
    },
    {
      key: 'speed',
      header: 'Speed',
      cellClassName: 'font-mono text-xs',
      cell: (iface) => formatBitsPerSecond(iface.speed),
    },
    { key: 'admin', header: 'Admin', cell: (iface) => renderStatusBadge(iface.adminStatus) },
    { key: 'oper', header: 'Oper', cell: (iface) => renderStatusBadge(iface.operStatus) },
    {
      key: 'mac',
      header: 'MAC',
      cellClassName: 'font-mono text-xs text-text-muted',
      cell: (iface) => iface.macAddress || '—',
    },
    {
      key: 'description',
      header: 'Description',
      cellClassName: 'text-text-muted',
      cell: (iface) => iface.description || '—',
    },
  ];

  const neighborColumns: DataTableColumn<(typeof neighbors)[number]>[] = [
    {
      key: 'localInterface',
      header: t('walkAnalyzer.localInterface'),
      cellClassName: 'font-mono text-xs',
      cell: (neighbor) => neighbor.localInterface || '—',
    },
    {
      key: 'protocol',
      header: t('walkAnalyzer.protocol'),
      headerClassName: 'w-24',
      cell: (neighbor) => renderProtocolBadge(neighbor.protocol),
    },
    {
      key: 'remoteDevice',
      header: t('walkAnalyzer.remoteDevice'),
      cell: (neighbor) => neighbor.remoteDevice || '—',
    },
    {
      key: 'remoteInterface',
      header: t('walkAnalyzer.remoteInterface'),
      cellClassName: 'font-mono text-xs text-text-muted',
      cell: (neighbor) => neighbor.remoteInterface || '—',
    },
  ];

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
            <h2 className="text-2xl font-semibold text-text-primary">
              {t('walkAnalyzer.pageTitle')}
            </h2>
            <p className="text-sm text-text-muted">
              <Trans i18nKey="walkAnalyzer.engineNote" ns="pages" components={{ code: <code /> }} />
            </p>
          </header>

          <div className="grid gap-comfortable md:grid-cols-2">
            <label className="block text-sm">
              <span className="text-text-secondary">{t('walkAnalyzer.fromWalksDir')}</span>
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
            <Button
              type="button"
              variant="outline"
              tone="blue"
              onClick={() => void run()}
              action="edit"
              disabled={busy !== 'idle' || !targetPath}
              title="Parses the walk file into device identity, interfaces, and neighbors. Read-only — never modifies the file."
              data-testid="walk-analyzer-analyze-button"
            >
              {busy === 'analyzing' ? 'Analyzing…' : 'Analyze'}
            </Button>
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
                <DataTable
                  rows={interfaces}
                  columns={interfaceColumns}
                  getRowKey={(iface) => `${iface.index}-${iface.name}`}
                  emptyMessage={null}
                  rowClassName={() => 'text-text-primary hover:bg-bg-base/40'}
                />
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
                <DataTable
                  rows={neighbors}
                  columns={neighborColumns}
                  getRowKey={(neighbor) =>
                    `${neighbor.localInterface}-${neighbor.remoteDevice}-${neighbor.remoteInterface}`
                  }
                  emptyMessage={null}
                  rowClassName={() => 'text-text-primary hover:bg-bg-base/40'}
                />
              )}
            </CardContent>
          </Card>
        </div>
      )}
    </div>
  );
};
