import { useState, useCallback, useEffect, useRef } from 'react';
import {
  fetchProtocolDebugLevels,
  updateProtocolDebugLevels,
  resetProtocolDebugLevels,
} from '../api/client';
import type {
  DebugLevel,
  DebugProtocol,
  ProtocolCategory,
  ProtocolDebugConfig,
} from '../api/types';

// Default protocol configurations with categories
const DEFAULT_PROTOCOL_CONFIGS: ProtocolDebugConfig[] = [
  // Discovery protocols
  { protocol: 'SNMP', level: 'INFO', category: 'discovery' },
  { protocol: 'LLDP', level: 'INFO', category: 'discovery' },
  { protocol: 'CDP', level: 'INFO', category: 'discovery' },
  // Switching protocols
  { protocol: 'STP', level: 'INFO', category: 'switching' },
  { protocol: 'LACP', level: 'INFO', category: 'switching' },
  // Routing protocols
  { protocol: 'OSPF', level: 'INFO', category: 'routing' },
  { protocol: 'BGP', level: 'INFO', category: 'routing' },
  { protocol: 'EIGRP', level: 'INFO', category: 'routing' },
  { protocol: 'RIP', level: 'INFO', category: 'routing' },
  { protocol: 'ISIS', level: 'INFO', category: 'routing' },
  // Redundancy protocols
  { protocol: 'VRRP', level: 'INFO', category: 'redundancy' },
  { protocol: 'HSRP', level: 'INFO', category: 'redundancy' },
  { protocol: 'GLBP', level: 'INFO', category: 'redundancy' },
  { protocol: 'BFD', level: 'INFO', category: 'redundancy' },
  // MPLS (routing category)
  { protocol: 'MPLS', level: 'INFO', category: 'routing' },
  // Multicast protocols
  { protocol: 'PIM', level: 'INFO', category: 'multicast' },
  { protocol: 'IGMP', level: 'INFO', category: 'multicast' },
  { protocol: 'MSDP', level: 'INFO', category: 'multicast' },
  // Monitoring protocols
  { protocol: 'NetFlow', level: 'INFO', category: 'monitoring' },
];

// Debug levels in order from least to most verbose
export const DEBUG_LEVELS: DebugLevel[] = ['OFF', 'ERROR', 'WARN', 'INFO', 'DEBUG', 'TRACE'];

// Protocol categories with display names
export const PROTOCOL_CATEGORIES: Record<ProtocolCategory, string> = {
  discovery: 'Discovery Protocols',
  switching: 'Switching Protocols',
  routing: 'Routing Protocols',
  redundancy: 'Redundancy Protocols',
  multicast: 'Multicast Protocols',
  monitoring: 'Monitoring Protocols',
};

// Order for displaying categories
export const CATEGORY_ORDER: ProtocolCategory[] = [
  'discovery',
  'switching',
  'routing',
  'redundancy',
  'multicast',
  'monitoring',
];

export interface UseProtocolDebugLevelsResult {
  /** Current protocol debug configurations */
  protocols: ProtocolDebugConfig[];
  /** Whether data is being loaded */
  loading: boolean;
  /** Whether a save/reset operation is in progress */
  saving: boolean;
  /** Error message if any */
  error: string | null;
  /** Success message after save/reset */
  successMessage: string | null;
  /** Whether there are unsaved changes */
  hasChanges: boolean;
  /** Update the level for a specific protocol */
  updateLevel: (protocol: DebugProtocol, level: DebugLevel) => void;
  /** Save all changes to the API */
  saveChanges: () => Promise<void>;
  /** Reset all protocols to default levels */
  resetToDefaults: () => Promise<void>;
  /** Discard unsaved changes */
  discardChanges: () => void;
  /** Get protocols grouped by category */
  getProtocolsByCategory: () => Map<ProtocolCategory, ProtocolDebugConfig[]>;
  /** Clear any displayed messages */
  clearMessage: () => void;
}

export function useProtocolDebugLevels(): UseProtocolDebugLevelsResult {
  const [protocols, setProtocols] = useState<ProtocolDebugConfig[]>(DEFAULT_PROTOCOL_CONFIGS);
  const [savedProtocols, setSavedProtocols] = useState<ProtocolDebugConfig[]>(DEFAULT_PROTOCOL_CONFIGS);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);
  const messageTimeoutRef = useRef<number | null>(null);

  // Load initial data from API
  useEffect(() => {
    let cancelled = false;

    const loadData = async () => {
      try {
        const response = await fetchProtocolDebugLevels();
        if (!cancelled) {
          setProtocols(response.protocols);
          setSavedProtocols(response.protocols);
          setError(null);
        }
      } catch (err) {
        if (!cancelled) {
          // Use defaults if API fails
          setProtocols(DEFAULT_PROTOCOL_CONFIGS);
          setSavedProtocols(DEFAULT_PROTOCOL_CONFIGS);
          // Don't show error for initial load failure - just use defaults
          console.warn('Failed to load protocol debug levels, using defaults:', err);
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    };

    loadData();

    return () => {
      cancelled = true;
    };
  }, []);

  // Clear message after timeout
  const showMessage = useCallback((message: string, isError: boolean) => {
    if (messageTimeoutRef.current) {
      clearTimeout(messageTimeoutRef.current);
    }
    if (isError) {
      setError(message);
      setSuccessMessage(null);
    } else {
      setSuccessMessage(message);
      setError(null);
    }
    messageTimeoutRef.current = window.setTimeout(() => {
      setError(null);
      setSuccessMessage(null);
    }, 5000);
  }, []);

  const clearMessage = useCallback(() => {
    if (messageTimeoutRef.current) {
      clearTimeout(messageTimeoutRef.current);
    }
    setError(null);
    setSuccessMessage(null);
  }, []);

  // Check if there are unsaved changes
  const hasChanges = protocols.some((p, i) => {
    const saved = savedProtocols[i];
    return saved && p.level !== saved.level;
  });

  // Update the level for a specific protocol
  const updateLevel = useCallback((protocol: DebugProtocol, level: DebugLevel) => {
    setProtocols((prev) =>
      prev.map((p) =>
        p.protocol === protocol ? { ...p, level } : p
      )
    );
    clearMessage();
  }, [clearMessage]);

  // Save changes to API
  const saveChanges = useCallback(async () => {
    if (saving) return;

    setSaving(true);
    clearMessage();

    try {
      const changedProtocols = protocols.filter((p, i) => {
        const saved = savedProtocols[i];
        return saved && p.level !== saved.level;
      });

      if (changedProtocols.length === 0) {
        showMessage('No changes to save', false);
        setSaving(false);
        return;
      }

      const response = await updateProtocolDebugLevels({
        protocols: changedProtocols.map((p) => ({
          protocol: p.protocol,
          level: p.level,
        })),
      });

      setProtocols(response.protocols);
      setSavedProtocols(response.protocols);
      showMessage(`Updated ${changedProtocols.length} protocol(s)`, false);
    } catch (err) {
      showMessage((err as Error).message || 'Failed to save changes', true);
    } finally {
      setSaving(false);
    }
  }, [protocols, savedProtocols, saving, clearMessage, showMessage]);

  // Reset all protocols to defaults
  const resetToDefaults = useCallback(async () => {
    if (saving) return;

    setSaving(true);
    clearMessage();

    try {
      const response = await resetProtocolDebugLevels();
      setProtocols(response.protocols);
      setSavedProtocols(response.protocols);
      showMessage('All protocols reset to default levels', false);
    } catch {
      // If API fails, reset locally
      setProtocols(DEFAULT_PROTOCOL_CONFIGS);
      setSavedProtocols(DEFAULT_PROTOCOL_CONFIGS);
      showMessage('Reset to defaults (local only)', false);
    } finally {
      setSaving(false);
    }
  }, [saving, clearMessage, showMessage]);

  // Discard unsaved changes
  const discardChanges = useCallback(() => {
    setProtocols(savedProtocols);
    clearMessage();
  }, [savedProtocols, clearMessage]);

  // Get protocols grouped by category
  const getProtocolsByCategory = useCallback(() => {
    const grouped = new Map<ProtocolCategory, ProtocolDebugConfig[]>();

    for (const category of CATEGORY_ORDER) {
      grouped.set(category, []);
    }

    for (const protocol of protocols) {
      const list = grouped.get(protocol.category);
      if (list) {
        list.push(protocol);
      }
    }

    return grouped;
  }, [protocols]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (messageTimeoutRef.current) {
        clearTimeout(messageTimeoutRef.current);
      }
    };
  }, []);

  return {
    protocols,
    loading,
    saving,
    error,
    successMessage,
    hasChanges,
    updateLevel,
    saveChanges,
    resetToDefaults,
    discardChanges,
    getProtocolsByCategory,
    clearMessage,
  };
}
