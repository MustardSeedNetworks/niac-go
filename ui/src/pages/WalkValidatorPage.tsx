import { useCallback, useEffect, useMemo, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';
import { fixWalk, validateAllWalks, validateWalk } from '../api/client';
import { fetchLibraryWalks, type LibraryFileEntry } from '../api/library-client';
import type {
  WalkBatchValidationResponse,
  WalkValidationIssue,
  WalkValidationResponse,
  WalkValidationResult,
} from '../api/types';
import { GlossaryPopover } from '../components/GlossaryPopover';
import { useToast } from '../hooks/useToast';
import { ConfirmDialog } from '../shared/ConfirmDialog';
import { DataTable, type DataTableColumn } from '../shared/DataTable';
import { StatusPill } from '../shared/StatusPill';
import { ActionButton } from '../ui/ActionButton';
import { Card, CardContent } from '../ui/Card';
import { ContainedTable } from '../ui/ContainedTable';
import { useDialogLabels } from '../ui/dialogLabels';
import { Tooltip } from '../ui/Tooltip';
import { useTableLabels } from '../ui/tableLabels';

type Severity = 'error' | 'warning' | 'info';

const SEVERITY_TONE: Record<Severity, 'error' | 'warning' | 'info'> = {
  error: 'error',
  warning: 'warning',
  info: 'info',
};

const SEVERITY_ORDER: Severity[] = ['error', 'warning', 'info'];

const severityCounts = (issues: WalkValidationIssue[]): Record<Severity, number> => {
  const out: Record<Severity, number> = { error: 0, warning: 0, info: 0 };
  for (const issue of issues) {
    if (issue.severity in out) {
      out[issue.severity as Severity] += 1;
    }
  }
  return out;
};

export function WalkValidatorPage() {
  const dialogLabels = useDialogLabels();
  const { t } = useTranslation('pages');
  const { t: tCommon } = useTranslation('common');
  const [files, setFiles] = useState<LibraryFileEntry[]>([]);
  const [filesError, setFilesError] = useState<string | null>(null);
  const [filesLoading, setFilesLoading] = useState(true);

  const [selectedFile, setSelectedFile] = useState<string>('');
  const [customPath, setCustomPath] = useState<string>('');
  const [oidFilter, setOidFilter] = useState<string>('');

  const [response, setResponse] = useState<WalkValidationResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState<'idle' | 'validating' | 'fixing'>('idle');
  const [showAutoFixConfirm, setShowAutoFixConfirm] = useState(false);
  const [batchResponse, setBatchResponse] = useState<WalkBatchValidationResponse | null>(null);
  const [batchBusy, setBatchBusy] = useState(false);
  const { showError } = useToast();

  // Hydrate the file dropdown from /api/v1/library/walks. The daemon's
  // walk validator (validateWalkFilePath) falls back to the library
  // walks dir when the file isn't in the legacy config-dir allow-list,
  // so we can send library-relative names like "cisco/c3900.walk"
  // unchanged.
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

  const issues = response?.result?.issues ?? [];
  const counts = useMemo(() => severityCounts(issues), [issues]);

  // OID filter applies to the FULL issue list before the 200-row display
  // cap below, so a match past the cap is still reachable by filtering.
  const filteredIssues = useMemo(() => {
    const needle = oidFilter.trim().toLowerCase();
    if (!needle) return issues;
    return issues.filter((issue) => (issue.oid ?? '').toLowerCase().includes(needle));
  }, [issues, oidFilter]);

  const ISSUES_DISPLAY_CAP = 200;
  const visibleIssues = filteredIssues.slice(0, ISSUES_DISPLAY_CAP);

  const run = useCallback(
    async (action: 'validating' | 'fixing') => {
      if (!targetPath) {
        setError(t('walkValidator.pickFirst'));
        return;
      }
      setError(null);
      setBusy(action);
      try {
        const result =
          action === 'fixing' ? await fixWalk(targetPath) : await validateWalk(targetPath);
        setResponse(result);
        setOidFilter('');
      } catch (err) {
        setError((err as Error).message);
      } finally {
        setBusy('idle');
      }
    },
    [targetPath, t],
  );

  const runBatch = useCallback(async () => {
    setBatchBusy(true);
    try {
      const result = await validateAllWalks();
      setBatchResponse(result);
    } catch (err) {
      showError(err);
    } finally {
      setBatchBusy(false);
    }
  }, [showError]);

  const batchResults = useMemo(
    () => (batchResponse ? Object.values(batchResponse.results) : []),
    [batchResponse],
  );

  const labels = useTableLabels(t('walkValidator.issuesTable.message').toLowerCase());
  const issueColumns: DataTableColumn<WalkValidationIssue>[] = [
    {
      key: 'line',
      header: t('walkValidator.issuesTable.line'),
      cell: (issue) => <span className="font-mono text-xs text-text-muted">{issue.line}</span>,
    },
    {
      key: 'severity',
      header: t('walkValidator.issuesTable.severity'),
      cell: (issue) => (
        <StatusPill
          tone={SEVERITY_TONE[issue.severity as Severity] ?? 'unknown'}
          label={issue.severity}
        />
      ),
    },
    {
      key: 'message',
      header: t('walkValidator.issuesTable.message'),
      cell: (issue) => issue.message,
    },
    {
      key: 'oid',
      header: t('walkValidator.issuesTable.oid'),
      cell: (issue) => <span className="font-mono text-xs text-text-muted">{issue.oid ?? ''}</span>,
    },
    {
      key: 'original',
      header: t('walkValidator.issuesTable.original'),
      cell: (issue) => (
        <span className="truncate font-mono text-xs text-text-muted">{issue.original}</span>
      ),
    },
  ];

  const batchColumns: DataTableColumn<WalkValidationResult>[] = [
    {
      key: 'filename',
      header: t('walkValidator.batchTable.filename'),
      cell: (result) => <span className="font-mono text-xs">{result.filename}</span>,
    },
    {
      key: 'status',
      header: t('walkValidator.batchTable.status'),
      cell: (result) => (
        <StatusPill
          tone={result.valid ? 'success' : 'error'}
          label={
            result.valid
              ? t('walkValidator.batchStatusValid')
              : t('walkValidator.batchStatusInvalid')
          }
        />
      ),
    },
    {
      key: 'issues',
      header: t('walkValidator.batchTable.issues'),
      cell: (result) => <span className="text-text-muted">{result.issues.length}</span>,
    },
  ];

  return (
    <div className="stack-xl">
      <Card className="border-surface-border bg-bg-surface/70">
        <CardContent className="stack-lg">
          <header>
            <h2 className="flex items-center gap-compact text-2xl font-semibold text-text-primary">
              {t('walkValidator.pageTitle')}
              <GlossaryPopover term="snmpWalk" />
            </h2>
            <p className="text-sm text-text-muted">
              <Trans
                i18nKey="walkValidator.engineNote"
                ns="pages"
                components={{
                  code: <code />,
                  oid: <GlossaryPopover term="oid" />,
                }}
              />
            </p>
          </header>

          <div className="grid gap-comfortable md:grid-cols-2">
            <label className="block text-sm">
              <span className="text-text-secondary">{t('walkValidator.fromWalksDir')}</span>
              <Tooltip text={t('walkValidator.fromWalksDirHelp')} className="w-full">
                <select
                  data-testid="walk-validator-picker"
                  value={selectedFile}
                  onChange={(e) => setSelectedFile(e.target.value)}
                  disabled={filesLoading || files.length === 0}
                  className="mt-tight w-full rounded border border-surface-border bg-bg-base/60 px-3 py-row text-sm text-text-primary focus:border-status-info focus:outline-none disabled:opacity-50"
                >
                  {filesLoading && <option>{tCommon('status.loading')}</option>}
                  {!filesLoading && files.length === 0 && (
                    <option>{t('walkValidator.noWalksFound')}</option>
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
              <span className="text-text-secondary">{t('walkValidator.pastePathLabel')}</span>
              <Tooltip text={t('walkValidator.pastePathHelp')} className="w-full">
                <input
                  type="text"
                  value={customPath}
                  onChange={(e) => setCustomPath(e.target.value)}
                  placeholder="/srv/niac/walks/cisco-c9300.walk"
                  className="mt-tight w-full rounded border border-surface-border bg-bg-base/60 px-3 py-row font-mono text-xs text-text-primary placeholder:text-text-muted focus:border-status-info focus:outline-none"
                />
              </Tooltip>
            </label>
          </div>

          <div className="flex flex-wrap items-center gap-default">
            <ActionButton
              variant="primary"
              action="edit"
              type="button"
              onClick={() => void run('validating')}
              disabled={busy !== 'idle' || !targetPath}
              title={t('walkValidator.validateHelp')}
            >
              {busy === 'validating'
                ? t('walkValidator.validatingButton')
                : t('walkValidator.validateButton')}
            </ActionButton>
            <ActionButton
              action="edit"
              variant="secondary"
              type="button"
              onClick={() => setShowAutoFixConfirm(true)}
              disabled={busy !== 'idle' || !targetPath}
              title={t('walkValidator.autoFixHelp')}
            >
              {busy === 'fixing'
                ? t('walkValidator.fixingButton')
                : t('walkValidator.autoFixButton')}
            </ActionButton>
            <ActionButton
              action="edit"
              variant="secondary"
              type="button"
              onClick={() => void runBatch()}
              disabled={batchBusy}
              title={t('walkValidator.validateAllHelp')}
            >
              {batchBusy
                ? t('walkValidator.validatingAllButton')
                : t('walkValidator.validateAllButton')}
            </ActionButton>
            {error && (
              <span className="text-sm text-status-error" role="alert">
                {error}
              </span>
            )}
          </div>
        </CardContent>
      </Card>

      {response?.result && (
        <Card className="border-surface-border bg-bg-surface/70">
          <CardContent className="stack-lg">
            <header className="flex flex-wrap items-baseline gap-comfortable">
              <h2 className="heading-3 text-text-primary">
                {response.message ?? t('walkValidator.resultHeading')}
              </h2>
              <span className="text-sm text-text-muted">
                {t('walkValidator.lineSummary', {
                  total: response.result.totalLines,
                  valid: response.result.validLines,
                })}
              </span>
              <StatusPill
                tone={response.result.valid ? 'success' : 'error'}
                label={
                  response.result.valid
                    ? t('walkValidator.resultValid')
                    : t('walkValidator.resultInvalid')
                }
              />
              {SEVERITY_ORDER.map((sev) => (
                <StatusPill key={sev} tone={SEVERITY_TONE[sev]} label={`${sev}: ${counts[sev]}`} />
              ))}
              {typeof response.result.fixedCount === 'number' && (
                <StatusPill
                  tone="success"
                  label={t('walkValidator.fixedCount', { fixed: response.result.fixedCount })}
                />
              )}
            </header>

            {issues.length > 0 && (
              <label className="block text-sm">
                <span className="text-text-secondary">{t('walkValidator.oidFilterLabel')}</span>
                <Tooltip text={t('walkValidator.oidFilterHelp')} className="w-full">
                  <input
                    type="text"
                    value={oidFilter}
                    onChange={(e) => setOidFilter(e.target.value)}
                    placeholder={t('walkValidator.oidFilterPlaceholder')}
                    className="mt-tight w-full max-w-sm rounded border border-surface-border bg-bg-base/60 px-3 py-row font-mono text-xs text-text-primary placeholder:text-text-muted focus:border-status-info focus:outline-none"
                  />
                </Tooltip>
              </label>
            )}

            {issues.length === 0 ? (
              <p className="text-sm text-text-muted">{t('walkValidator.noIssues')}</p>
            ) : filteredIssues.length === 0 ? (
              <p className="text-sm text-text-muted">{t('walkValidator.noMatchingIssues')}</p>
            ) : (
              <ContainedTable label={t('walkValidator.resultHeading')}>
                <DataTable
                  rows={visibleIssues}
                  columns={issueColumns}
                  getRowKey={(issue) => `${issue.line}-${issue.oid ?? ''}-${issue.message}`}
                  caption={t('walkValidator.resultHeading')}
                  labels={labels}
                  empty={null}
                />
              </ContainedTable>
            )}
            {filteredIssues.length > ISSUES_DISPLAY_CAP && (
              <p className="text-xs text-text-muted">
                {t('walkValidator.showingFiltered', {
                  shown: ISSUES_DISPLAY_CAP,
                  filtered: filteredIssues.length,
                  total: issues.length,
                })}
              </p>
            )}
          </CardContent>
        </Card>
      )}

      {batchResponse && (
        <Card className="border-surface-border bg-bg-surface/70">
          <CardContent className="stack-lg">
            <header className="flex flex-wrap items-baseline gap-comfortable">
              <h2 className="heading-3 text-text-primary">{t('walkValidator.batchHeading')}</h2>
              <span className="text-sm text-text-muted">{batchResponse.message}</span>
              <StatusPill
                tone={batchResponse.success ? 'success' : 'error'}
                label={
                  batchResponse.success
                    ? t('walkValidator.batchStatusValid')
                    : t('walkValidator.batchStatusInvalid')
                }
              />
            </header>

            {batchResults.length === 0 ? (
              <p className="text-sm text-text-muted">{t('walkValidator.batchEmpty')}</p>
            ) : (
              <ContainedTable label={t('walkValidator.batchHeading')}>
                <DataTable
                  rows={batchResults}
                  columns={batchColumns}
                  getRowKey={(result) => result.filename}
                  caption={t('walkValidator.batchHeading')}
                  labels={labels}
                  empty={null}
                />
              </ContainedTable>
            )}
          </CardContent>
        </Card>
      )}

      <ConfirmDialog
        {...dialogLabels}
        open={showAutoFixConfirm}
        onConfirm={() => {
          setShowAutoFixConfirm(false);
          void run('fixing');
        }}
        onCancel={() => setShowAutoFixConfirm(false)}
        title={t('walkValidator.autoFixConfirmTitle')}
        message={t('walkValidator.autoFixConfirmMessage', { file: targetPath })}
        confirmLabel={tCommon('buttons.autoFix')}
        cancelLabel={tCommon('buttons.cancel')}
        variant="primary"
      />
    </div>
  );
}
