import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import {
  createDevice,
  deleteDevice,
  fetchConfigDevice,
  fetchWalkFiles,
  updateDevice,
} from '../../api/client';
import type { Device } from '../../api/types';
import { useApiResource } from '../../hooks/useApiResource';
import { getErrorMessage } from '../../utils/format';
import { buildDeviceRawYaml, createEmptyDevice, type StatusMessage } from './deviceEditorUtils';

/**
 * Custom hook that manages device editor state and actions
 */
export function useDeviceEditor() {
  const { hostname } = useParams<{ hostname: string }>();
  const navigate = useNavigate();
  const isNewDevice = hostname === 'new';

  // State
  const [device, setDevice] = useState<Device>(createEmptyDevice());
  const [originalDevice, setOriginalDevice] = useState<Device | null>(null);
  const [saving, setSaving] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [message, setMessage] = useState<StatusMessage | null>(null);
  const [expandedSections, setExpandedSections] = useState<Set<string>>(new Set(['basic']));
  const [showYamlPreview, setShowYamlPreview] = useState(false);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);

  // Stable key counter for IP entries (avoids using array index as key)
  const ipKeyCounter = useRef(0);
  const ipKeysRef = useRef<number[]>([]);

  // Fetch existing device if editing
  const {
    data: fetchedDevice,
    loading,
    error,
    refetch,
  } = useApiResource(() => {
    if (isNewDevice || !hostname) {
      return Promise.resolve({ device: createEmptyDevice() });
    }
    return fetchConfigDevice(hostname);
  }, [hostname, isNewDevice]);

  // Fetch available walk files
  const { data: walkFiles } = useApiResource(fetchWalkFiles, []);

  // Update local state when fetched device changes
  useEffect(() => {
    if (fetchedDevice?.device) {
      setDevice(fetchedDevice.device);
      setOriginalDevice(fetchedDevice.device);
    }
  }, [fetchedDevice]);

  // Check if device has been modified
  const isDirty = useMemo(() => {
    if (isNewDevice) {
      return device.hostname.trim() !== '';
    }
    if (!originalDevice) {
      return false;
    }
    return JSON.stringify(device) !== JSON.stringify(originalDevice);
  }, [device, originalDevice, isNewDevice]);

  // Toggle section expansion
  const toggleSection = useCallback((section: string) => {
    setExpandedSections((prev) => {
      const next = new Set(prev);
      if (next.has(section)) {
        next.delete(section);
      } else {
        next.add(section);
      }
      return next;
    });
  }, []);

  // Handle form field changes
  const updateField = useCallback(<K extends keyof Device>(field: K, value: Device[K]) => {
    setDevice((prev) => ({ ...prev, [field]: value }));
    setMessage(null);
  }, []);

  // Handle save
  const handleSave = useCallback(async () => {
    if (!device.hostname.trim()) {
      setMessage({ type: 'error', text: 'Hostname is required' });
      return;
    }

    if (!device.mac.trim()) {
      setMessage({ type: 'error', text: 'MAC address is required' });
      return;
    }

    setSaving(true);
    setMessage(null);

    try {
      // FIX #294, #295: Serialize device config to rawYaml for backend
      const rawYaml = buildDeviceRawYaml(device);

      if (isNewDevice) {
        await createDevice({
          hostname: device.hostname,
          type: device.type,
          mac: device.mac,
          ip: device.ip,
          rawYaml: rawYaml,
        });
        setMessage({ type: 'success', text: 'Device created successfully' });
        // Navigate to the new device's edit page
        setTimeout(() => {
          navigate(`/device-config/${encodeURIComponent(device.hostname)}`);
        }, 500);
      } else {
        if (!hostname) {
          setMessage({ type: 'error', text: 'Missing hostname for update' });
          setSaving(false);
          return;
        }
        await updateDevice(hostname, { rawYaml: rawYaml });
        setMessage({ type: 'success', text: 'Device updated successfully' });
        setOriginalDevice(device);
        // If hostname changed, navigate to new URL
        if (device.hostname !== hostname) {
          setTimeout(() => {
            navigate(`/device-config/${encodeURIComponent(device.hostname)}`);
          }, 500);
        }
      }
    } catch (err) {
      setMessage({ type: 'error', text: getErrorMessage(err) });
    } finally {
      setSaving(false);
    }
  }, [device, hostname, isNewDevice, navigate]);

  // Handle delete
  const handleDelete = useCallback(async () => {
    if (!hostname || isNewDevice) {
      return;
    }

    setDeleting(true);
    try {
      await deleteDevice(hostname);
      navigate('/device-config');
    } catch (err) {
      setMessage({ type: 'error', text: getErrorMessage(err) });
      setDeleting(false);
    }
  }, [hostname, isNewDevice, navigate]);

  // Handle cancel/discard
  const handleDiscard = useCallback(() => {
    if (isNewDevice) {
      navigate('/device-config');
    } else if (originalDevice) {
      setDevice(originalDevice);
      setMessage(null);
    }
  }, [isNewDevice, originalDevice, navigate]);

  return {
    // Params
    hostname,
    isNewDevice,
    navigate,

    // State
    device,
    saving,
    deleting,
    message,
    expandedSections,
    showYamlPreview,
    showDeleteConfirm,
    loading,
    error,
    walkFiles,

    // Computed
    isDirty,

    // Refs
    ipKeyCounter,
    ipKeysRef,

    // Actions
    toggleSection,
    updateField,
    handleSave,
    handleDelete,
    handleDiscard,
    refetch,
    setShowYamlPreview,
    setShowDeleteConfirm,
  };
}
