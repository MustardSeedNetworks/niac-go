import { LayoutGrid, LayoutList } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import type { DeviceType } from '../../api/types';
import { normalizeDeviceType } from '../../constants/device-types';
import { useDeviceTypeLabels } from '../../hooks/useDeviceTypeLabels';
import { SearchInput } from '../../shared/SearchInput';
import { SegmentedControl } from '../../shared/SegmentedControl';
import { Select } from '../../shared/Select';

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

export function DeviceSearchFilters({
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
}: DeviceSearchFiltersProps) {
  const { t } = useTranslation('devices');
  const { t: tCommon } = useTranslation('common');
  const typeLabels = useDeviceTypeLabels();

  return (
    <div className="flex flex-wrap items-center gap-2">
      <SearchInput
        value={searchQuery}
        onValueChange={onSearchChange}
        label={t('list.searchPlaceholder')}
        clearLabel={t('list.searchClearAria')}
        placeholder={t('list.searchPlaceholder')}
        className="min-w-56 flex-1"
      />
      <div className="w-44">
        <Select
          aria-label={t('list.typeFilterAria')}
          value={typeFilter}
          options={[
            { value: 'all', label: t('list.allTypes') },
            ...deviceTypes.map((type) => ({
              value: type,
              label: typeLabels[normalizeDeviceType(type)],
            })),
          ]}
          onValueChange={(value) => {
            const next = deviceTypes.find((type) => type === value);
            onTypeFilterChange(next ?? 'all');
          }}
        />
      </div>
      <div className="w-44">
        <Select
          aria-label={t('list.protocolFilterAria')}
          value={protocolFilter}
          options={[
            { value: 'all', label: t('list.allProtocols') },
            ...protocols.map((protocol) => ({ value: protocol, label: protocol })),
          ]}
          onValueChange={onProtocolFilterChange}
        />
      </div>
      <SegmentedControl
        label={tCommon('labels.view')}
        value={viewMode}
        onChange={(value) => onViewModeChange(value === 'cards' ? 'cards' : 'table')}
        options={[
          {
            value: 'table',
            label: t('list.tableViewTitle'),
            icon: <LayoutList className="h-4 w-4" />,
          },
          {
            value: 'cards',
            label: t('list.cardViewTitle'),
            icon: <LayoutGrid className="h-4 w-4" />,
          },
        ]}
      />
    </div>
  );
}
