import { Filter, LayoutGrid, LayoutList, Search, X } from 'lucide-react';
import type { FC } from 'react';
import { useTranslation } from 'react-i18next';
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
  const { t } = useTranslation('devices');
  return (
    <div className="flex flex-wrap gap-comfortable">
      {/* Search */}
      <div className="relative flex-1 min-w-[250px]">
        <Search className="absolute left-3 top-1/2 h-5 w-5 -translate-y-1/2 text-text-muted" />
        <input
          type="text"
          placeholder={t('list.searchPlaceholder')}
          value={searchQuery}
          onChange={(e) => onSearchChange(e.target.value)}
          className="w-full rounded-lg border border-surface-border bg-bg-base/60 py-2.5 pl-10 pr-icon text-sm text-text-primary placeholder:text-text-muted focus:border-brand-accent focus:outline-none"
        />
        {searchQuery && (
          <button
            type="button"
            onClick={() => onSearchChange('')}
            className="absolute right-3 top-1/2 -translate-y-1/2 text-text-muted hover:text-text-primary"
            aria-label={t('list.searchClearAria')}
          >
            <X className="h-4 w-4" />
          </button>
        )}
      </div>

      {/* Type filter */}
      <div className="flex items-center gap-compact">
        <Filter className={`${iconSizes.md} text-text-muted`} />
        <select
          value={typeFilter}
          onChange={(e) => onTypeFilterChange(e.target.value as DeviceType | 'all')}
          className="rounded-lg border border-surface-border bg-bg-base/60 py-row px-3 text-sm text-text-primary focus:border-brand-accent focus:outline-none"
        >
          <option value="all">{t('list.allTypes')}</option>
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
        className="rounded-lg border border-surface-border bg-bg-base/60 py-row px-3 text-sm text-text-primary focus:border-brand-accent focus:outline-none"
      >
        <option value="all">{t('list.allProtocols')}</option>
        {protocols.map((proto) => (
          <option key={proto} value={proto}>
            {proto}
          </option>
        ))}
      </select>

      {/* View toggle */}
      <div className="flex items-center gap-tight p-1 rounded-lg bg-bg-base/60 border border-surface-border">
        <button
          type="button"
          onClick={() => onViewModeChange('table')}
          className={`pad-xs rounded-md transition-colors ${
            viewMode === 'table'
              ? 'bg-brand-primary text-text-primary'
              : 'text-text-muted hover:text-text-primary hover:bg-surface-hover'
          }`}
          title={t('list.tableViewTitle')}
          aria-label={t('list.tableViewTitle')}
        >
          <LayoutList className={iconSizes.md} />
        </button>
        <button
          type="button"
          onClick={() => onViewModeChange('cards')}
          className={`pad-xs rounded-md transition-colors ${
            viewMode === 'cards'
              ? 'bg-brand-primary text-text-primary'
              : 'text-text-muted hover:text-text-primary hover:bg-surface-hover'
          }`}
          title={t('list.cardViewTitle')}
          aria-label={t('list.cardViewTitle')}
        >
          <LayoutGrid className={iconSizes.md} />
        </button>
      </div>
    </div>
  );
};
