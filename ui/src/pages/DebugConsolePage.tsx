import { ChevronDown, ChevronUp, Settings2 } from 'lucide-react';
import { type FC, useCallback, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import type { LogEntry, LogLevel, Protocol } from '../api/types';
import { DebugLevelControl } from '../components/debug/DebugLevelControl';
import { LogFilters } from '../components/LogFilters';
import { LogViewer } from '../components/LogViewer';
import { iconSizes } from '../constants/sizes';
import { useLogStream } from '../hooks/useEventSource';
import { useUIStore } from '../stores/ui-store';
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
  const { t } = useTranslation('pages');
  const { t: tCommon } = useTranslation('common');
  const addNotification = useUIStore((s) => s.addNotification);
  // Only toast on a drop after a real connection was established — the
  // initial "still connecting" window on mount isn't a disconnect.
  const hasConnectedRef = useRef(false);
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

  // Fires when the stream drops after having connected at least once — the
  // browser's EventSource auto-reconnects, so this is informational only.
  const handleDisconnect = useCallback(() => {
    if (!hasConnectedRef.current) {
      return;
    }
    addNotification({
      type: 'warning',
      title: tCommon('toast.streamDisconnectedTitle'),
      message: tCommon('toast.streamDisconnectedMessage'),
    });
  }, [addNotification, tCommon]);

  const handleConnect = useCallback(() => {
    hasConnectedRef.current = true;
  }, []);

  // SSE connection for log streaming (auto-reconnects)
  const { connected, reconnect } = useLogStream({
    onMessage: handleMessage,
    onConnect: handleConnect,
    onDisconnect: handleDisconnect,
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
        label: t('debug.statusPaused'),
        color: 'yellow' as const,
        indicator: 'bg-status-warning',
      };
    }
    if (connected) {
      return {
        label: t('debug.statusLive'),
        color: 'green' as const,
        indicator: 'bg-status-success animate-pulse',
      };
    }
    // SSE auto-reconnects, so disconnected state is brief
    return {
      label: t('debug.statusConnecting'),
      color: 'yellow' as const,
      indicator: 'bg-status-warning animate-pulse',
    };
  }, [connected, paused, t]);

  return (
    <div className="stack-xl">
      {/* Main Console Card */}
      <Card className="border-surface-border bg-bg-surface/70">
        <CardContent className="stack-lg">
          {/* Header with Connection Status */}
          <div className="flex-between">
            <h2 className="heading-2 text-text-primary">{t('debug.title')}</h2>
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
                {t('debug.debugLevelButton')}
              </Button>
              {/* Connection Status */}
              <button
                type="button"
                onClick={reconnect}
                className="flex items-center gap-compact rounded-lg border border-surface-border bg-bg-base/50 px-3 py-compact-md text-sm transition-colors hover:bg-bg-elevated/50"
                title={connected ? t('debug.connectedTitle') : t('debug.clickToReconnectTitle')}
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
                {t('debug.showingCount', {
                  filtered: filteredLogs.length,
                  total: logs.length,
                })}
              </span>
              {levelFilter !== 'All' && (
                <Tag colorScheme="purple">{t('debug.levelTag', { level: levelFilter })}</Tag>
              )}
              {protocolFilter !== 'All' && (
                <Tag colorScheme="blue">{t('debug.protocolTag', { protocol: protocolFilter })}</Tag>
              )}
              {searchQuery && (
                <Tag colorScheme="gray">{t('debug.searchTag', { query: searchQuery })}</Tag>
              )}
              <button
                type="button"
                onClick={() => {
                  setLevelFilter('All');
                  setProtocolFilter('All');
                  setSearchQuery('');
                }}
                className="text-brand-accent hover:text-brand-accent underline"
              >
                {t('debug.clearFiltersButton')}
              </button>
            </div>
          )}

          {/* Log Viewer */}
          <LogViewer logs={filteredLogs} searchQuery={searchQuery} autoScroll={autoScroll} />

          {/* Footer Info */}
          <div className="flex-between text-xs text-text-muted">
            <span>
              {t('debug.bufferStatus', { count: logs.length, max: MAX_LOG_BUFFER })}
              {logs.length >= MAX_LOG_BUFFER && ` ${t('debug.bufferOldestRemoved')}`}
            </span>
            <span>{t('debug.sseEndpointLabel')}</span>
          </div>
        </CardContent>
      </Card>

      {/* Global debug-level control */}
      {showDebugSettings && (
        <div id="debug-settings-panel">
          <Card className="border-surface-border bg-bg-surface/70">
            <CardContent className="stack-lg">
              <DebugLevelControl />
            </CardContent>
          </Card>
        </div>
      )}

      {/* Clear logs confirmation modal */}
      <ConfirmModal
        isOpen={showClearConfirm}
        onConfirm={confirmClear}
        onCancel={() => setShowClearConfirm(false)}
        title={t('debug.clearAllLogsTitle')}
        message={t('debug.clearAllLogsMessage')}
        confirmLabel={t('debug.clearButton')}
      />
    </div>
  );
};
