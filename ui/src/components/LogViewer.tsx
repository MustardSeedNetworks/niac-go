import { AlertCircle, Check, ChevronDown, ChevronRight, Copy, Terminal } from 'lucide-react';
import { type FC, memo, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type { LogEntry, LogLevel } from '../api/types';
import { iconSizes } from '../constants/sizes';
import { copyToClipboard } from '../utils/file';

/**
 * LogViewer Component
 *
 * Renders a scrollable list of log entries with level-based coloring,
 * search highlighting, expandable details, and copy-to-clipboard.
 * Supports auto-scroll to keep the latest entries visible.
 */
export interface LogViewerProps {
  logs: LogEntry[];
  searchQuery: string;
  autoScroll: boolean;
}

type LogLevelKey = Lowercase<LogLevel>;

const toLogLevelKey = (level: LogLevel): LogLevelKey => level.toLowerCase() as LogLevelKey;

// Level-based color classes for styling log entries
const LEVEL_COLORS: Record<
  LogLevelKey,
  { bg: string; text: string; border: string; accent: string }
> = {
  error: {
    bg: 'bg-status-error/10',
    text: 'text-status-error',
    border: 'border-status-error/30',
    accent: 'bg-status-error',
  },
  warn: {
    bg: 'bg-status-warning/10',
    text: 'text-status-warning',
    border: 'border-status-warning/30',
    accent: 'bg-status-warning',
  },
  info: {
    bg: 'bg-white/5',
    text: 'text-text-primary',
    border: 'border-white/10',
    accent: 'bg-status-info',
  },
  debug: {
    bg: 'bg-bg-muted/10',
    text: 'text-text-muted',
    border: 'border-border-muted/20',
    accent: 'bg-bg-muted',
  },
};

// Level badge colors
const LEVEL_BADGE_COLORS: Record<LogLevelKey, string> = {
  error: 'bg-status-error/20 text-status-error border-status-error/30',
  warn: 'bg-status-warning/20 text-status-warning border-status-warning/30',
  info: 'bg-status-info/20 text-status-info border-status-info/30',
  debug: 'bg-bg-muted/20 text-text-muted border-border-muted/30',
};

// Format timestamp for display
function formatTimestamp(timestamp: string): string {
  try {
    const date = new Date(timestamp);
    return date.toLocaleTimeString('en-US', {
      hour12: false,
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    });
  } catch {
    return timestamp;
  }
}

// Format full timestamp for details view
function formatFullTimestamp(timestamp: string): string {
  try {
    const date = new Date(timestamp);
    return date.toLocaleString('en-US', {
      year: 'numeric',
      month: 'short',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      hour12: false,
    });
  } catch {
    return timestamp;
  }
}

/**
 * Highlight matching substrings within text.
 *
 * Escapes regex special characters in the query, splits the text on matches,
 * and wraps matched portions in a `<mark>` element. Returns the original
 * text unchanged if the query is empty or the regex fails.
 */
function highlightText(text: string, query: string): React.ReactNode {
  if (!query.trim()) {
    return text;
  }

  try {
    const regex = new RegExp(`(${query.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')})`, 'gi');
    const parts = text.split(regex);

    let offset = 0;
    return parts.map((part) => {
      const key = `${offset}-${part}`;
      offset += part.length;
      if (part.toLowerCase() === query.toLowerCase()) {
        return (
          <mark key={key} className="bg-status-warning/40 text-status-warning rounded px-0.5">
            {part}
          </mark>
        );
      }
      return (
        <span key={key} className="whitespace-pre-wrap">
          {part}
        </span>
      );
    });
  } catch {
    return text;
  }
}

// Format JSON details for display
function formatDetails(details: Record<string, unknown>): string {
  return JSON.stringify(details, null, 2);
}

// Individual log entry component
const LogEntryRow: FC<{ log: LogEntry; searchQuery: string }> = memo(({ log, searchQuery }) => {
  const [expanded, setExpanded] = useState(false);
  const [copied, setCopied] = useState(false);

  const levelKey = toLogLevelKey(log.level);
  const colors = LEVEL_COLORS[levelKey] || LEVEL_COLORS.info;
  const badgeColor = LEVEL_BADGE_COLORS[levelKey] || LEVEL_BADGE_COLORS.info;

  const hasDetails = log.details && Object.keys(log.details).length > 0;

  const copyLog = useCallback(async () => {
    const fullLog = `[${log.timestamp}] [${log.level}] [${log.protocol}] ${log.message}${log.details ? `\n${formatDetails(log.details)}` : ''}`;
    try {
      await copyToClipboard(fullLog);
    } finally {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  }, [log]);

  const handleCopy = useCallback(
    (e: React.MouseEvent) => {
      e.stopPropagation();
      void copyLog();
    },
    [copyLog],
  );

  const toggleExpand = useCallback(() => {
    if (hasDetails) {
      setExpanded((prev) => !prev);
    }
  }, [hasDetails]);

  const handleRowKeyDown = useCallback(
    (event: React.KeyboardEvent<HTMLButtonElement>) => {
      if (!hasDetails) {
        return;
      }
      if (event.key === 'Enter' || event.key === ' ') {
        event.preventDefault();
        toggleExpand();
      }
    },
    [hasDetails, toggleExpand],
  );

  return (
    <div className={`border-b ${colors.border} transition-colors`}>
      {/* Main log row */}
      <div
        className={`flex items-start gap-2 px-3 py-2 font-mono text-sm ${colors.bg} hover:bg-white/5 transition-colors`}
      >
        <button
          type="button"
          onClick={toggleExpand}
          onKeyDown={handleRowKeyDown}
          disabled={!hasDetails}
          aria-expanded={hasDetails ? expanded : undefined}
          className={`flex items-start gap-2 flex-1 text-left ${hasDetails ? 'cursor-pointer' : 'cursor-default'}`}
        >
          {/* Expand indicator */}
          <span
            className={`shrink-0 mt-0.5 ${hasDetails ? 'text-text-muted' : 'text-transparent'}`}
          >
            {expanded ? (
              <ChevronDown className={iconSizes.md} />
            ) : (
              <ChevronRight className={iconSizes.md} />
            )}
          </span>

          {/* Level indicator bar */}
          <span className={`shrink-0 w-1 h-5 rounded-full ${colors.accent}`} />

          {/* Timestamp */}
          <span className="shrink-0 text-text-muted tabular-nums">
            {formatTimestamp(log.timestamp)}
          </span>

          {/* Level Badge */}
          <span
            className={`shrink-0 rounded border px-1.5 py-0.5 text-xs font-semibold uppercase ${badgeColor}`}
          >
            {log.level}
          </span>

          {/* Protocol Badge */}
          <span className="shrink-0 rounded border border-brand-500/30 bg-brand-500/20 px-1.5 py-0.5 text-xs font-semibold text-brand-400">
            {log.protocol}
          </span>

          {/* Message */}
          <span className={`flex-1 break-all ${colors.text}`}>
            {highlightText(log.message, searchQuery)}
          </span>

          {/* Source (if available) */}
          {log.source && (
            <span className="shrink-0 text-text-disabled text-xs font-mono">{log.source}</span>
          )}
        </button>

        {/* Copy button */}
        <button
          type="button"
          onClick={handleCopy}
          className="shrink-0 p-1 rounded hover:bg-white/10 text-text-muted hover:text-text-secondary transition-colors"
          title="Copy log entry"
          aria-label="Copy log entry"
        >
          {copied ? (
            <Check className={`${iconSizes.sm} text-status-success`} />
          ) : (
            <Copy className={iconSizes.sm} />
          )}
        </button>
      </div>

      {/* Expanded details */}
      {expanded && hasDetails && (
        <div className={`px-3 py-3 ${colors.bg} border-t ${colors.border}`}>
          <div className="ml-6 space-y-2">
            {/* Metadata row */}
            <div className="flex flex-wrap gap-4 text-xs text-text-muted">
              <span>
                <strong className="text-text-muted">Time:</strong>{' '}
                {formatFullTimestamp(log.timestamp)}
              </span>
              {log.source && (
                <span>
                  <strong className="text-text-muted">Source:</strong> {log.source}
                </span>
              )}
              <span>
                <strong className="text-text-muted">ID:</strong> {log.id}
              </span>
            </div>

            {/* Details JSON */}
            <div className="rounded-lg border border-white/10 bg-bg-base/70 overflow-hidden">
              <div className="flex items-center justify-between px-3 py-1.5 bg-bg-surface/50 border-b border-white/10">
                <span className="text-xs font-medium text-text-muted">Details</span>
                <button
                  type="button"
                  onClick={async (e) => {
                    e.stopPropagation();
                    if (log.details) {
                      await navigator.clipboard.writeText(formatDetails(log.details));
                    }
                  }}
                  className="p-1 rounded hover:bg-white/10 text-text-muted hover:text-text-secondary transition-colors"
                  title="Copy details JSON"
                  aria-label="Copy details JSON"
                >
                  <Copy className={iconSizes.xs} />
                </button>
              </div>
              <pre className="p-3 text-xs font-mono text-text-secondary overflow-x-auto">
                {formatDetails(log.details ?? {})}
              </pre>
            </div>
          </div>
        </div>
      )}
    </div>
  );
});

LogEntryRow.displayName = 'LogEntryRow';

export const LogViewer: FC<LogViewerProps> = memo(({ logs, searchQuery, autoScroll }) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const endRef = useRef<HTMLDivElement>(null);

  // Scroll to bottom when new logs arrive and auto-scroll is enabled
  useEffect(() => {
    if (autoScroll && endRef.current) {
      endRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [autoScroll]);

  // Memoize the rendered logs to prevent unnecessary re-renders
  const renderedLogs = useMemo(
    () => logs.map((log) => <LogEntryRow key={log.id} log={log} searchQuery={searchQuery} />),
    [logs, searchQuery],
  );

  // Calculate log statistics
  const stats = useMemo(
    () =>
      logs.reduce(
        (acc, log) => {
          acc[log.level] = (acc[log.level] || 0) + 1;
          return acc;
        },
        {} as Record<string, number>,
      ),
    [logs],
  );

  if (logs.length === 0) {
    return (
      <div
        ref={containerRef}
        className="flex h-96 items-center justify-center rounded-lg border border-white/10 bg-bg-base/50"
      >
        <div className="text-center space-y-3">
          <div
            className={`mx-auto ${iconSizes['3xl']} rounded-full bg-bg-elevated/50 flex items-center justify-center`}
          >
            <Terminal className={`${iconSizes.xl} text-text-disabled`} />
          </div>
          <div>
            <p className="text-text-muted font-medium">No logs to display</p>
            <p className="mt-1 text-sm text-text-disabled">
              Logs will appear here when the debug console receives data
            </p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-2">
      {/* Log Level Stats Bar */}
      <div className="flex items-center gap-4 text-xs">
        {Object.entries(stats).map(([level, count]) => {
          const colors = LEVEL_COLORS[toLogLevelKey(level as LogLevel)];
          const isError = level === 'ERROR';
          return (
            <div
              key={level}
              className={`flex items-center gap-1.5 px-2 py-1 rounded ${colors?.bg || 'bg-bg-muted/10'}`}
            >
              {isError && count > 0 && (
                <AlertCircle className={`${iconSizes.xs} text-status-error`} />
              )}
              <span className={`${colors?.accent || 'bg-bg-muted'} h-2 w-2 rounded-full`} />
              <span className={`font-medium ${colors?.text || 'text-text-muted'}`}>{level}</span>
              <span className="text-text-muted">{count}</span>
            </div>
          );
        })}
      </div>

      {/* Log Container */}
      <div
        ref={containerRef}
        className="h-[500px] overflow-y-auto rounded-lg border border-white/10 bg-bg-base/70 scrollbar-thin scrollbar-track-gray-900 scrollbar-thumb-gray-700"
        role="log"
        aria-label="Debug console log output"
        aria-live="polite"
      >
        {renderedLogs}
        <div ref={endRef} />
      </div>
    </div>
  );
});

LogViewer.displayName = 'LogViewer';
