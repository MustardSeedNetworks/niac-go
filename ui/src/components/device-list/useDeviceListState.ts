import {
  useCallback,
  useDeferredValue,
  useMemo,
  useOptimistic,
  useState,
  useTransition,
} from 'react';
import { cloneDevice, deleteDevice, fetchConfigDevices } from '../../api/client';
import type { Device, DeviceListResponse, DeviceType } from '../../api/types';
import { useApiResource } from '../../hooks/useApiResource';
import { getErrorMessage } from '../../utils/format';
import { safeGetItem, safeSetItem } from '../../utils/storage';
import type { StatusMessage } from './DeviceStatusMessage';
import {
  getDeviceProtocols,
  matchesProtocolFilter,
  matchesSearchQuery,
  PROTOCOL_RULES,
} from './deviceFilters';

export interface UseDeviceListStateReturn {
  // Data
  deviceList: DeviceListResponse | null;
  devices: Device[];
  filteredDevices: Device[];
  deviceTypes: DeviceType[];
  protocols: string[];
  loading: boolean;
  error: Error | null;
  // React 19: isPending indicates non-blocking filter updates
  isFilterPending: boolean;

  // Filters
  searchQuery: string;
  setSearchQuery: (query: string) => void;
  typeFilter: DeviceType | 'all';
  setTypeFilter: (type: DeviceType | 'all') => void;
  protocolFilter: string;
  setProtocolFilter: (protocol: string) => void;
  clearFilters: () => void;

  // View mode
  viewMode: 'cards' | 'table';
  handleViewModeChange: (mode: 'cards' | 'table') => void;

  // Selection
  selectedDevices: Set<string>;
  toggleDeviceSelection: (hostname: string) => void;
  handleSelectAll: () => void;
  clearSelection: () => void;

  // Messages
  message: StatusMessage | null;
  setMessage: (message: StatusMessage | null) => void;

  // Modals
  showDeleteConfirm: string | null;
  setShowDeleteConfirm: (hostname: string | null) => void;
  showCloneModal: string | null;
  setShowCloneModal: (hostname: string | null) => void;
  showBulkDeleteConfirm: boolean;
  setShowBulkDeleteConfirm: (show: boolean) => void;

  // Actions
  refetch: () => void;
  handleDelete: (hostname: string) => Promise<void>;
  handleClone: (hostname: string, newHostname: string) => Promise<void>;
  handleBulkDeleteConfirm: () => Promise<void>;
  handleDeviceProtocols: (device: Device) => string[];
}

export const useDeviceListState = (): UseDeviceListStateReturn => {
  const {
    data: deviceList,
    loading,
    error,
    refetch,
  } = useApiResource(fetchConfigDevices, [], { intervalMs: 30000 });

  const [searchQuery, setSearchQuery] = useState('');
  const [typeFilter, setTypeFilterState] = useState<DeviceType | 'all'>('all');
  const [protocolFilter, setProtocolFilterState] = useState<string>('all');
  const [viewMode, setViewMode] = useState<'cards' | 'table'>(() => {
    const stored = safeGetItem('niac-device-view-mode');
    return stored === 'cards' || stored === 'table' ? stored : 'table';
  });
  const [message, setMessage] = useState<StatusMessage | null>(null);
  const [selectedDevices, setSelectedDevices] = useState<Set<string>>(new Set());
  const [showDeleteConfirm, setShowDeleteConfirm] = useState<string | null>(null);
  const [showCloneModal, setShowCloneModal] = useState<string | null>(null);
  const [showBulkDeleteConfirm, setShowBulkDeleteConfirm] = useState(false);

  // React 19: useTransition for non-blocking filter updates
  const [isFilterPending, startTransition] = useTransition();

  // React 19: useOptimistic for instant UI feedback during delete
  const actualDevices = deviceList?.devices ?? [];
  const [optimisticDevices, removeOptimisticDevice] = useOptimistic(
    actualDevices,
    (state: Device[], hostnameToRemove: string) =>
      state.filter((d) => d.hostname !== hostnameToRemove),
  );

  const deferredSearchQuery = useDeferredValue(searchQuery);
  const devices = optimisticDevices;

  // Persist view mode
  const handleViewModeChange = useCallback((mode: 'cards' | 'table') => {
    setViewMode(mode);
    safeSetItem('niac-device-view-mode', mode);
  }, []);

  // Get unique device types for filter
  const deviceTypes = useMemo(() => {
    const types = new Set<DeviceType>();
    for (const d of devices) {
      if (d.type) {
        types.add(d.type);
      }
    }
    return Array.from(types);
  }, [devices]);

  // Get unique protocols for filter
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

  // Filter devices
  const filteredDevices = useMemo(() => {
    return devices.filter((device) => {
      if (!matchesSearchQuery(device, deferredSearchQuery)) {
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
  }, [devices, deferredSearchQuery, typeFilter, protocolFilter]);

  // Get protocols enabled for a device
  const handleDeviceProtocols = useCallback((device: Device) => getDeviceProtocols(device), []);

  // Handle device selection
  const toggleDeviceSelection = useCallback((hostname: string) => {
    setSelectedDevices((prev) => {
      const next = new Set(prev);
      if (next.has(hostname)) {
        next.delete(hostname);
      } else {
        next.add(hostname);
      }
      return next;
    });
  }, []);

  // Handle select all
  const handleSelectAll = useCallback(() => {
    if (selectedDevices.size === filteredDevices.length) {
      setSelectedDevices(new Set());
    } else {
      setSelectedDevices(new Set(filteredDevices.map((d) => d.hostname)));
    }
  }, [selectedDevices.size, filteredDevices]);

  // Clear selection
  const clearSelection = useCallback(() => {
    setSelectedDevices(new Set());
  }, []);

  // React 19: Wrap filter updates in startTransition for non-blocking UI
  const setTypeFilter = useCallback((type: DeviceType | 'all') => {
    startTransition(() => {
      setTypeFilterState(type);
    });
  }, []);

  const setProtocolFilter = useCallback((protocol: string) => {
    startTransition(() => {
      setProtocolFilterState(protocol);
    });
  }, []);

  // Clear filters
  const clearFilters = useCallback(() => {
    setSearchQuery('');
    startTransition(() => {
      setTypeFilterState('all');
      setProtocolFilterState('all');
    });
  }, []);

  // Handle device deletion with React 19 optimistic update
  const handleDelete = useCallback(
    async (hostname: string) => {
      // React 19: Optimistically remove device for instant feedback
      removeOptimisticDevice(hostname);
      setShowDeleteConfirm(null);
      setSelectedDevices((prev) => {
        const next = new Set(prev);
        next.delete(hostname);
        return next;
      });

      try {
        await deleteDevice(hostname);
        setMessage({
          type: 'success',
          text: `Device "${hostname}" deleted successfully`,
        });
        refetch();
      } catch (err) {
        // On error, refetch will restore the actual state
        setMessage({ type: 'error', text: getErrorMessage(err) });
        refetch();
      }
    },
    [refetch, removeOptimisticDevice],
  );

  // Handle device clone
  const handleClone = useCallback(
    async (hostname: string, newHostname: string) => {
      try {
        await cloneDevice(hostname, { newHostname: newHostname });
        setMessage({
          type: 'success',
          text: `Device cloned as "${newHostname}"`,
        });
        setShowCloneModal(null);
        refetch();
      } catch (err) {
        setMessage({ type: 'error', text: getErrorMessage(err) });
      }
    },
    [refetch],
  );

  // Handle bulk delete confirmation
  const handleBulkDeleteConfirm = useCallback(async () => {
    setShowBulkDeleteConfirm(false);

    let successCount = 0;
    let errorCount = 0;

    for (const hostname of selectedDevices) {
      try {
        await deleteDevice(hostname);
        successCount++;
      } catch {
        errorCount++;
      }
    }

    setSelectedDevices(new Set());
    refetch();

    if (errorCount === 0) {
      setMessage({
        type: 'success',
        text: `${successCount} devices deleted successfully`,
      });
    } else {
      setMessage({
        type: 'error',
        text: `Deleted ${successCount} devices, ${errorCount} failed`,
      });
    }
  }, [selectedDevices, refetch]);

  return {
    deviceList,
    devices,
    filteredDevices,
    deviceTypes,
    protocols,
    loading,
    error,
    // React 19: Expose filter pending state for UI feedback
    isFilterPending,
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
  };
};
