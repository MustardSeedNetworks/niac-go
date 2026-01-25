import { type FC, useCallback, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { fetchConfigDevices } from '../api/client';
import type { Device, DeviceType } from '../api/types';
import { CloneDeviceModal } from '../components/device-list/CloneDeviceModal';
import { ConfirmDeleteModal } from '../components/device-list/ConfirmDeleteModal';
import { DeviceCardView } from '../components/device-list/DeviceCardView';
import { DeviceListHeader } from '../components/device-list/DeviceListHeader';
import {
  EmptyState,
  ErrorState,
  LoadingState,
  NoResultsState,
} from '../components/device-list/DeviceListStates';
import { DeviceTableView } from '../components/device-list/DeviceTableView';
import {
  getDeviceProtocols,
  matchesProtocolFilter,
  matchesSearchQuery,
  PROTOCOL_RULES,
} from '../components/device-list/deviceListUtils';
import { useDeviceListHandlers } from '../components/device-list/useDeviceListHandlers';
import { useApiResource } from '../hooks/useApiResource';
import { ConfirmModal } from '../ui/ConfirmModal';

export const DeviceListPage: FC = () => {
  const navigate = useNavigate();

  const {
    data: deviceList,
    loading,
    error,
    refetch,
  } = useApiResource(fetchConfigDevices, [], { intervalMs: 30000 });

  const [searchQuery, setSearchQuery] = useState('');
  const [typeFilter, setTypeFilter] = useState<DeviceType | 'all'>('all');
  const [protocolFilter, setProtocolFilter] = useState<string>('all');
  const [viewMode, setViewMode] = useState<'cards' | 'table'>(() => {
    const stored = localStorage.getItem('niac-device-view-mode');
    return stored === 'cards' || stored === 'table' ? stored : 'table';
  });

  const {
    message,
    setMessage,
    deleteProgress,
    selectedDevices,
    showDeleteConfirm,
    setShowDeleteConfirm,
    showCloneModal,
    setShowCloneModal,
    showBulkDeleteConfirm,
    setShowBulkDeleteConfirm,
    toggleDeviceSelection,
    selectAllDevices,
    clearSelection,
    handleDelete,
    handleClone,
    handleBulkDeleteConfirm,
  } = useDeviceListHandlers({ refetch });

  const handleViewModeChange = useCallback((mode: 'cards' | 'table') => {
    setViewMode(mode);
    localStorage.setItem('niac-device-view-mode', mode);
  }, []);

  const devices = deviceList?.devices ?? [];

  const deviceTypes = useMemo(() => {
    const types = new Set<DeviceType>();
    for (const d of devices) {
      if (d.type) {
        types.add(d.type);
      }
    }
    return Array.from(types);
  }, [devices]);

  const protocols = useMemo(() => {
    const protos = new Set<string>();
    for (const d of devices) {
      for (const rule of PROTOCOL_RULES) {
        if (rule.isEnabled(d)) {
          protos.add(rule.label);
        }
      }
    }
    return Array.from(protos).sort();
  }, [devices]);

  const filteredDevices = useMemo(() => {
    const normalizedQuery = searchQuery.trim().toLowerCase();
    return devices.filter((device) => {
      if (normalizedQuery && !matchesSearchQuery(device, normalizedQuery)) {
        return false;
      }
      if (typeFilter !== 'all' && device.type !== typeFilter) {
        return false;
      }
      if (!matchesProtocolFilter(device, protocolFilter)) {
        return false;
      }
      return true;
    });
  }, [devices, searchQuery, typeFilter, protocolFilter]);

  const handleDeviceProtocols = useCallback((device: Device) => getDeviceProtocols(device), []);

  const handleSelectAll = useCallback(() => {
    if (selectedDevices.size === filteredDevices.length) {
      clearSelection();
    } else {
      selectAllDevices(filteredDevices.map((d) => d.hostname));
    }
  }, [selectedDevices.size, filteredDevices, clearSelection, selectAllDevices]);

  const clearFilters = useCallback(() => {
    setSearchQuery('');
    setTypeFilter('all');
    setProtocolFilter('all');
  }, []);

  const navigateToAddDevice = useCallback(() => navigate('/device-config/new'), [navigate]);

  return (
    <div className="space-y-6">
      <DeviceListHeader
        deviceCount={devices.length}
        loading={loading}
        onRefresh={refetch}
        onAddDevice={navigateToAddDevice}
        searchQuery={searchQuery}
        onSearchChange={setSearchQuery}
        typeFilter={typeFilter}
        onTypeFilterChange={setTypeFilter}
        protocolFilter={protocolFilter}
        onProtocolFilterChange={setProtocolFilter}
        viewMode={viewMode}
        onViewModeChange={handleViewModeChange}
        deviceTypes={deviceTypes}
        protocols={protocols}
        selectedCount={selectedDevices.size}
        onDeleteSelected={() => setShowBulkDeleteConfirm(true)}
        onClearSelection={clearSelection}
        deleteProgress={deleteProgress}
        message={message}
        onDismissMessage={() => setMessage(null)}
      />

      {loading && !deviceList && <LoadingState viewMode={viewMode} />}

      {error && <ErrorState errorMessage={error.message} onRetry={refetch} />}

      {deviceList && devices.length === 0 && <EmptyState onAddDevice={navigateToAddDevice} />}

      {devices.length > 0 && filteredDevices.length === 0 && (
        <NoResultsState onClearFilters={clearFilters} />
      )}

      {filteredDevices.length > 0 && viewMode === 'table' && (
        <DeviceTableView
          devices={filteredDevices}
          selectedDevices={selectedDevices}
          onSelectDevice={toggleDeviceSelection}
          onSelectAll={handleSelectAll}
          onEdit={(hostname) => navigate(`/device-config/${encodeURIComponent(hostname)}`)}
          onClone={(hostname) => setShowCloneModal(hostname)}
          onDelete={(hostname) => setShowDeleteConfirm(hostname)}
          getDeviceProtocols={handleDeviceProtocols}
        />
      )}

      {filteredDevices.length > 0 && viewMode === 'cards' && (
        <DeviceCardView
          devices={filteredDevices}
          selectedDevices={selectedDevices}
          onSelectDevice={toggleDeviceSelection}
          onSelectAll={handleSelectAll}
          onEdit={(hostname) => navigate(`/device-config/${encodeURIComponent(hostname)}`)}
          onClone={(hostname) => setShowCloneModal(hostname)}
          onDelete={(hostname) => setShowDeleteConfirm(hostname)}
          getDeviceProtocols={handleDeviceProtocols}
        />
      )}

      {showDeleteConfirm && (
        <ConfirmDeleteModal
          hostname={showDeleteConfirm}
          onConfirm={() => handleDelete(showDeleteConfirm)}
          onCancel={() => setShowDeleteConfirm(null)}
        />
      )}

      {showCloneModal && (
        <CloneDeviceModal
          hostname={showCloneModal}
          onClone={(newHostname) => handleClone(showCloneModal, newHostname)}
          onCancel={() => setShowCloneModal(null)}
        />
      )}

      <ConfirmModal
        isOpen={showBulkDeleteConfirm}
        onConfirm={handleBulkDeleteConfirm}
        onCancel={() => setShowBulkDeleteConfirm(false)}
        title="Delete Selected Devices"
        message={
          <>
            Are you sure you want to delete{' '}
            <strong>
              {selectedDevices.size} device
              {selectedDevices.size !== 1 ? 's' : ''}
            </strong>
            ? This action cannot be undone.
          </>
        }
        confirmLabel="Delete All"
        confirmTone="red"
      />
    </div>
  );
};
