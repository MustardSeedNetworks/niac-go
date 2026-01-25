import { Plus, RefreshCw, Server } from 'lucide-react';
import type { FC } from 'react';
import type { DeviceType } from '../../api/types';
import { Button } from '../../ui/Button';
import { Card, CardContent } from '../../ui/Card';
import { H2, P } from '../../ui/Typography';
import { BulkActions, DeleteProgress, StatusMessage } from './DeviceActions';
import { DeviceFilters } from './DeviceFilters';

export interface DeviceListHeaderProps {
  deviceCount: number;
  loading: boolean;
  onRefresh: () => void;
  onAddDevice: () => void;
  // Filter props
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
  // Selection props
  selectedCount: number;
  onDeleteSelected: () => void;
  onClearSelection: () => void;
  // Progress props
  deleteProgress: { current: number; total: number } | null;
  // Message props
  message: { type: 'success' | 'error'; text: string } | null;
  onDismissMessage: () => void;
}

export const DeviceListHeader: FC<DeviceListHeaderProps> = ({
  deviceCount,
  loading,
  onRefresh,
  onAddDevice,
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
  selectedCount,
  onDeleteSelected,
  onClearSelection,
  deleteProgress,
  message,
  onDismissMessage,
}) => {
  return (
    <Card className="border-white/5 bg-gray-900/70">
      <CardContent className="space-y-4">
        <div className="flex flex-wrap items-center justify-between gap-4">
          <div>
            <H2 className="mb-0 flex items-center gap-2">
              <Server className="h-5 w-5 text-violet-300" />
              Device Configuration
            </H2>
            <P className="text-gray-400 mt-1">
              Manage network device configurations for simulation.
              {deviceCount > 0 && (
                <span className="ml-2 text-violet-300">{deviceCount} devices</span>
              )}
            </P>
          </div>
          <div className="flex gap-2">
            <Button
              variant="outline"
              leftIcon={<RefreshCw className="h-4 w-4" />}
              onClick={onRefresh}
              disabled={loading}
            >
              Refresh
            </Button>
            <Button tone="violet" leftIcon={<Plus className="h-4 w-4" />} onClick={onAddDevice}>
              Add Device
            </Button>
          </div>
        </div>

        <DeviceFilters
          searchQuery={searchQuery}
          onSearchChange={onSearchChange}
          typeFilter={typeFilter}
          onTypeFilterChange={onTypeFilterChange}
          protocolFilter={protocolFilter}
          onProtocolFilterChange={onProtocolFilterChange}
          viewMode={viewMode}
          onViewModeChange={onViewModeChange}
          deviceTypes={deviceTypes}
          protocols={protocols}
        />

        <BulkActions
          selectedCount={selectedCount}
          onDeleteSelected={onDeleteSelected}
          onClearSelection={onClearSelection}
        />

        {deleteProgress && (
          <DeleteProgress current={deleteProgress.current} total={deleteProgress.total} />
        )}

        {message && (
          <StatusMessage type={message.type} text={message.text} onDismiss={onDismissMessage} />
        )}
      </CardContent>
    </Card>
  );
};
