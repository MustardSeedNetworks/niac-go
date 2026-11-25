import { type ReactNode, useCallback, useEffect, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';
import { analyzeWalk } from '../api/client';
import { fetchLibraryWalks, type LibraryFileEntry } from '../api/library-client';
import type { WalkAnalyzeResponse } from '../api/types';
import { TableSection } from '../components/TableSection';
import { WalkProfileCreator } from '../components/walk/WalkProfileCreator';
import { CategoryChip } from '../shared/CategoryChip';
import { DataTable, type DataTableColumn } from '../shared/DataTable';
import { EmptyState } from '../shared/EmptyState';
import { StatusPill } from '../shared/StatusPill';
import { ActionButton } from '../ui/ActionButton';
import { Card, CardContent, CardRow, CardValue } from '../ui/Card';
import { Tooltip } from '../ui/Tooltip';
import { useTableLabels } from '../ui/tableLabels';
import { formatBitsPerSecond } from '../utils/format';

const STATUS_TONE: Record<string, 'success' | 'error' | 'warning'> = {
  up: 'success',
  down: 'error',
  testing: 'warning',
};

function renderStatusBadge(status: string | undefined): ReactNode {
  if (!status) {
    return <span className="text-text-muted">—</span>;
  }
  return <StatusPill tone={STATUS_TONE[status.toLowerCase()] ?? 'unknown'} label={status} />;
}

function renderProtocolBadge(protocol: string): ReactNode {
  return <CategoryChip label={protocol.toUpperCase()} />;
}

export function WalkAnalyzerPage() {
  const { t } = useTranslation('pages');
  const { t: tCommon } = useTranslation('common');
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
        setSelectedFile((current) => current || entries[0]?.name || '');
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
  }, []);

  const targetPath = customPath.trim() || selectedFile;
  const result = response?.result;
  const interfaces = result?.interfaces ?? [];
  const neighbors = result?.neighbors ?? [];

  const labels = useTableLabels(t('walkAnalyzer.interfacesHeading').toLowerCase());
  const interfaceColumns: DataTableColumn<(typeof interfaces)[number]>[] = [
    {
      key: 'index',
      header: t('walkAnalyzer.interfacesTable.index'),
      cell: (iface) => <span className="font-mono text-xs text-text-muted">{iface.index}</span>,
    },
    {
      key: 'name',
      header: t('walkAnalyzer.interfacesTable.name'),
      cell: (iface) => iface.name || '—',
    },
    {
      key: 'type',
      header: t('walkAnalyzer.interfacesTable.type'),
      cell: (iface) => <span className="text-text-muted">{iface.type || '—'}</span>,
    },
    {
      key: 'speed',
      header: t('walkAnalyzer.interfacesTable.speed'),
      cell: (iface) => (
        <span className="font-mono text-xs">{formatBitsPerSecond(iface.speed)}</span>
      ),
    },
    {
      key: 'admin',
      header: t('walkAnalyzer.interfacesTable.admin'),
      cell: (iface) => renderStatusBadge(iface.adminStatus),
    },
    {
      key: 'oper',
      header: t('walkAnalyzer.interfacesTable.oper'),
      cell: (iface) => renderStatusBadge(iface.operStatus),
    },
    {
      key: 'mac',
      header: 'MAC',
      cell: (iface) => (
        <span className="font-mono text-xs text-text-muted">{iface.macAddress || '—'}</span>
      ),
    },
    {
      key: 'description',
      header: t('walkAnalyzer.interfacesTable.description'),
      cell: (iface) => <span className="text-text-muted">{iface.description || '—'}</span>,
    },
  ];

  const neighborColumns: DataTableColumn<(typeof neighbors)[number]>[] = [
    {
      key: 'localInterface',
      header: t('walkAnalyzer.localInterface'),
      cell: (neighbor) => (
        <span className="font-mono text-xs">{neighbor.localInterface || '—'}</span>
      ),
    },
    {
      key: 'protocol',
      header: t('walkAnalyzer.protocol'),
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
      cell: (neighbor) => (
        <span className="font-mono text-xs text-text-muted">{neighbor.remoteInterface || '—'}</span>
      ),
    },
  ];

  const run = useCallback(async () => {
    if (!targetPath) {
      setError(t('walkAnalyzer.pickFirst'));
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
  }, [targetPath, t]);

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
              <Tooltip text={t('walkAnalyzer.fromWalksDirHelp')} className="w-full">
                <select
                  value={selectedFile}
                  onChange={(e) => setSelectedFile(e.target.value)}
                  disabled={filesLoading || files.length === 0}
                  data-testid="walk-analyzer-picker"
                  className="mt-tight w-full rounded border border-surface-border bg-bg-base/60 px-3 py-row text-sm text-text-primary focus:border-status-info focus:outline-none disabled:opacity-50"
                >
                  {filesLoading && <option>{tCommon('status.loading')}</option>}
                  {!filesLoading && files.length === 0 && (
                    <option>{t('walkAnalyzer.noWalksFound')}</option>
                  )}
                  {files.map((f) => (
                    <option key={f.name} value={f.name}>
                      {f.name} ({Math.round(f.sizeBytes / 1024)} KB)
                    </option>
                  ))}
                </select>
              </Tooltip>
              {filesError && (
                <span className="mt-tight block text-xs text-status-error">{filesError}</span>
              )}
            </label>

            <label className="block text-sm">
              <span className="text-text-secondary">{t('walkAnalyzer.pastePathLabel')}</span>
              <Tooltip text={t('walkAnalyzer.pastePathHelp')} className="w-full">
                <input
                  type="text"
                  value={customPath}
                  onChange={(e) => setCustomPath(e.target.value)}
                  placeholder="/srv/niac/walks/cisco-c9300.walk"
                  data-testid="walk-analyzer-path-input"
                  className="mt-tight w-full rounded border border-surface-border bg-bg-base/60 px-3 py-row font-mono text-xs text-text-primary placeholder:text-text-muted focus:border-status-info focus:outline-none"
                />
              </Tooltip>
            </label>
          </div>

          <div className="flex flex-wrap items-center gap-default">
            <ActionButton
              action="edit"
              variant="secondary"
              type="button"
              onClick={() => void run()}
              disabled={busy !== 'idle' || !targetPath}
              title={t('walkAnalyzer.analyzeHelp')}
              data-testid="walk-analyzer-analyze-button"
            >
              {busy === 'analyzing'
                ? t('walkAnalyzer.analyzingButton')
                : t('walkAnalyzer.analyzeButton')}
            </ActionButton>
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
              <h2 className="heading-3 text-text-primary">{t('walkAnalyzer.deviceHeading')}</h2>
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
              <h2 className="heading-3 text-text-primary">{t('walkAnalyzer.statisticsHeading')}</h2>
              <div className="grid grid-cols-2 gap-comfortable sm:grid-cols-4">
                <CardValue
                  label={t('walkAnalyzer.stats.interfaces')}
                  value={result.statistics.totalInterfaces}
                />
                <CardValue
                  label={t('walkAnalyzer.stats.physical')}
                  value={result.statistics.physicalInterfaces}
                />
                <CardValue
                  label={t('walkAnalyzer.stats.logical')}
                  value={result.statistics.logicalInterfaces}
                />
                <CardValue
                  label={t('walkAnalyzer.stats.neighbors')}
                  value={result.statistics.totalNeighbors}
                />
              </div>
            </CardContent>
          </Card>

          <TableSection
            title={t('walkAnalyzer.interfacesHeading')}
            data-testid="walk-analyzer-interfaces-table"
          >
            {interfaces.length === 0 ? (
              <EmptyState title={t('walkAnalyzer.noInterfacesFound')} />
            ) : (
              <DataTable
                rows={interfaces}
                columns={interfaceColumns}
                getRowKey={(iface) => `${iface.index}-${iface.name}`}
                caption={t('walkAnalyzer.interfacesHeading')}
                labels={labels}
                empty={null}
              />
            )}
          </TableSection>

          <TableSection
            title={t('walkAnalyzer.neighborsHeading')}
            data-testid="walk-analyzer-neighbors-table"
          >
            {neighbors.length === 0 ? (
              <EmptyState title={t('walkAnalyzer.noNeighborsFound')} />
            ) : (
              <DataTable
                rows={neighbors}
                columns={neighborColumns}
                getRowKey={(neighbor) =>
                  `${neighbor.localInterface}-${neighbor.remoteDevice}-${neighbor.remoteInterface}`
                }
                caption={t('walkAnalyzer.neighborsHeading')}
                labels={labels}
                empty={null}
              />
            )}
          </TableSection>
        </div>
      )}
    </div>
  );
}
