import { useCallback, useEffect, useMemo, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import {
  createDevice,
  deleteDevice,
  fetchConfigDevice,
  fetchDeviceEditorSchema,
  fetchWalkFiles,
  updateDevice,
} from '../../api/client';
import type { Device, FileEntry } from '../../api/types';
import { useApiResource } from '../../hooks/useApiResource';
import { getErrorMessage } from '../../utils/format';
import type { StatusMessage } from './DeviceEditorHeader';

/**
 * Create an empty device with default values
 */
export const createEmptyDevice = (): Device => ({
  hostname: '',
  mac: '',
  type: 'switch',
  ip: '',
  ips: [],
});

export interface UseDeviceEditorReturn {
  // URL params
  hostname: string | undefined;
  isNewDevice: boolean;

  // Device state
  device: Device;
  originalDevice: Device | null;
  isDirty: boolean;

  // Loading state
  loading: boolean;
  error: Error | null;
  refetch: () => void;

  // Walk files for SNMP
  walkFiles: FileEntry[] | null;

  // Per-type visibility schema (#546 part 1). The editor uses this to
  // hide sections that don't apply to the currently picked
  // device.type — a switch shouldn't see DNS, a host shouldn't see
  // STP, etc. Always falls back to "show everything" on fetch error
  // so a network blip never breaks the form.
  visibleSections: Set<string>;

  // UI state
  saving: boolean;
  deleting: boolean;
  message: StatusMessage | null;
  expandedSections: Set<string>;
  showYamlPreview: boolean;
  showDeleteConfirm: boolean;

  // Actions
  navigate: ReturnType<typeof useNavigate>;
  setShowYamlPreview: (show: boolean) => void;
  setShowDeleteConfirm: (show: boolean) => void;
  setMessage: (message: StatusMessage | null) => void;
  toggleSection: (section: string) => void;
  updateField: <K extends keyof Device>(field: K, value: Device[K]) => void;
  handleSave: () => Promise<void>;
  handleDelete: () => Promise<void>;
  handleDiscard: () => void;
}

export const useDeviceEditor = (): UseDeviceEditorReturn => {
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

  // Per-type editor schema. Re-fetches when device.type changes; on
  // error or while loading we fall through to the "show everything"
  // set below so the form never disappears entirely.
  const { data: schema } = useApiResource(
    () => fetchDeviceEditorSchema(device.type || 'unknown'),
    [device.type],
  );
  const visibleSections = useMemo(() => {
    if (schema?.visibleSections && schema.visibleSections.length > 0) {
      return new Set(schema.visibleSections);
    }
    return new Set([
      'basic',
      'snmp',
      'lldp',
      'cdp',
      'stp',
      'ips',
      'dhcp',
      'dns',
      'http',
      'ftp',
      'netbios',
      'traffic',
    ]);
  }, [schema]);

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
      if (isNewDevice) {
        await createDevice(device);
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
        await updateDevice(hostname, device);
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
    hostname,
    isNewDevice,
    device,
    originalDevice,
    isDirty,
    loading,
    error,
    refetch,
    walkFiles,
    visibleSections,
    saving,
    deleting,
    message,
    expandedSections,
    showYamlPreview,
    showDeleteConfirm,
    navigate,
    setShowYamlPreview,
    setShowDeleteConfirm,
    setMessage,
    toggleSection,
    updateField,
    handleSave,
    handleDelete,
    handleDiscard,
  };
};
