import { memo, useEffect, useRef, useMemo, type FC } from 'react';
import type { LogEntry, LogLevel } from '../api/types';

export interface LogViewerProps {
  logs: LogEntry[];
  searchQuery: string;
  autoScroll: boolean;
}

// Level-based color classes for styling log entries
const LEVEL_COLORS: Record<LogLevel, { bg: string; text: string; border: string }> = {
  ERROR: {
    bg: 'bg-red-500/10',
    text: 'text-red-400',
    border: 'border-red-500/30',
  },
  WARN: {
    bg: 'bg-yellow-500/10',
    text: 'text-yellow-400',
    border: 'border-yellow-500/30',
  },
  INFO: {
    bg: 'bg-white/5',
    text: 'text-gray-200',
    border: 'border-white/10',
  },
  DEBUG: {
    bg: 'bg-gray-500/10',
    text: 'text-gray-500',
    border: 'border-gray-500/20',
  },
};

// Level badge colors
const LEVEL_BADGE_COLORS: Record<LogLevel, string> = {
  ERROR: 'bg-red-500/20 text-red-400 border-red-500/30',
  WARN: 'bg-yellow-500/20 text-yellow-400 border-yellow-500/30',
  INFO: 'bg-blue-500/20 text-blue-400 border-blue-500/30',
  DEBUG: 'bg-gray-500/20 text-gray-500 border-gray-500/30',
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

// Highlight matching text in a string
function highlightText(text: string, query: string): React.ReactNode {
  if (!query.trim()) {
    return text;
  }

  try {
    const regex = new RegExp(`(${query.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')})`, 'gi');
    const parts = text.split(regex);

    return parts.map((part, index) => {
      if (part.toLowerCase() === query.toLowerCase()) {
        return (
          <mark key={index} className="bg-yellow-400/40 text-yellow-200 rounded px-0.5">
            {part}
          </mark>
        );
      }
      return part;
    });
  } catch {
    return text;
  }
}

// Individual log entry component
const LogEntryRow: FC<{ log: LogEntry; searchQuery: string }> = memo(({ log, searchQuery }) => {
  const colors = LEVEL_COLORS[log.level] || LEVEL_COLORS.INFO;
  const badgeColor = LEVEL_BADGE_COLORS[log.level] || LEVEL_BADGE_COLORS.INFO;

  return (
    <div
      className={`flex items-start gap-3 px-3 py-2 font-mono text-sm border-b ${colors.border} ${colors.bg} hover:bg-white/5 transition-colors`}
    >
      {/* Timestamp */}
      <span className="shrink-0 text-gray-500 tabular-nums">
        [{formatTimestamp(log.timestamp)}]
      </span>

      {/* Level Badge */}
      <span
        className={`shrink-0 rounded border px-1.5 py-0.5 text-xs font-semibold uppercase ${badgeColor}`}
      >
        {log.level}
      </span>

      {/* Protocol Badge */}
      <span className="shrink-0 rounded border border-violet-500/30 bg-violet-500/20 px-1.5 py-0.5 text-xs font-semibold text-violet-400">
        {log.protocol}
      </span>

      {/* Message */}
      <span className={`flex-1 break-all ${colors.text}`}>
        {highlightText(log.message, searchQuery)}
      </span>

      {/* Source (if available) */}
      {log.source && (
        <span className="shrink-0 text-gray-600 text-xs">
          {log.source}
        </span>
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
  }, [logs, autoScroll]);

  // Memoize the rendered logs to prevent unnecessary re-renders
  const renderedLogs = useMemo(() => {
    return logs.map((log) => (
      <LogEntryRow key={log.id} log={log} searchQuery={searchQuery} />
    ));
  }, [logs, searchQuery]);

  if (logs.length === 0) {
    return (
      <div
        ref={containerRef}
        className="flex h-96 items-center justify-center rounded-lg border border-white/10 bg-gray-950/50"
      >
        <div className="text-center">
          <p className="text-gray-400">No logs to display</p>
          <p className="mt-1 text-sm text-gray-600">
            Logs will appear here when the debug console receives data
          </p>
        </div>
      </div>
    );
  }

  return (
    <div
      ref={containerRef}
      className="h-[500px] overflow-y-auto rounded-lg border border-white/10 bg-gray-950/70 scrollbar-thin scrollbar-track-gray-900 scrollbar-thumb-gray-700"
      role="log"
      aria-label="Debug console log output"
      aria-live="polite"
    >
      {renderedLogs}
      <div ref={endRef} />
    </div>
  );
});

LogViewer.displayName = 'LogViewer';
