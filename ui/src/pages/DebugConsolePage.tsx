import { ChevronDown, ChevronUp, Settings2 } from 'lucide-react';
import { type FC, useCallback, useMemo, useState } from 'react';
import type { LogEntry, LogLevel, Protocol } from '../api/types';
import { ProtocolDebugLevels } from '../components/debug/ProtocolDebugLevels';
import { LogFilters } from '../components/LogFilters';
import { LogViewer } from '../components/LogViewer';
import { iconSizes } from '../constants/sizes';
import { useLogStream } from '../hooks/useEventSource';
import { Button } from '../ui/Button';
import { Card, CardContent } from '../ui/Card';
import { ConfirmModal } from '../ui/ConfirmModal';
import { Tag } from '../ui/Tag';

// Maximum number of logs to buffer
const MAX_LOG_BUFFER = 1000;

// Generate a unique ID for each log entry
function generateLogId(): string {
  return `${Date.now()}-${Math.random().toString(36).substring(2, 11)}`;
}

// Map incoming SSE data to LogEntry format
function mapToLogEntry(data: unknown): LogEntry | null {
  if (!data || typeof data !== 'object') {
    return null;
  }

  const rawLog = data as Record<string, unknown>;

  // Handle different log formats from the SSE stream
  const level = (rawLog.level as string)?.toUpperCase() as LogLevel;
  const validLevels: LogLevel[] = ['ERROR', 'WARN', 'INFO', 'DEBUG'];

  return {
    id: (rawLog.id as string) || generateLogId(),
    timestamp: (rawLog.timestamp as string) || new Date().toISOString(),
    level: validLevels.includes(level) ? level : 'INFO',
    protocol: (rawLog.protocol as Protocol) || (rawLog.source as string) || 'SYSTEM',
    message: (rawLog.message as string) || JSON.stringify(rawLog),
    source: rawLog.sourceIp as string | undefined,
    details: rawLog.details as Record<string, unknown> | undefined,
  };
}

export const DebugConsolePage: FC = () => {
  // Log storage and filters
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [levelFilter, setLevelFilter] = useState<LogLevel | 'All'>('All');
  const [protocolFilter, setProtocolFilter] = useState<Protocol | 'All'>('All');
  const [searchQuery, setSearchQuery] = useState('');
  const [autoScroll, setAutoScroll] = useState(true);
  const [paused, setPaused] = useState(false);
  const [showDebugSettings, setShowDebugSettings] = useState(false);
  const [showClearConfirm, setShowClearConfirm] = useState(false);

  // Handle incoming log messages
  const handleMessage = useCallback(
    (data: unknown) => {
      // Skip adding logs when paused
      if (paused) {
        return;
      }

      const logEntry = mapToLogEntry(data);
      if (logEntry) {
        setLogs((prevLogs) => {
          const newLogs = [...prevLogs, logEntry];
          // Trim to max buffer size
          if (newLogs.length > MAX_LOG_BUFFER) {
            return newLogs.slice(-MAX_LOG_BUFFER);
          }
          return newLogs;
        });
      }
    },
    [paused],
  );

  // Toggle pause state
  const handlePauseToggle = useCallback(() => {
    setPaused((prev) => !prev);
  }, []);

  // SSE connection for log streaming (auto-reconnects)
  const { connected, reconnect } = useLogStream({
    onMessage: handleMessage,
  });

  // Filter logs based on current filters
  const filteredLogs = useMemo(() => {
    return logs.filter((log) => {
      // Level filter
      if (levelFilter !== 'All' && log.level !== levelFilter) {
        return false;
      }

      // Protocol filter
      if (protocolFilter !== 'All' && log.protocol !== protocolFilter) {
        return false;
      }

      // Search filter
      if (searchQuery.trim()) {
        const query = searchQuery.toLowerCase();
        const matchesMessage = log.message.toLowerCase().includes(query);
        const matchesProtocol = log.protocol.toLowerCase().includes(query);
        const matchesSource = log.source?.toLowerCase().includes(query);
        if (!(matchesMessage || matchesProtocol || matchesSource)) {
          return false;
        }
      }

      return true;
    });
  }, [logs, levelFilter, protocolFilter, searchQuery]);

  // Export logs to file
  const handleExport = useCallback(() => {
    const content = filteredLogs
      .map((log) => {
        const timestamp = new Date(log.timestamp).toISOString();
        return `[${timestamp}] [${log.level}] [${log.protocol}] ${log.message}`;
      })
      .join('\n');

    const blob = new Blob([content], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `niac-debug-${new Date().toISOString().replace(/[:.]/g, '-')}.log`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
  }, [filteredLogs]);

  // Request confirmation before clearing all buffered logs
  const handleClear = useCallback(() => {
    setShowClearConfirm(true);
  }, []);

  // Clear all logs once the user confirms
  const confirmClear = useCallback(() => {
    setShowClearConfirm(false);
    setLogs([]);
  }, []);

  // Connection status display
  const connectionStatus = useMemo(() => {
    if (paused) {
      return {
        label: 'Paused',
        color: 'yellow' as const,
        indicator: 'bg-status-warning',
      };
    }
    if (connected) {
      return {
        label: 'Live',
        color: 'green' as const,
        indicator: 'bg-status-success animate-pulse',
      };
    }
    // SSE auto-reconnects, so disconnected state is brief
    return {
      label: 'Connecting...',
      color: 'yellow' as const,
      indicator: 'bg-status-warning animate-pulse',
    };
  }, [connected, paused]);

  return (
    <div className="stack-xl">
      {/* Main Console Card */}
      <Card className="border-surface-border bg-bg-surface/70">
        <CardContent className="stack-lg">
          {/* Header with Connection Status */}
          <div className="flex-between">
            <h2 className="heading-2 text-text-primary">Debug Console</h2>
            <div className="flex items-center gap-default">
              {/* Debug Settings Toggle */}
              <Button
                variant="outline"
                size="sm"
                onClick={() => setShowDebugSettings(!showDebugSettings)}
                leftIcon={<Settings2 className={iconSizes.md} />}
                rightIcon={
                  showDebugSettings ? (
                    <ChevronUp className={iconSizes.md} />
                  ) : (
                    <ChevronDown className={iconSizes.md} />
                  )
                }
                aria-expanded={showDebugSettings}
                aria-controls="debug-settings-panel"
              >
                Protocol Settings
              </Button>
              {/* Connection Status */}
              <button
                type="button"
                onClick={reconnect}
                className="flex items-center gap-compact rounded-lg border border-surface-border bg-bg-base/50 px-3 py-compact-md text-sm transition-colors hover:bg-bg-elevated/50"
                title={connected ? 'Connected to log stream' : 'Click to reconnect'}
              >
                <span className={`h-2 w-2 rounded-full ${connectionStatus.indicator}`} />
                <Tag colorScheme={connectionStatus.color}>{connectionStatus.label}</Tag>
              </button>
            </div>
          </div>

          {/* Filters */}
          <LogFilters
            levelFilter={levelFilter}
            protocolFilter={protocolFilter}
            searchQuery={searchQuery}
            autoScroll={autoScroll}
            logCount={logs.length}
            paused={paused}
            onLevelChange={setLevelFilter}
            onProtocolChange={setProtocolFilter}
            onSearchChange={setSearchQuery}
            onAutoScrollChange={setAutoScroll}
            onPauseToggle={handlePauseToggle}
            onExport={handleExport}
            onClear={handleClear}
          />

          {/* Filter Status */}
          {(levelFilter !== 'All' || protocolFilter !== 'All' || searchQuery) && (
            <div className="flex flex-wrap items-center gap-compact text-sm text-text-muted">
              <span>
                Showing {filteredLogs.length} of {logs.length} logs
              </span>
              {levelFilter !== 'All' && <Tag colorScheme="purple">Level: {levelFilter}</Tag>}
              {protocolFilter !== 'All' && <Tag colorScheme="blue">Protocol: {protocolFilter}</Tag>}
              {searchQuery && <Tag colorScheme="gray">Search: "{searchQuery}"</Tag>}
              <button
                type="button"
                onClick={() => {
                  setLevelFilter('All');
                  setProtocolFilter('All');
                  setSearchQuery('');
                }}
                className="text-brand-accent hover:text-brand-accent underline"
              >
                Clear filters
              </button>
            </div>
          )}

          {/* Log Viewer */}
          <LogViewer logs={filteredLogs} searchQuery={searchQuery} autoScroll={autoScroll} />

          {/* Footer Info */}
          <div className="flex-between text-xs text-text-muted">
            <span>
              Buffer: {logs.length}/{MAX_LOG_BUFFER} logs
              {logs.length >= MAX_LOG_BUFFER && ' (oldest logs will be removed)'}
            </span>
            <span>SSE: /api/v1/stream/logs</span>
          </div>
        </CardContent>
      </Card>

      {/* Protocol Debug Settings Panel */}
      {showDebugSettings && (
        <div id="debug-settings-panel">
          <ProtocolDebugLevels />
        </div>
      )}

      {/* Clear logs confirmation modal */}
      <ConfirmModal
        isOpen={showClearConfirm}
        onConfirm={confirmClear}
        onCancel={() => setShowClearConfirm(false)}
        title="Clear All Logs"
        message="Clear all logs? This action cannot be undone."
        confirmLabel="Clear"
      />
    </div>
  );
};
