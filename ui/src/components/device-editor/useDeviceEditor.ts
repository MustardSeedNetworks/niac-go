import { useCallback, useEffect, useMemo, useState } from 'react';
import { type FieldErrors, type Path, type PathValue, useForm } from 'react-hook-form';
import { useTranslation } from 'react-i18next';
import { useLocation, useNavigate, useParams } from 'react-router-dom';
import * as v from 'valibot';
import {
  createDevice,
  deleteDevice,
  fetchConfigDevice,
  fetchDeviceEditorSchema,
  updateDevice,
} from '../../api/client';
import { fetchLibraryWalks, type LibraryFileEntry } from '../../api/library-client';
import type { Device } from '../../api/types';
import { useApiResource } from '../../hooks/useApiResource';
import { DeviceFormSchema } from '../../schemas/forms';
import { getErrorMessage } from '../../utils/format';
import type { StatusMessage } from './DeviceEditorHeader';
import { useUnsavedChangesGuard } from './useUnsavedChangesGuard';

/**
 * Create an empty device with default values
 */
export const createEmptyDevice = (): Device => ({
  hostname: '',
  mac: '',
  type: 'switch',
  ip: '',
  ips: [],
  interfaceDetails: [],
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
  walkFiles: LibraryFileEntry[] | null;

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
  // Inline, per-field validation errors surfaced next to the offending
  // input (hostname / mac / ip). Populated on a failed save from the
  // valibot DeviceFormSchema; cleared as the user edits the field.
  fieldErrors: FieldErrors<Device>;
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

  // Unsaved-changes navigation guard (#920 — the editor previously lost
  // edits silently on navigate-away). requestNavigateBack replaces a bare
  // `navigate('/device-config')` for the "Back" button; pendingLeavePath /
  // confirmLeave / cancelLeave drive the confirmation modal rendered by
  // DeviceEditorPage.
  requestNavigateBack: () => void;
  pendingLeavePath: string | null;
  confirmLeave: () => void;
  cancelLeave: () => void;
}

export const useDeviceEditor = (): UseDeviceEditorReturn => {
  const { t } = useTranslation('devices');
  const { hostname } = useParams<{ hostname: string }>();
  const navigate = useNavigate();
  const location = useLocation();
  const isNewDevice = hostname === 'new' || location.pathname === '/device-config/new';

  // react-hook-form owns the device values. Inputs use the custom
  // `updateField` setter (not `register`), so the form runs headless:
  // `watch()` exposes current values, `safeParse` validates on save, and
  // `setError` drives the inline field errors. Keeping this contract means
  // the ~14 section components stay unchanged.
  const form = useForm<Device>({ defaultValues: createEmptyDevice() });
  const { setValue, getValues, watch, reset, setError, clearErrors, formState } = form;
  const device = watch();

  // State
  const [originalDevice, setOriginalDevice] = useState<Device | null>(null);
  const [saving, setSaving] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [message, setMessage] = useState<StatusMessage | null>(null);
  const [expandedSections, setExpandedSections] = useState<Set<string>>(
    () => new Set(location.hash === '#snmp' ? ['basic', 'snmp'] : ['basic']),
  );
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
  const { data: walkFiles } = useApiResource(fetchLibraryWalks, []);

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
      'interfaces',
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
      reset(fetchedDevice.device);
      setOriginalDevice(fetchedDevice.device);
    }
  }, [fetchedDevice, reset]);

  // Deep-link support: the Running Devices walk browser links here with
  // `#snmp` so a copied walk name can actually be used. Once the section
  // has rendered (post-loading), scroll it into view.
  useEffect(() => {
    if (location.hash !== '#snmp' || loading) {
      return;
    }
    document.getElementById('snmp-section')?.scrollIntoView({ behavior: 'smooth', block: 'start' });
  }, [location.hash, loading]);

  // Check if device has been modified. Deliberately kept as a structural
  // JSON compare (not rhf's formState.isDirty) so the unsaved-changes
  // navigation guard behaves exactly as before the rhf migration.
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
  const updateField = useCallback(
    <K extends keyof Device>(field: K, value: Device[K]) => {
      // Bridge the ergonomic `keyof Device` public signature to rhf's
      // `Path`/`PathValue` types — a top-level key is always a valid path,
      // but the compiler can't prove it through the generic.
      setValue(field as Path<Device>, value as PathValue<Device, Path<Device>>, {
        shouldDirty: true,
      });
      clearErrors(field as Path<Device>);
      setMessage(null);
    },
    [setValue, clearErrors],
  );

  // Handle save
  const handleSave = useCallback(async () => {
    const current = getValues();

    // Format-level validation (hostname / mac / ip) before the round-trip.
    // The Go side validates the full structure; this catches the common
    // mistakes inline. Uses safeParse so unrelated device sections pass
    // through untouched.
    clearErrors();
    const result = v.safeParse(DeviceFormSchema, current);
    if (!result.success) {
      // Surface the first issue per field. valibot reports pipe issues in
      // order, so an empty hostname shows "Hostname is required" (minLength)
      // rather than the later format message.
      const flagged = new Set<string>();
      for (const issue of result.issues) {
        const key = issue.path?.[0]?.key;
        if (typeof key === 'string' && !flagged.has(key)) {
          flagged.add(key);
          setError(key as Path<Device>, { message: issue.message });
        }
      }
      setMessage({
        type: 'error',
        text: result.issues[0]?.message ?? t('editor.messages.fixHighlightedFields'),
      });
      return;
    }

    setSaving(true);
    setMessage(null);

    try {
      if (isNewDevice) {
        await createDevice(current);
        setMessage({ type: 'success', text: t('editor.messages.createdSuccess') });
        // Navigate to the new device's edit page
        setTimeout(() => {
          navigate(`/device-config/${encodeURIComponent(current.hostname)}`);
        }, 500);
      } else {
        if (!hostname) {
          setMessage({ type: 'error', text: t('editor.messages.missingHostname') });
          setSaving(false);
          return;
        }
        await updateDevice(hostname, current);
        setMessage({ type: 'success', text: t('editor.messages.updatedSuccess') });
        setOriginalDevice(current);
        // If hostname changed, navigate to new URL
        if (current.hostname !== hostname) {
          setTimeout(() => {
            navigate(`/device-config/${encodeURIComponent(current.hostname)}`);
          }, 500);
        }
      }
    } catch (err) {
      setMessage({ type: 'error', text: getErrorMessage(err) });
    } finally {
      setSaving(false);
    }
  }, [getValues, clearErrors, setError, hostname, isNewDevice, navigate, t]);

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
      reset(originalDevice);
      setMessage(null);
    }
  }, [isNewDevice, originalDevice, navigate, reset]);

  const {
    pendingPath: pendingLeavePath,
    requestNavigate: requestNavigateBackTo,
    confirmNavigate: confirmLeave,
    cancelNavigate: cancelLeave,
  } = useUnsavedChangesGuard(isDirty, navigate);
  const requestNavigateBack = useCallback(
    () => requestNavigateBackTo('/device-config'),
    [requestNavigateBackTo],
  );

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
    fieldErrors: formState.errors,
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
    requestNavigateBack,
    pendingLeavePath,
    confirmLeave,
    cancelLeave,
  };
};
