import { memo, type FC, type ChangeEvent } from 'react';
import { Search, Download, Trash2 } from 'lucide-react';
import { Button, SmallText } from '../ui';
import type { LogLevel, Protocol } from '../api/types';

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
  onLevelChange: (level: LogLevel | 'All') => void;
  onProtocolChange: (protocol: Protocol | 'All') => void;
  onSearchChange: (query: string) => void;
  onAutoScrollChange: (enabled: boolean) => void;
  onExport: () => void;
  onClear: () => void;
}

export const LogFilters: FC<LogFiltersProps> = memo(({
  levelFilter,
  protocolFilter,
  searchQuery,
  autoScroll,
  logCount,
  onLevelChange,
  onProtocolChange,
  onSearchChange,
  onAutoScrollChange,
  onExport,
  onClear,
}) => {
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
    <div className="space-y-3">
      {/* Filter Controls Row */}
      <div className="flex flex-wrap items-center gap-4">
        {/* Level Filter */}
        <div className="flex items-center gap-2">
          <label htmlFor="level-filter" className="text-sm text-gray-400">
            Level:
          </label>
          <select
            id="level-filter"
            value={levelFilter}
            onChange={handleLevelChange}
            className="rounded-lg border border-white/10 bg-gray-950/60 px-3 py-1.5 text-sm text-white focus:border-violet-400 focus:outline-none"
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
        <div className="flex items-center gap-2">
          <label htmlFor="protocol-filter" className="text-sm text-gray-400">
            Protocol:
          </label>
          <select
            id="protocol-filter"
            value={protocolFilter}
            onChange={handleProtocolChange}
            className="rounded-lg border border-white/10 bg-gray-950/60 px-3 py-1.5 text-sm text-white focus:border-violet-400 focus:outline-none"
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
        <div className="flex flex-1 items-center gap-2">
          <label htmlFor="log-search" className="text-sm text-gray-400">
            Search:
          </label>
          <div className="relative flex-1 max-w-md">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-500" />
            <input
              id="log-search"
              type="text"
              value={searchQuery}
              onChange={handleSearchChange}
              placeholder="Filter logs..."
              className="w-full rounded-lg border border-white/10 bg-gray-950/60 py-1.5 pl-9 pr-3 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none"
              aria-label="Search logs"
            />
          </div>
        </div>
      </div>

      {/* Action Controls Row */}
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div className="flex items-center gap-4">
          {/* Auto-scroll Toggle */}
          <label className="flex cursor-pointer items-center gap-2">
            <input
              type="checkbox"
              checked={autoScroll}
              onChange={handleAutoScrollChange}
              className="h-4 w-4 rounded border-gray-600 bg-gray-800 text-violet-500 focus:ring-violet-500 focus:ring-offset-gray-900"
            />
            <span className="text-sm text-gray-300">Auto-scroll</span>
          </label>

          {/* Log Count */}
          <SmallText className="text-gray-500">
            {logCount} log{logCount !== 1 ? 's' : ''} buffered
          </SmallText>
        </div>

        {/* Action Buttons */}
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={onExport}
            leftIcon={<Download className="h-4 w-4" />}
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
});

LogFilters.displayName = 'LogFilters';
