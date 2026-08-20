import { type FC, useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { fixWalk, validateAllWalks, validateWalk } from '../api/client';
import { fetchLibraryWalks, type LibraryFileEntry } from '../api/library-client';
import type {
  WalkBatchValidationResponse,
  WalkValidationIssue,
  WalkValidationResponse,
} from '../api/types';
import { useErrorToast } from '../hooks/useErrorToast';
import { Card, CardContent } from '../ui/Card';
import { ConfirmModal } from '../ui/ConfirmModal';
import { InfoPopover } from '../ui/InfoPopover';

type Severity = 'error' | 'warning' | 'info';

const SEVERITY_BADGE: Record<Severity, string> = {
  error: 'bg-status-error/20 text-status-error ring-status-error/40',
  warning: 'bg-status-warning/20 text-status-warning ring-status-warning/40',
  info: 'bg-status-info/20 text-status-info ring-status-info/40',
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

export const WalkValidatorPage: FC = () => {
  const { t } = useTranslation('pages');
  const { t: tCommon } = useTranslation('common');
  const { t: tHelp } = useTranslation('help');
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
  const showError = useErrorToast();

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
        setError('Pick a walk file or enter a path first.');
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
    [targetPath],
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

  return (
    <div className="stack-xl">
      <Card className="border-surface-border bg-bg-surface/70">
        <CardContent className="stack-lg">
          <header>
            <h1 className="flex items-center gap-compact text-2xl font-semibold text-text-primary">
              {t('walkValidator.pageTitle')}
              <InfoPopover
                label={tCommon('jargon.ariaLabel', { term: 'SNMP walk' })}
                title="SNMP walk"
              >
                {tHelp('jargon.snmpWalk')}
              </InfoPopover>
            </h1>
            <p className="text-sm text-text-muted">
              Same engine as <code>niac analyze-walk</code>. Validate detects malformed lines,
              missing OIDs
              <InfoPopover label={tCommon('jargon.ariaLabel', { term: 'OID' })} title="OID">
                {tHelp('jargon.oid')}
              </InfoPopover>
              , and unquoted strings; fix auto-rewrites the file in place.
            </p>
          </header>

          <div className="grid gap-comfortable md:grid-cols-2">
            <label className="block text-sm">
              <span className="text-text-secondary">From walks/ directory</span>
              <select
                value={selectedFile}
                onChange={(e) => setSelectedFile(e.target.value)}
                disabled={filesLoading || files.length === 0}
                title="Hydrated from /api/v1/library/walks (the sandboxed walks directory). Use the absolute-path field to validate a walk outside this directory."
                className="mt-tight w-full rounded border border-surface-border bg-bg-base/60 px-3 py-row text-sm text-text-primary focus:border-status-info focus:outline-none disabled:opacity-50"
              >
                {filesLoading && <option>Loading…</option>}
                {!filesLoading && files.length === 0 && (
                  <option>{t('walkValidator.noWalksFound')}</option>
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
              <span className="text-text-secondary">{t('walkValidator.pastePathLabel')}</span>
              <input
                type="text"
                value={customPath}
                onChange={(e) => setCustomPath(e.target.value)}
                placeholder="/srv/niac/walks/cisco-c9300.walk"
                title="Absolute path to a walk file. Takes precedence over the dropdown selection. The path is bounded server-side; ../ traversal is rejected."
                className="mt-tight w-full rounded border border-surface-border bg-bg-base/60 px-3 py-row font-mono text-xs text-text-primary placeholder:text-text-muted focus:border-status-info focus:outline-none"
              />
            </label>
          </div>

          <div className="flex flex-wrap items-center gap-default">
            <button
              type="button"
              onClick={() => void run('validating')}
              disabled={busy !== 'idle' || !targetPath}
              title="Read-only validation: parses the walk and returns per-line issues. Doesn't modify the file."
              className="rounded bg-status-info/20 px-3 py-compact-md text-sm font-medium text-status-info ring-1 ring-status-info/40 hover:bg-status-info/30 disabled:opacity-50"
            >
              {busy === 'validating' ? 'Validating…' : 'Validate'}
            </button>
            <button
              type="button"
              onClick={() => setShowAutoFixConfirm(true)}
              disabled={busy !== 'idle' || !targetPath}
              className="rounded bg-status-warning/20 px-3 py-compact-md text-sm font-medium text-status-warning ring-1 ring-status-warning/40 hover:bg-status-warning/30 disabled:opacity-50"
              title="Validate and rewrite the file in place. A .bak is created next to the original."
            >
              {busy === 'fixing' ? 'Fixing…' : 'Auto-fix'}
            </button>
            <button
              type="button"
              onClick={() => void runBatch()}
              disabled={batchBusy}
              title="Validate every walk file referenced by the running config in one pass."
              className="rounded bg-bg-base/60 px-3 py-compact-md text-sm font-medium text-text-primary ring-1 ring-surface-border hover:bg-bg-base/80 disabled:opacity-50"
            >
              {batchBusy
                ? t('walkValidator.validatingAllButton')
                : t('walkValidator.validateAllButton')}
            </button>
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
              <h2 className="heading-3 text-text-primary">{response.message ?? 'Result'}</h2>
              <span className="text-sm text-text-muted">
                {response.result.totalLines} lines, {response.result.validLines} valid
              </span>
              <span
                className={`rounded px-cell py-0.5 text-xs font-medium ring-1 ${
                  response.result.valid
                    ? 'bg-status-success/20 text-status-success ring-status-success/40'
                    : 'bg-status-error/20 text-status-error ring-status-error/40'
                }`}
              >
                {response.result.valid ? 'VALID' : 'INVALID'}
              </span>
              {SEVERITY_ORDER.map((sev) => (
                <span
                  key={sev}
                  className={`rounded px-cell py-0.5 text-xs font-medium ring-1 ${SEVERITY_BADGE[sev]}`}
                >
                  {sev}: {counts[sev]}
                </span>
              ))}
              {typeof response.result.fixedCount === 'number' && (
                <span className="rounded bg-status-success/20 px-cell py-0.5 text-xs font-medium text-status-success ring-1 ring-status-success/40">
                  fixed: {response.result.fixedCount}
                </span>
              )}
            </header>

            {issues.length > 0 && (
              <label className="block text-sm">
                <span className="text-text-secondary">{t('walkValidator.oidFilterLabel')}</span>
                <input
                  type="text"
                  value={oidFilter}
                  onChange={(e) => setOidFilter(e.target.value)}
                  placeholder={t('walkValidator.oidFilterPlaceholder')}
                  title="Filters the full issue list by OID substring before applying the display cap below."
                  className="mt-tight w-full max-w-sm rounded border border-surface-border bg-bg-base/60 px-3 py-row font-mono text-xs text-text-primary placeholder:text-text-muted focus:border-status-info focus:outline-none"
                />
              </label>
            )}

            {issues.length === 0 ? (
              <p className="text-sm text-text-muted">No issues reported.</p>
            ) : filteredIssues.length === 0 ? (
              <p className="text-sm text-text-muted">{t('walkValidator.noMatchingIssues')}</p>
            ) : (
              <table className="w-full text-sm">
                <thead className="bg-bg-base/40 text-left text-xs uppercase tracking-wider text-text-muted">
                  <tr>
                    <th className="px-3 py-row w-16">{t('walkValidator.issuesTable.line')}</th>
                    <th className="px-3 py-row w-24">{t('walkValidator.issuesTable.severity')}</th>
                    <th className="px-3 py-row">{t('walkValidator.issuesTable.message')}</th>
                    <th className="px-3 py-row">{t('walkValidator.issuesTable.oid')}</th>
                    <th className="px-3 py-row">{t('walkValidator.issuesTable.original')}</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-knob/5">
                  {visibleIssues.map((issue, idx) => (
                    <tr
                      key={`${issue.line}-${idx}`}
                      className="text-text-primary hover:bg-bg-base/40"
                    >
                      <td className="px-3 py-row font-mono text-xs text-text-muted">
                        {issue.line}
                      </td>
                      <td className="px-3 py-row">
                        <span
                          className={`rounded px-cell py-0.5 text-[10px] font-medium ring-1 ${SEVERITY_BADGE[issue.severity as Severity] ?? ''}`}
                        >
                          {issue.severity}
                        </span>
                      </td>
                      <td className="px-3 py-row">{issue.message}</td>
                      <td className="px-3 py-row font-mono text-xs text-text-muted">
                        {issue.oid ?? ''}
                      </td>
                      <td className="px-3 py-row truncate font-mono text-xs text-text-muted">
                        {issue.original}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
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
              <span
                className={`rounded px-cell py-0.5 text-xs font-medium ring-1 ${
                  batchResponse.success
                    ? 'bg-status-success/20 text-status-success ring-status-success/40'
                    : 'bg-status-error/20 text-status-error ring-status-error/40'
                }`}
              >
                {batchResponse.success
                  ? t('walkValidator.batchStatusValid')
                  : t('walkValidator.batchStatusInvalid')}
              </span>
            </header>

            {batchResults.length === 0 ? (
              <p className="text-sm text-text-muted">{t('walkValidator.batchEmpty')}</p>
            ) : (
              <table className="w-full text-sm">
                <thead className="bg-bg-base/40 text-left text-xs uppercase tracking-wider text-text-muted">
                  <tr>
                    <th className="px-3 py-row">{t('walkValidator.batchTable.filename')}</th>
                    <th className="px-3 py-row w-24">{t('walkValidator.batchTable.status')}</th>
                    <th className="px-3 py-row w-24">{t('walkValidator.batchTable.issues')}</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-knob/5">
                  {batchResults.map((result) => (
                    <tr key={result.filename} className="text-text-primary hover:bg-bg-base/40">
                      <td className="px-3 py-row font-mono text-xs">{result.filename}</td>
                      <td className="px-3 py-row">
                        <span
                          className={`rounded px-cell py-0.5 text-[10px] font-medium ring-1 ${
                            result.valid
                              ? 'bg-status-success/20 text-status-success ring-status-success/40'
                              : 'bg-status-error/20 text-status-error ring-status-error/40'
                          }`}
                        >
                          {result.valid
                            ? t('walkValidator.batchStatusValid')
                            : t('walkValidator.batchStatusInvalid')}
                        </span>
                      </td>
                      <td className="px-3 py-row text-text-muted">{result.issues.length}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </CardContent>
        </Card>
      )}

      <ConfirmModal
        isOpen={showAutoFixConfirm}
        onConfirm={() => {
          setShowAutoFixConfirm(false);
          void run('fixing');
        }}
        onCancel={() => setShowAutoFixConfirm(false)}
        title={t('walkValidator.autoFixConfirmTitle')}
        message={t('walkValidator.autoFixConfirmMessage', { file: targetPath })}
        confirmLabel={tCommon('buttons.autoFix')}
        cancelLabel={tCommon('buttons.cancel')}
        confirmTone="blue"
      />
    </div>
  );
};
