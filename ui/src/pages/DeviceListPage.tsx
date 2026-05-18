import type { FC } from 'react';
import { useNavigate } from 'react-router-dom';
import { CloneDeviceModal } from '../components/device-list/CloneDeviceModal';
import { ConfirmDeleteModal } from '../components/device-list/ConfirmDeleteModal';
import { DeviceBulkActions } from '../components/device-list/DeviceBulkActions';
import { DeviceCardView } from '../components/device-list/DeviceCardView';
import { DeviceListHeader } from '../components/device-list/DeviceListHeader';
import {
  DeviceListEmptyState,
  DeviceListErrorState,
  DeviceListLoadingState,
  DeviceListNoResultsState,
} from '../components/device-list/DeviceListStates';
import { DeviceSearchFilters } from '../components/device-list/DeviceSearchFilters';
import { DeviceStatusMessage } from '../components/device-list/DeviceStatusMessage';
import { DeviceTableView } from '../components/device-list/DeviceTableView';
import { useDeviceListState } from '../components/device-list/useDeviceListState';
import { DeviceListProvider } from '../contexts/DeviceListContext';
import { Card, CardContent } from '../ui/Card';
import { ConfirmModal } from '../ui/ConfirmModal';

export const DeviceListPage: FC = () => {
  const navigate = useNavigate();

  const {
    deviceList,
    devices,
    filteredDevices,
    deviceTypes,
    protocols,
    loading,
    error,
    searchQuery,
    setSearchQuery,
    typeFilter,
    setTypeFilter,
    protocolFilter,
    setProtocolFilter,
    clearFilters,
    viewMode,
    handleViewModeChange,
    selectedDevices,
    toggleDeviceSelection,
    handleSelectAll,
    clearSelection,
    message,
    setMessage,
    showDeleteConfirm,
    setShowDeleteConfirm,
    showCloneModal,
    setShowCloneModal,
    showBulkDeleteConfirm,
    setShowBulkDeleteConfirm,
    refetch,
    handleDelete,
    handleClone,
    handleBulkDeleteConfirm,
    handleDeviceProtocols,
  } = useDeviceListState();

  return (
    <div className="space-y-6">
      {/* Header section */}
      <Card className="border-surface-border bg-bg-surface/70">
        <CardContent className="space-y-4">
          <DeviceListHeader
            deviceCount={devices.length}
            filteredDevices={filteredDevices}
            loading={loading}
            onRefresh={refetch}
          />

          <DeviceSearchFilters
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
          />

          <DeviceBulkActions
            selectedCount={selectedDevices.size}
            onDeleteSelected={() => setShowBulkDeleteConfirm(true)}
            onClearSelection={clearSelection}
          />

          <DeviceStatusMessage message={message} onDismiss={() => setMessage(null)} />
        </CardContent>
      </Card>

      {/* Loading state */}
      {loading && !deviceList && <DeviceListLoadingState viewMode={viewMode} />}

      {/* Error state */}
      {error && <DeviceListErrorState error={error} onRetry={refetch} />}

      {/* Empty state */}
      {deviceList && devices.length === 0 && <DeviceListEmptyState />}

      {/* No search results */}
      {devices.length > 0 && filteredDevices.length === 0 && (
        <DeviceListNoResultsState onClearFilters={clearFilters} />
      )}

      {/* Device list views */}
      {filteredDevices.length > 0 && (
        <DeviceListProvider
          devices={filteredDevices}
          selectedDevices={selectedDevices}
          onSelectDevice={toggleDeviceSelection}
          onSelectAll={handleSelectAll}
          onEdit={(hostname) => navigate(`/device-config/${encodeURIComponent(hostname)}`)}
          onClone={(hostname) => setShowCloneModal(hostname)}
          onDelete={(hostname) => setShowDeleteConfirm(hostname)}
          getDeviceProtocols={handleDeviceProtocols}
        >
          {viewMode === 'table' && <DeviceTableView />}
          {viewMode === 'cards' && <DeviceCardView />}
        </DeviceListProvider>
      )}

      {/* Delete confirmation modal */}
      {showDeleteConfirm && (
        <ConfirmDeleteModal
          hostname={showDeleteConfirm}
          onConfirm={() => handleDelete(showDeleteConfirm)}
          onCancel={() => setShowDeleteConfirm(null)}
        />
      )}

      {/* Clone device modal */}
      {showCloneModal && (
        <CloneDeviceModal
          hostname={showCloneModal}
          onClone={(newHostname) => handleClone(showCloneModal, newHostname)}
          onCancel={() => setShowCloneModal(null)}
        />
      )}

      {/* Bulk delete confirmation modal */}
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
