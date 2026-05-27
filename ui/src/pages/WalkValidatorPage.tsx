import { type FC, useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { fetchLibraryWalks, fixWalk, type LibraryFileEntry, validateWalk } from '../api/client';
import type { WalkValidationIssue, WalkValidationResponse } from '../api/types';
import { Card, CardContent } from '../ui/Card';

type Severity = 'error' | 'warning' | 'info';

const SEVERITY_BADGE: Record<Severity, string> = {
  error: 'bg-status-error/20 text-status-error ring-status-error/40',
  warning: 'bg-status-warning/20 text-status-warning ring-status-warning/40',
  info: 'bg-sky-500/20 text-sky-200 ring-sky-400/40',
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
  const [files, setFiles] = useState<LibraryFileEntry[]>([]);
  const [filesError, setFilesError] = useState<string | null>(null);
  const [filesLoading, setFilesLoading] = useState(true);

  const [selectedFile, setSelectedFile] = useState<string>('');
  const [customPath, setCustomPath] = useState<string>('');

  const [response, setResponse] = useState<WalkValidationResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState<'idle' | 'validating' | 'fixing'>('idle');

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

  const issues = response?.result?.issues ?? [];
  const counts = useMemo(() => severityCounts(issues), [issues]);

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
      } catch (err) {
        setError((err as Error).message);
      } finally {
        setBusy('idle');
      }
    },
    [targetPath],
  );

  return (
    <div className="stack-xl">
      <Card className="border-surface-border bg-bg-surface/70">
        <CardContent className="stack-lg">
          <header>
            <h1 className="text-2xl font-semibold text-text-primary">
              {t('walkValidator.pageTitle')}
            </h1>
            <p className="text-sm text-text-muted">
              Same engine as <code>niac analyze-walk</code>. Validate detects malformed lines,
              missing OIDs, and unquoted strings; fix auto-rewrites the file in place.
            </p>
          </header>

          <div className="grid gap-comfortable md:grid-cols-2">
            <label className="block text-sm">
              <span className="text-text-secondary">From walks/ directory</span>
              <select
                value={selectedFile}
                onChange={(e) => setSelectedFile(e.target.value)}
                disabled={filesLoading || files.length === 0}
                title="Hydrated from /api/v1/files?kind=walks (the sandboxed walks directory). Use the absolute-path field to validate a walk outside this directory."
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
                className="mt-tight w-full rounded border border-surface-border bg-bg-base/60 px-3 py-row font-mono text-xs text-text-primary placeholder-gray-500 focus:border-status-info focus:outline-none"
              />
            </label>
          </div>

          <div className="flex flex-wrap items-center gap-default">
            <button
              type="button"
              onClick={() => void run('validating')}
              disabled={busy !== 'idle' || !targetPath}
              title="Read-only validation: parses the walk and returns per-line issues. Doesn't modify the file."
              className="rounded bg-status-info/20 px-3 py-compact-md text-sm font-medium text-status-info ring-1 ring-cyan-400/40 hover:bg-status-info/30 disabled:opacity-50"
            >
              {busy === 'validating' ? 'Validating…' : 'Validate'}
            </button>
            <button
              type="button"
              onClick={() => void run('fixing')}
              disabled={busy !== 'idle' || !targetPath}
              className="rounded bg-status-warning/20 px-3 py-compact-md text-sm font-medium text-status-warning ring-1 ring-status-warning/40 hover:bg-status-warning/30 disabled:opacity-50"
              title="Validate and rewrite the file in place. A .bak is created next to the original."
            >
              {busy === 'fixing' ? 'Fixing…' : 'Auto-fix'}
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

            {issues.length === 0 ? (
              <p className="text-sm text-text-muted">No issues reported.</p>
            ) : (
              <table className="w-full text-sm">
                <thead className="bg-bg-base/40 text-left text-xs uppercase tracking-wider text-text-muted">
                  <tr>
                    <th className="px-3 py-row w-16">Line</th>
                    <th className="px-3 py-row w-24">Severity</th>
                    <th className="px-3 py-row">Message</th>
                    <th className="px-3 py-row">Original</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-white/5">
                  {issues.slice(0, 200).map((issue, idx) => (
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
                      <td className="px-3 py-row truncate font-mono text-xs text-text-muted">
                        {issue.original}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
            {issues.length > 200 && (
              <p className="text-xs text-text-muted">
                Showing first 200 of {issues.length} issues — auto-fix to clear them all.
              </p>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  );
};
