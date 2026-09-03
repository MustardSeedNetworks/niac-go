import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useLocation, useNavigate, useParams } from 'react-router';
import * as v from 'valibot';
import {
  createDevice,
  deleteDevice,
  fetchConfigDevice,
  fetchDeviceEditorSchema,
  updateDevice,
} from '../../api/client';
import { fetchLibraryWalks, type LibraryFileEntry } from '../../api/library-client';
import { useApiResource } from '../../hooks/useApiResource';
import { AuthoredDeviceSchema } from '../../schemas/forms';
import { parseAuthoredDevice, serializeAuthoredDevice } from '../../utils/authored-device-yaml';
import { getErrorMessage } from '../../utils/format';
import type { StatusMessage } from './DeviceEditorHeader';
import type { AuthoredDevice, AuthoredValue } from './generated/authored-device.generated';
import { DEVICE_SECTIONS } from './generated/sections.generated';
import { useUnsavedChangesGuard } from './useUnsavedChangesGuard';

/**
 * The identity fields the editor validates inline, and the only ones that can
 * carry a field error.
 */
export type IdentityField = 'name' | 'mac' | 'vendor' | 'ips';

export type DeviceFieldErrors = Partial<Record<IdentityField, string>>;

/** A new device starts as the smallest document the daemon will load. */
export const createEmptyDevice = (): AuthoredDevice => ({ type: 'switch' });

/**
 * The per-type schema names sections by the keys the hand-built editor used.
 * The generated manifest names them after the daemon's own YAML keys, which
 * differ in one place.
 */
const RELEVANCE_ALIASES: Record<string, string> = { snmp: 'snmp_agent' };

export interface UseDeviceEditorReturn {
  // URL params
  hostname: string | undefined;
  isNewDevice: boolean;

  // Device state — the authored YAML document itself, so what the editor
  // holds is what the daemon loads (P1b-2).
  device: AuthoredDevice;
  originalDevice: AuthoredDevice | null;
  isDirty: boolean;
  /** The document as it would be POSTed; also what the preview pane shows. */
  yaml: string;

  // Loading state
  loading: boolean;
  error: Error | null;
  refetch: () => void;

  // Walk files for SNMP
  walkFiles: LibraryFileEntry[] | null;

  /**
   * Generated sections in render order: the ones the daemon calls relevant to
   * this device type first, then the rest. Order only — every section renders,
   * because a hidden section is a field the author cannot reach while the
   * parity gate reports it bound.
   */
  sections: typeof DEVICE_SECTIONS;

  // UI state
  saving: boolean;
  deleting: boolean;
  message: StatusMessage | null;
  fieldErrors: DeviceFieldErrors;
  expandedSections: Set<string>;
  showYamlPreview: boolean;
  showDeleteConfirm: boolean;

  // Actions
  navigate: ReturnType<typeof useNavigate>;
  setShowYamlPreview: (show: boolean) => void;
  setShowDeleteConfirm: (show: boolean) => void;
  setMessage: (message: StatusMessage | null) => void;
  toggleSection: (section: string) => void;
  /** Replace one top-level key of the authored document. */
  updateField: (key: keyof AuthoredDevice, value: AuthoredValue) => void;
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

  const [device, setDevice] = useState<AuthoredDevice>(createEmptyDevice);
  const [originalDevice, setOriginalDevice] = useState<AuthoredDevice | null>(null);
  const [fieldErrors, setFieldErrors] = useState<DeviceFieldErrors>({});
  const [saving, setSaving] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [message, setMessage] = useState<StatusMessage | null>(null);
  const [expandedSections, setExpandedSections] = useState<Set<string>>(
    () => new Set(location.hash === '#snmp' ? ['basic', 'snmp_agent'] : ['basic']),
  );
  const [showYamlPreview, setShowYamlPreview] = useState(false);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);

  // Fetch existing device if editing. The single-device GET always serializes
  // `rawYaml`; that document, not the camelCase projection beside it, is what
  // the editor loads.
  const {
    data: fetched,
    loading,
    error,
    refetch,
  } = useApiResource(() => {
    if (isNewDevice || !hostname) {
      return Promise.resolve(null);
    }
    return fetchConfigDevice(hostname);
  }, [hostname, isNewDevice]);

  const { data: walkFiles } = useApiResource(fetchLibraryWalks, []);

  // Per-type editor schema. Re-fetches when the type changes; on error or
  // while loading the relevance order is simply the manifest's own.
  const { data: schema } = useApiResource(
    () => fetchDeviceEditorSchema(device.type ?? 'unknown'),
    [device.type],
  );
  const sections = useMemo(() => {
    const relevant = new Set(
      (schema?.visibleSections ?? []).map((key) => RELEVANCE_ALIASES[key] ?? key),
    );
    if (relevant.size === 0) {
      return DEVICE_SECTIONS;
    }
    return [
      ...DEVICE_SECTIONS.filter((section) => relevant.has(section.key)),
      ...DEVICE_SECTIONS.filter((section) => !relevant.has(section.key)),
    ];
  }, [schema]);

  useEffect(() => {
    if (!fetched) {
      return;
    }
    // A detail response without `rawYaml` is a device the daemon could not
    // serialize. Loading an empty document over it would let the next save
    // replace the real device with nothing, so refuse instead.
    const loaded = fetched.rawYaml ? parseAuthoredDevice(fetched.rawYaml) : null;
    if (!loaded) {
      setMessage({ type: 'error', text: t('editor.messages.missingDocument') });
      return;
    }
    setDevice(loaded);
    setOriginalDevice(loaded);
  }, [fetched, t]);

  // Deep-link support: the Running Devices walk browser links here with
  // `#snmp` so a copied walk name can actually be used. Once the section
  // has rendered (post-loading), scroll it into view.
  useEffect(() => {
    if (location.hash !== '#snmp' || loading) {
      return;
    }
    document
      .getElementById('snmp_agent-section')
      ?.scrollIntoView({ behavior: 'smooth', block: 'start' });
  }, [location.hash, loading]);

  // The serialized document is both what a save sends and what the preview
  // shows, so dirtiness is measured on it: clearing a field the author never
  // set is not an edit, and comparing the in-memory objects would say it was.
  const yaml = useMemo(() => serializeAuthoredDevice(device), [device]);
  const isDirty = useMemo(() => {
    if (isNewDevice) {
      return Boolean(device.name?.trim());
    }
    return originalDevice !== null && yaml !== serializeAuthoredDevice(originalDevice);
  }, [yaml, device.name, originalDevice, isNewDevice]);

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

  const updateField = useCallback((key: keyof AuthoredDevice, value: AuthoredValue) => {
    setDevice((prev) => ({ ...prev, [key]: value }));
    setFieldErrors((prev) => {
      if (!(key in prev)) {
        return prev;
      }
      const { [key as IdentityField]: _cleared, ...rest } = prev;
      return rest;
    });
    setMessage(null);
  }, []);

  const handleSave = useCallback(async () => {
    const result = v.safeParse(AuthoredDeviceSchema, device);
    if (!result.success) {
      const errors: DeviceFieldErrors = {};
      for (const issue of result.issues) {
        const key = issue.path?.[0]?.key;
        if (typeof key === 'string' && !(key in errors)) {
          errors[key as IdentityField] = issue.message;
        }
      }
      setFieldErrors(errors);
      setMessage({
        type: 'error',
        text: result.issues[0]?.message ?? t('editor.messages.fixHighlightedFields'),
      });
      return;
    }

    const name = result.output.name;
    setSaving(true);
    setMessage(null);

    try {
      if (isNewDevice) {
        await createDevice(name, yaml);
        setMessage({ type: 'success', text: t('editor.messages.createdSuccess') });
        setTimeout(() => {
          navigate(`/device-config/${encodeURIComponent(name)}`);
        }, 500);
      } else {
        if (!hostname) {
          setMessage({ type: 'error', text: t('editor.messages.missingHostname') });
          setSaving(false);
          return;
        }
        await updateDevice(hostname, yaml);
        setMessage({ type: 'success', text: t('editor.messages.updatedSuccess') });
        setOriginalDevice(device);
      }
    } catch (err) {
      setMessage({ type: 'error', text: getErrorMessage(err) });
    } finally {
      setSaving(false);
    }
  }, [device, yaml, hostname, isNewDevice, navigate, t]);

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

  const handleDiscard = useCallback(() => {
    if (isNewDevice) {
      navigate('/device-config');
    } else if (originalDevice) {
      setDevice(originalDevice);
      setFieldErrors({});
      setMessage(null);
    }
  }, [isNewDevice, originalDevice, navigate]);

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
    yaml,
    loading,
    error,
    refetch,
    walkFiles,
    sections,
    saving,
    deleting,
    message,
    fieldErrors,
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
