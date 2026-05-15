import { Filter, LayoutGrid, LayoutList, Search, X } from 'lucide-react';
import type { FC } from 'react';
import type { DeviceType } from '../../api/types';
import { iconSizes } from '../../constants/sizes';

interface DeviceSearchFiltersProps {
  searchQuery: string;
  onSearchChange: (query: string) => void;
  typeFilter: DeviceType | 'all';
  onTypeFilterChange: (type: DeviceType | 'all') => void;
  protocolFilter: string;
  onProtocolFilterChange: (protocol: string) => void;
  viewMode: 'cards' | 'table';
  onViewModeChange: (mode: 'cards' | 'table') => void;
  deviceTypes: DeviceType[];
  protocols: string[];
}

export const DeviceSearchFilters: FC<DeviceSearchFiltersProps> = ({
  searchQuery,
  onSearchChange,
  typeFilter,
  onTypeFilterChange,
  protocolFilter,
  onProtocolFilterChange,
  viewMode,
  onViewModeChange,
  deviceTypes,
  protocols,
}) => {
  return (
    <div className="flex flex-wrap gap-4">
      {/* Search */}
      <div className="relative flex-1 min-w-[250px]">
        <Search className="absolute left-3 top-1/2 h-5 w-5 -translate-y-1/2 text-gray-400" />
        <input
          type="text"
          placeholder="Search by hostname, MAC, or IP..."
          value={searchQuery}
          onChange={(e) => onSearchChange(e.target.value)}
          className="w-full rounded-lg border border-white/10 bg-gray-950/60 py-2.5 pl-10 pr-10 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none"
        />
        {searchQuery && (
          <button
            type="button"
            onClick={() => onSearchChange('')}
            className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-white"
            aria-label="Clear search"
          >
            <X className="h-4 w-4" />
          </button>
        )}
      </div>

      {/* Type filter */}
      <div className="flex items-center gap-2">
        <Filter className={`${iconSizes.md} text-gray-400`} />
        <select
          value={typeFilter}
          onChange={(e) => onTypeFilterChange(e.target.value as DeviceType | 'all')}
          className="rounded-lg border border-white/10 bg-gray-950/60 py-2 px-3 text-sm text-white focus:border-violet-400 focus:outline-none"
        >
          <option value="all">All Types</option>
          {deviceTypes.map((type) => (
            <option key={type} value={type}>
              {type.replace('_', ' ')}
            </option>
          ))}
        </select>
      </div>

      {/* Protocol filter */}
      <select
        value={protocolFilter}
        onChange={(e) => onProtocolFilterChange(e.target.value)}
        className="rounded-lg border border-white/10 bg-gray-950/60 py-2 px-3 text-sm text-white focus:border-violet-400 focus:outline-none"
      >
        <option value="all">All Protocols</option>
        {protocols.map((proto) => (
          <option key={proto} value={proto}>
            {proto}
          </option>
        ))}
      </select>

      {/* View toggle */}
      <div className="flex items-center gap-1 p-1 rounded-lg bg-gray-950/60 border border-white/10">
        <button
          type="button"
          onClick={() => onViewModeChange('table')}
          className={`p-2 rounded-md transition-colors ${
            viewMode === 'table'
              ? 'bg-violet-600 text-white'
              : 'text-gray-400 hover:text-white hover:bg-white/10'
          }`}
          title="Table view"
          aria-label="Table view"
        >
          <LayoutList className={iconSizes.md} />
        </button>
        <button
          type="button"
          onClick={() => onViewModeChange('cards')}
          className={`p-2 rounded-md transition-colors ${
            viewMode === 'cards'
              ? 'bg-violet-600 text-white'
              : 'text-gray-400 hover:text-white hover:bg-white/10'
          }`}
          title="Card view"
          aria-label="Card view"
        >
          <LayoutGrid className={iconSizes.md} />
        </button>
      </div>
    </div>
  );
};
