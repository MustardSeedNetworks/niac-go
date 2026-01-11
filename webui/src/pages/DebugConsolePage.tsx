import { useState, useCallback, useMemo, type FC } from 'react';
import { Settings2, ChevronDown, ChevronUp } from 'lucide-react';
import { Card, CardContent, Tag, Button } from '@krisarmstrong/web-foundation';
import { useLogStream } from '../hooks/useWebSocket';
import { LogFilters } from '../components/LogFilters';
import { LogViewer } from '../components/LogViewer';
import { ProtocolDebugLevels } from '../components/debug/ProtocolDebugLevels';
import type { LogEntry, LogLevel, Protocol } from '../api/types';

// Maximum number of logs to buffer
const MAX_LOG_BUFFER = 1000;

// Generate a unique ID for each log entry
function generateLogId(): string {
  return `${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
}

// Map incoming WebSocket data to LogEntry format
function mapToLogEntry(data: unknown): LogEntry | null {
  if (!data || typeof data !== 'object') {
    return null;
  }

  const rawLog = data as Record<string, unknown>;

  // Handle different log formats from the WebSocket
  const level = (rawLog.level as string)?.toUpperCase() as LogLevel;
  const validLevels: LogLevel[] = ['ERROR', 'WARN', 'INFO', 'DEBUG'];

  return {
    id: (rawLog.id as string) || generateLogId(),
    timestamp: (rawLog.timestamp as string) || new Date().toISOString(),
    level: validLevels.includes(level) ? level : 'INFO',
    protocol: (rawLog.protocol as Protocol) || (rawLog.source as string) || 'SYSTEM',
    message: (rawLog.message as string) || JSON.stringify(rawLog),
    source: rawLog.sourceIP as string | undefined,
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
  const [showDebugSettings, setShowDebugSettings] = useState(false);

  // Handle incoming log messages
  const handleMessage = useCallback((data: unknown) => {
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
  }, []);

  // WebSocket connection for log streaming
  const { connected, reconnecting, reconnect } = useLogStream({
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
        const matchesSource = log.source?.toLowerCase().includes(query) || false;
        if (!matchesMessage && !matchesProtocol && !matchesSource) {
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

  // Clear all logs
  const handleClear = useCallback(() => {
    if (window.confirm('Clear all logs? This action cannot be undone.')) {
      setLogs([]);
    }
  }, []);

  // Connection status display
  const connectionStatus = useMemo(() => {
    if (connected) {
      return {
        label: 'Connected',
        color: 'green' as const,
        indicator: 'bg-green-400 animate-pulse',
      };
    }
    if (reconnecting) {
      return {
        label: 'Reconnecting...',
        color: 'yellow' as const,
        indicator: 'bg-yellow-400 animate-pulse',
      };
    }
    return {
      label: 'Disconnected',
      color: 'red' as const,
      indicator: 'bg-red-400',
    };
  }, [connected, reconnecting]);

  return (
    <div className="space-y-6">
      {/* Main Console Card */}
      <Card className="border-white/5 bg-gray-900/70">
        <CardContent className="space-y-4">
          {/* Header with Connection Status */}
          <div className="flex items-center justify-between">
            <h2 className="text-xl font-semibold text-white">Debug Console</h2>
            <div className="flex items-center gap-3">
              {/* Debug Settings Toggle */}
              <Button
                variant="outline"
                size="sm"
                onClick={() => setShowDebugSettings(!showDebugSettings)}
                leftIcon={<Settings2 className="h-4 w-4" />}
                rightIcon={showDebugSettings ? <ChevronUp className="h-4 w-4" /> : <ChevronDown className="h-4 w-4" />}
                aria-expanded={showDebugSettings}
                aria-controls="debug-settings-panel"
              >
                Protocol Settings
              </Button>
              {/* Connection Status */}
              <button
                onClick={reconnect}
                className="flex items-center gap-2 rounded-lg border border-white/10 bg-gray-950/50 px-3 py-1.5 text-sm transition-colors hover:bg-gray-800/50"
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
            onLevelChange={setLevelFilter}
            onProtocolChange={setProtocolFilter}
            onSearchChange={setSearchQuery}
            onAutoScrollChange={setAutoScroll}
            onExport={handleExport}
            onClear={handleClear}
          />

          {/* Filter Status */}
          {(levelFilter !== 'All' || protocolFilter !== 'All' || searchQuery) && (
            <div className="flex flex-wrap items-center gap-2 text-sm text-gray-400">
              <span>Showing {filteredLogs.length} of {logs.length} logs</span>
              {levelFilter !== 'All' && (
                <Tag colorScheme="purple">Level: {levelFilter}</Tag>
              )}
              {protocolFilter !== 'All' && (
                <Tag colorScheme="blue">Protocol: {protocolFilter}</Tag>
              )}
              {searchQuery && (
                <Tag colorScheme="gray">Search: "{searchQuery}"</Tag>
              )}
              <button
                onClick={() => {
                  setLevelFilter('All');
                  setProtocolFilter('All');
                  setSearchQuery('');
                }}
                className="text-violet-400 hover:text-violet-300 underline"
              >
                Clear filters
              </button>
            </div>
          )}

          {/* Log Viewer */}
          <LogViewer
            logs={filteredLogs}
            searchQuery={searchQuery}
            autoScroll={autoScroll}
          />

          {/* Footer Info */}
          <div className="flex items-center justify-between text-xs text-gray-500">
            <span>
              Buffer: {logs.length}/{MAX_LOG_BUFFER} logs
              {logs.length >= MAX_LOG_BUFFER && ' (oldest logs will be removed)'}
            </span>
            <span>
              WebSocket: /ws/logs
            </span>
          </div>
        </CardContent>
      </Card>

      {/* Protocol Debug Settings Panel */}
      {showDebugSettings && (
        <div id="debug-settings-panel">
          <ProtocolDebugLevels />
        </div>
      )}
    </div>
  );
};
