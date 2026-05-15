import { type FC, useCallback, useEffect, useMemo, useState } from 'react';
import { fetchFiles, fixWalk, validateWalk } from '../api/client';
import type { FileEntry, WalkValidationIssue, WalkValidationResponse } from '../api/types';
import { Card, CardContent } from '../ui/Card';

type Severity = 'error' | 'warning' | 'info';

const SEVERITY_BADGE: Record<Severity, string> = {
  error: 'bg-red-500/20 text-red-200 ring-red-400/40',
  warning: 'bg-amber-500/20 text-amber-200 ring-amber-400/40',
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
  const [files, setFiles] = useState<FileEntry[]>([]);
  const [filesError, setFilesError] = useState<string | null>(null);
  const [filesLoading, setFilesLoading] = useState(true);

  const [selectedFile, setSelectedFile] = useState<string>('');
  const [customPath, setCustomPath] = useState<string>('');

  const [response, setResponse] = useState<WalkValidationResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState<'idle' | 'validating' | 'fixing'>('idle');

  // Hydrate the file dropdown from /api/v1/files?kind=walks.
  useEffect(() => {
    let cancelled = false;
    setFilesLoading(true);
    fetchFiles('walks')
      .then((entries) => {
        if (cancelled) return;
        setFiles(entries);
        setFilesError(null);
        if (entries.length > 0 && !selectedFile) {
          setSelectedFile(entries[0].path);
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
    <div className="space-y-6">
      <Card className="border-white/5 bg-gray-900/70">
        <CardContent className="space-y-4">
          <header>
            <h1 className="text-2xl font-semibold text-white">SNMP Walks</h1>
            <p className="text-sm text-gray-400">
              Same engine as <code>niac analyze-walk</code>. Validate detects malformed lines,
              missing OIDs, and unquoted strings; fix auto-rewrites the file in place.
            </p>
          </header>

          <div className="grid gap-4 md:grid-cols-2">
            <label className="block text-sm">
              <span className="text-gray-300">From walks/ directory</span>
              <select
                value={selectedFile}
                onChange={(e) => setSelectedFile(e.target.value)}
                disabled={filesLoading || files.length === 0}
                title="Hydrated from /api/v1/files?kind=walks (the sandboxed walks directory). Use the absolute-path field to validate a walk outside this directory."
                className="mt-1 w-full rounded border border-white/5 bg-gray-950/60 px-3 py-2 text-sm text-gray-100 focus:border-cyan-400 focus:outline-none disabled:opacity-50"
              >
                {filesLoading && <option>Loading…</option>}
                {!filesLoading && files.length === 0 && <option>No walks found</option>}
                {files.map((f) => (
                  <option key={f.path} value={f.path}>
                    {f.name} ({Math.round(f.sizeBytes / 1024)} KB)
                  </option>
                ))}
              </select>
              {filesError && <span className="mt-1 block text-xs text-red-300">{filesError}</span>}
            </label>

            <label className="block text-sm">
              <span className="text-gray-300">Or paste an absolute path</span>
              <input
                type="text"
                value={customPath}
                onChange={(e) => setCustomPath(e.target.value)}
                placeholder="/srv/niac/walks/cisco-c9300.walk"
                title="Absolute path to a walk file. Takes precedence over the dropdown selection. The path is bounded server-side; ../ traversal is rejected."
                className="mt-1 w-full rounded border border-white/5 bg-gray-950/60 px-3 py-2 font-mono text-xs text-gray-100 placeholder-gray-500 focus:border-cyan-400 focus:outline-none"
              />
            </label>
          </div>

          <div className="flex flex-wrap items-center gap-3">
            <button
              type="button"
              onClick={() => void run('validating')}
              disabled={busy !== 'idle' || !targetPath}
              title="Read-only validation: parses the walk and returns per-line issues. Doesn't modify the file."
              className="rounded bg-cyan-500/20 px-3 py-1.5 text-sm font-medium text-cyan-100 ring-1 ring-cyan-400/40 hover:bg-cyan-500/30 disabled:opacity-50"
            >
              {busy === 'validating' ? 'Validating…' : 'Validate'}
            </button>
            <button
              type="button"
              onClick={() => void run('fixing')}
              disabled={busy !== 'idle' || !targetPath}
              className="rounded bg-amber-500/20 px-3 py-1.5 text-sm font-medium text-amber-100 ring-1 ring-amber-400/40 hover:bg-amber-500/30 disabled:opacity-50"
              title="Validate and rewrite the file in place. A .bak is created next to the original."
            >
              {busy === 'fixing' ? 'Fixing…' : 'Auto-fix'}
            </button>
            {error && (
              <span className="text-sm text-red-300" role="alert">
                {error}
              </span>
            )}
          </div>
        </CardContent>
      </Card>

      {response?.result && (
        <Card className="border-white/5 bg-gray-900/70">
          <CardContent className="space-y-4">
            <header className="flex flex-wrap items-baseline gap-4">
              <h2 className="text-lg font-semibold text-white">{response.message ?? 'Result'}</h2>
              <span className="text-sm text-gray-400">
                {response.result.totalLines} lines, {response.result.validLines} valid
              </span>
              <span
                className={`rounded px-2 py-0.5 text-xs font-medium ring-1 ${
                  response.result.valid
                    ? 'bg-emerald-500/20 text-emerald-200 ring-emerald-400/40'
                    : 'bg-red-500/20 text-red-200 ring-red-400/40'
                }`}
              >
                {response.result.valid ? 'VALID' : 'INVALID'}
              </span>
              {SEVERITY_ORDER.map((sev) => (
                <span
                  key={sev}
                  className={`rounded px-2 py-0.5 text-xs font-medium ring-1 ${SEVERITY_BADGE[sev]}`}
                >
                  {sev}: {counts[sev]}
                </span>
              ))}
              {typeof response.result.fixedCount === 'number' && (
                <span className="rounded bg-emerald-500/20 px-2 py-0.5 text-xs font-medium text-emerald-200 ring-1 ring-emerald-400/40">
                  fixed: {response.result.fixedCount}
                </span>
              )}
            </header>

            {issues.length === 0 ? (
              <p className="text-sm text-gray-400">No issues reported.</p>
            ) : (
              <table className="w-full text-sm">
                <thead className="bg-gray-950/40 text-left text-xs uppercase tracking-wider text-gray-400">
                  <tr>
                    <th className="px-3 py-2 w-16">Line</th>
                    <th className="px-3 py-2 w-24">Severity</th>
                    <th className="px-3 py-2">Message</th>
                    <th className="px-3 py-2">Original</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-white/5">
                  {issues.slice(0, 200).map((issue, idx) => (
                    <tr key={`${issue.line}-${idx}`} className="text-gray-200 hover:bg-gray-950/40">
                      <td className="px-3 py-2 font-mono text-xs text-gray-400">{issue.line}</td>
                      <td className="px-3 py-2">
                        <span
                          className={`rounded px-2 py-0.5 text-[10px] font-medium ring-1 ${SEVERITY_BADGE[issue.severity as Severity] ?? ''}`}
                        >
                          {issue.severity}
                        </span>
                      </td>
                      <td className="px-3 py-2">{issue.message}</td>
                      <td className="px-3 py-2 truncate font-mono text-xs text-gray-500">
                        {issue.original}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
            {issues.length > 200 && (
              <p className="text-xs text-gray-500">
                Showing first 200 of {issues.length} issues — auto-fix to clear them all.
              </p>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  );
};
