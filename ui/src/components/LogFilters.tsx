import { Download, Pause, Play, Search, Trash2 } from 'lucide-react';
import { type ChangeEvent, type FC, memo } from 'react';
import { useTranslation } from 'react-i18next';
import type { LogLevel, Protocol } from '../api/types';
import { iconSizes } from '../constants/sizes';
import { Button } from '../ui/Button';
import { SmallText } from '../ui/Typography';

const LOG_LEVELS: Array<LogLevel | 'All'> = ['All', 'ERROR', 'WARN', 'INFO', 'DEBUG'];

const PROTOCOLS: Array<Protocol | 'All'> = [
  'All',
  'ARP',
  'ICMP',
  'DNS',
  'TCP',
  'UDP',
  'SNMP',
  'DHCP',
  'LLDP',
  'CDP',
  'HTTP',
  'HTTPS',
  'SSH',
  'TELNET',
];

export interface LogFiltersProps {
  levelFilter: LogLevel | 'All';
  protocolFilter: Protocol | 'All';
  searchQuery: string;
  autoScroll: boolean;
  logCount: number;
  paused?: boolean;
  onLevelChange: (level: LogLevel | 'All') => void;
  onProtocolChange: (protocol: Protocol | 'All') => void;
  onSearchChange: (query: string) => void;
  onAutoScrollChange: (enabled: boolean) => void;
  onPauseToggle?: () => void;
  onExport: () => void;
  onClear: () => void;
}

export const LogFilters: FC<LogFiltersProps> = memo(
  ({
    levelFilter,
    protocolFilter,
    searchQuery,
    autoScroll,
    logCount,
    paused = false,
    onLevelChange,
    onProtocolChange,
    onSearchChange,
    onAutoScrollChange,
    onPauseToggle,
    onExport,
    onClear,
  }) => {
    const { t } = useTranslation('common');
    const handleLevelChange = (e: ChangeEvent<HTMLSelectElement>) => {
      onLevelChange(e.target.value as LogLevel | 'All');
    };

    const handleProtocolChange = (e: ChangeEvent<HTMLSelectElement>) => {
      onProtocolChange(e.target.value as Protocol | 'All');
    };

    const handleSearchChange = (e: ChangeEvent<HTMLInputElement>) => {
      onSearchChange(e.target.value);
    };

    const handleAutoScrollChange = (e: ChangeEvent<HTMLInputElement>) => {
      onAutoScrollChange(e.target.checked);
    };

    return (
      <div className="stack">
        {/* Filter Controls Row */}
        <div className="flex flex-wrap items-center gap-comfortable">
          {/* Level Filter */}
          <div className="flex items-center gap-compact">
            <label htmlFor="level-filter" className="text-sm text-text-muted">
              Level:
            </label>
            <select
              id="level-filter"
              value={levelFilter}
              onChange={handleLevelChange}
              className="rounded-lg border border-surface-border bg-bg-base/60 px-3 py-compact-md text-sm text-text-primary focus:border-brand-accent focus:outline-none"
              aria-label="Filter by log level"
            >
              {LOG_LEVELS.map((level) => (
                <option key={level} value={level}>
                  {level}
                </option>
              ))}
            </select>
          </div>

          {/* Protocol Filter */}
          <div className="flex items-center gap-compact">
            <label htmlFor="protocol-filter" className="text-sm text-text-muted">
              Protocol:
            </label>
            <select
              id="protocol-filter"
              value={protocolFilter}
              onChange={handleProtocolChange}
              className="rounded-lg border border-surface-border bg-bg-base/60 px-3 py-compact-md text-sm text-text-primary focus:border-brand-accent focus:outline-none"
              aria-label="Filter by protocol"
            >
              {PROTOCOLS.map((protocol) => (
                <option key={protocol} value={protocol}>
                  {protocol}
                </option>
              ))}
            </select>
          </div>

          {/* Search Input */}
          <div className="flex flex-1 items-center gap-compact">
            <label htmlFor="log-search" className="text-sm text-text-muted">
              Search:
            </label>
            <div className="relative flex-1 max-w-md">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-text-muted" />
              <input
                id="log-search"
                type="text"
                value={searchQuery}
                onChange={handleSearchChange}
                placeholder="Filter logs..."
                className="w-full rounded-lg border border-surface-border bg-bg-base/60 py-compact-md pl-9 pr-3 text-sm text-text-primary placeholder:text-text-muted focus:border-brand-accent focus:outline-none"
                aria-label="Search logs"
              />
            </div>
          </div>
        </div>

        {/* Action Controls Row */}
        <div className="flex flex-wrap items-center justify-between gap-comfortable">
          <div className="flex items-center gap-comfortable">
            {/* Auto-scroll Toggle */}
            <label className="flex cursor-pointer items-center gap-compact">
              <input
                type="checkbox"
                checked={autoScroll}
                onChange={handleAutoScrollChange}
                className="h-4 w-4 rounded border-border-muted bg-bg-elevated text-brand-primary focus:ring-brand-primary focus:ring-offset-surface-base"
              />
              <span className="text-sm text-text-secondary">Auto-scroll</span>
            </label>

            {/* Log Count */}
            <SmallText className="text-text-muted">
              {t('plurals.logCount', { count: logCount })}
            </SmallText>
          </div>

          {/* Action Buttons */}
          <div className="flex items-center gap-compact">
            {onPauseToggle && (
              <Button
                variant={paused ? 'solid' : 'outline'}
                size="sm"
                onClick={onPauseToggle}
                leftIcon={
                  paused ? <Play className={iconSizes.md} /> : <Pause className={iconSizes.md} />
                }
                aria-label={paused ? 'Resume log stream' : 'Pause log stream'}
                className={
                  paused ? 'bg-status-success hover:bg-status-success border-status-success' : ''
                }
              >
                {paused ? 'Resume' : 'Pause'}
              </Button>
            )}
            <Button
              variant="outline"
              size="sm"
              onClick={onExport}
              leftIcon={<Download className={iconSizes.md} />}
              aria-label="Export logs to file"
            >
              Export
            </Button>
            <Button
              variant="ghost"
              size="sm"
              onClick={onClear}
              leftIcon={<Trash2 className="h-4 w-4" />}
              aria-label="Clear all logs"
            >
              Clear
            </Button>
          </div>
        </div>
      </div>
    );
  },
);

LogFilters.displayName = 'LogFilters';
