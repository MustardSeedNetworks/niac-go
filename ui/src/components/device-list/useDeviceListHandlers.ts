import { useCallback, useState } from 'react';
import { cloneDevice, deleteDevice } from '../../api/client';
import { getErrorMessage } from '../../utils/format';

export interface UseDeviceListHandlersOptions {
  refetch: () => void;
}

export interface MessageState {
  type: 'success' | 'error';
  text: string;
}

export interface DeleteProgressState {
  current: number;
  total: number;
}

export function useDeviceListHandlers({ refetch }: UseDeviceListHandlersOptions) {
  const [message, setMessage] = useState<MessageState | null>(null);
  const [deleteProgress, setDeleteProgress] = useState<DeleteProgressState | null>(null);
  const [selectedDevices, setSelectedDevices] = useState<Set<string>>(new Set());
  const [showDeleteConfirm, setShowDeleteConfirm] = useState<string | null>(null);
  const [showCloneModal, setShowCloneModal] = useState<string | null>(null);
  const [showBulkDeleteConfirm, setShowBulkDeleteConfirm] = useState(false);

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

  const selectAllDevices = useCallback((hostnames: string[]) => {
    setSelectedDevices(new Set(hostnames));
  }, []);

  const clearSelection = useCallback(() => {
    setSelectedDevices(new Set());
  }, []);

  const handleDelete = useCallback(
    async (hostname: string) => {
      try {
        await deleteDevice(hostname);
        setMessage({
          type: 'success',
          text: `Device "${hostname}" deleted successfully`,
        });
        setShowDeleteConfirm(null);
        setSelectedDevices((prev) => {
          const next = new Set(prev);
          next.delete(hostname);
          return next;
        });
        refetch();
      } catch (err) {
        setMessage({ type: 'error', text: getErrorMessage(err) });
      }
    },
    [refetch],
  );

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

  const handleBulkDeleteConfirm = useCallback(async () => {
    setShowBulkDeleteConfirm(false);

    const hostnames = Array.from(selectedDevices);
    const total = hostnames.length;
    let successCount = 0;
    let errorCount = 0;

    setDeleteProgress({ current: 0, total });

    const batchSize = 5;
    for (let i = 0; i < hostnames.length; i += batchSize) {
      const batch = hostnames.slice(i, i + batchSize);
      const results = await Promise.allSettled(batch.map((h) => deleteDevice(h)));

      for (const result of results) {
        if (result.status === 'fulfilled') {
          successCount++;
        } else {
          errorCount++;
        }
      }

      setDeleteProgress({ current: successCount + errorCount, total });
    }

    setDeleteProgress(null);
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
    // State
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
    // Actions
    toggleDeviceSelection,
    selectAllDevices,
    clearSelection,
    handleDelete,
    handleClone,
    handleBulkDeleteConfirm,
  };
}
