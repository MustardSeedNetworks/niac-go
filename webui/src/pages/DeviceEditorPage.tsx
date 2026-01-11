import { type FC, useState, useEffect, useCallback, useMemo } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
  Server,
  ArrowLeft,
  Save,
  Trash2,
  AlertCircle,
  Check,
  ChevronDown,
  ChevronRight,
  HelpCircle,
  RefreshCw,
  Router,
  Monitor,
  Wifi,
  Shield,
  HardDrive,
  Cpu,
} from 'lucide-react';
import {
  Card,
  CardContent,
  Button,
  H2,
  SmallText,
  Tag,
} from '@krisarmstrong/web-foundation';
import { useApiResource } from '../hooks/useApiResource';
import {
  fetchConfigDevice,
  createDevice,
  updateDevice,
  deleteDevice,
  fetchWalkFiles,
} from '../api/client';
import type {
  Device,
  DeviceType,
  SNMPAgent,
  LLDPConfig,
  CDPConfig,
  STPConfig,
  FileEntry,
} from '../api/types';
import { YamlEditor } from '../components/config/YamlEditor';

// Device type icons
const deviceTypeIcons: Record<DeviceType, typeof Server> = {
  router: Router,
  switch: Server,
  access_point: Wifi,
  firewall: Shield,
  server: HardDrive,
  workstation: Monitor,
  iot: Cpu,
  unknown: Server,
};

// Device type options
const deviceTypes: { value: DeviceType; label: string }[] = [
  { value: 'router', label: 'Router' },
  { value: 'switch', label: 'Switch' },
  { value: 'access_point', label: 'Access Point' },
  { value: 'firewall', label: 'Firewall' },
  { value: 'server', label: 'Server' },
  { value: 'workstation', label: 'Workstation' },
  { value: 'iot', label: 'IoT Device' },
  { value: 'unknown', label: 'Unknown' },
];

// Default empty device
const createEmptyDevice = (): Device => ({
  hostname: '',
  mac: '',
  type: 'switch',
  ip: '',
  ips: [],
});

export const DeviceEditorPage: FC = () => {
  const { hostname } = useParams<{ hostname: string }>();
  const navigate = useNavigate();
  const isNewDevice = hostname === 'new';

  // State
  const [device, setDevice] = useState<Device>(createEmptyDevice());
  const [originalDevice, setOriginalDevice] = useState<Device | null>(null);
  const [saving, setSaving] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);
  const [expandedSections, setExpandedSections] = useState<Set<string>>(new Set(['basic']));
  const [showYamlPreview, setShowYamlPreview] = useState(false);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);

  // Fetch existing device if editing
  const {
    data: fetchedDevice,
    loading,
    error,
    refetch,
  } = useApiResource(
    () => (isNewDevice ? Promise.resolve({ device: createEmptyDevice() }) : fetchConfigDevice(hostname!)),
    [hostname, isNewDevice]
  );

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
    if (isNewDevice) return device.hostname.trim() !== '';
    if (!originalDevice) return false;
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

  // Handle nested protocol config changes
  const updateProtocol = useCallback(<K extends keyof Device>(
    protocol: K,
    config: Device[K]
  ) => {
    setDevice((prev) => ({ ...prev, [protocol]: config }));
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
        await updateDevice(hostname!, device);
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
      setMessage({ type: 'error', text: (err as Error).message });
    } finally {
      setSaving(false);
    }
  }, [device, hostname, isNewDevice, navigate]);

  // Handle delete
  const handleDelete = useCallback(async () => {
    if (!hostname || isNewDevice) return;

    setDeleting(true);
    try {
      await deleteDevice(hostname);
      navigate('/device-config');
    } catch (err) {
      setMessage({ type: 'error', text: (err as Error).message });
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

  // Generate YAML preview
  const yamlPreview = useMemo(() => {
    try {
      // Simple YAML generation (in production, use a proper YAML library)
      const lines: string[] = ['devices:'];
      lines.push(`  - hostname: "${device.hostname}"`);
      lines.push(`    mac: "${device.mac}"`);
      if (device.type) lines.push(`    type: ${device.type}`);
      if (device.ip) lines.push(`    ip: "${device.ip}"`);
      if (device.ips && device.ips.length > 0) {
        lines.push('    ips:');
        device.ips.forEach((ip) => lines.push(`      - "${ip}"`));
      }
      if (device.snmp_agent) {
        lines.push('    snmp_agent:');
        if (device.snmp_agent.community) lines.push(`      community: "${device.snmp_agent.community}"`);
        if (device.snmp_agent.sys_name) lines.push(`      sys_name: "${device.snmp_agent.sys_name}"`);
        if (device.snmp_agent.walk_file) lines.push(`      walk_file: "${device.snmp_agent.walk_file}"`);
      }
      if (device.lldp?.enabled) {
        lines.push('    lldp:');
        lines.push('      enabled: true');
        if (device.lldp.system_description) lines.push(`      system_description: "${device.lldp.system_description}"`);
      }
      if (device.cdp?.enabled) {
        lines.push('    cdp:');
        lines.push('      enabled: true');
        if (device.cdp.platform) lines.push(`      platform: "${device.cdp.platform}"`);
      }
      return lines.join('\n');
    } catch {
      return '# Error generating YAML preview';
    }
  }, [device]);

  // Loading state
  if (!isNewDevice && loading) {
    return (
      <div className="space-y-6">
        <Card className="border-white/5 bg-gray-900/70">
          <CardContent className="flex items-center justify-center py-12">
            <div className="flex items-center gap-3 text-gray-400">
              <div className="h-5 w-5 animate-spin rounded-full border-2 border-violet-500 border-t-transparent" />
              <span>Loading device...</span>
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  // Error state
  if (!isNewDevice && error) {
    return (
      <div className="space-y-6">
        <Card className="border-red-500/30 bg-red-900/20">
          <CardContent className="space-y-3">
            <div className="flex items-start gap-3">
              <AlertCircle className="mt-1 h-5 w-5 text-red-400" />
              <div>
                <p className="font-semibold text-red-200">Failed to Load Device</p>
                <SmallText className="text-red-300/90">{error.message}</SmallText>
                <div className="flex gap-2 mt-3">
                  <Button variant="outline" size="sm" onClick={() => refetch()}>
                    Retry
                  </Button>
                  <Button variant="outline" size="sm" onClick={() => navigate('/device-config')}>
                    Back to List
                  </Button>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  const DeviceIcon = deviceTypeIcons[device.type ?? 'unknown'];

  return (
    <div className="space-y-6">
      {/* Header */}
      <Card className="border-white/5 bg-gray-900/70">
        <CardContent className="space-y-4">
          <div className="flex flex-wrap items-center justify-between gap-4">
            <div className="flex items-center gap-3">
              <button
                onClick={() => navigate('/device-config')}
                className="p-2 text-gray-400 hover:text-white hover:bg-white/10 rounded-lg transition-colors"
                title="Back to device list"
              >
                <ArrowLeft className="h-5 w-5" />
              </button>
              <DeviceIcon className="h-6 w-6 text-violet-300" />
              <div>
                <H2 className="mb-0">
                  {isNewDevice ? 'New Device' : device.hostname || 'Edit Device'}
                </H2>
                <SmallText className="text-gray-400">
                  {isNewDevice ? 'Create a new network device' : 'Edit device configuration'}
                </SmallText>
              </div>
              {isDirty && (
                <Tag colorScheme="yellow" className="ml-2">Unsaved Changes</Tag>
              )}
            </div>
            <div className="flex gap-2">
              <Button
                variant="outline"
                onClick={() => setShowYamlPreview(!showYamlPreview)}
              >
                {showYamlPreview ? 'Hide YAML' : 'Show YAML'}
              </Button>
              {!isNewDevice && (
                <Button
                  variant="outline"
                  leftIcon={<Trash2 className="h-4 w-4" />}
                  onClick={() => setShowDeleteConfirm(true)}
                  className="text-red-400 hover:text-red-300 border-red-400/30 hover:border-red-400/50"
                  disabled={deleting}
                >
                  Delete
                </Button>
              )}
              <Button
                variant="outline"
                onClick={handleDiscard}
                disabled={!isDirty || saving}
              >
                Discard
              </Button>
              <Button
                tone="violet"
                leftIcon={saving ? <RefreshCw className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}
                onClick={handleSave}
                disabled={!isDirty || saving}
              >
                {saving ? 'Saving...' : isNewDevice ? 'Create' : 'Save'}
              </Button>
            </div>
          </div>

          {/* Status message */}
          {message && (
            <div
              className={`flex items-center gap-2 rounded-lg p-3 ${
                message.type === 'success'
                  ? 'border border-green-500/30 bg-green-500/10 text-green-300'
                  : 'border border-red-500/30 bg-red-500/10 text-red-300'
              }`}
              role="alert"
            >
              {message.type === 'success' ? <Check className="h-4 w-4" /> : <AlertCircle className="h-4 w-4" />}
              <span>{message.text}</span>
            </div>
          )}
        </CardContent>
      </Card>

      {/* YAML Preview */}
      {showYamlPreview && (
        <Card className="border-white/5 bg-gray-900/70">
          <CardContent>
            <H2 className="mb-3 flex items-center gap-2 text-base">
              <span>YAML Preview</span>
              <Tag colorScheme="gray">Read-only</Tag>
            </H2>
            <YamlEditor
              value={yamlPreview}
              readOnly
              height="auto"
              minHeight="150px"
              maxHeight="300px"
            />
          </CardContent>
        </Card>
      )}

      {/* Basic Settings */}
      <CollapsibleSection
        title="Basic Settings"
        isExpanded={expandedSections.has('basic')}
        onToggle={() => toggleSection('basic')}
        required
      >
        <div className="grid gap-4 md:grid-cols-2">
          <FormField
            label="Hostname"
            required
            helpText="Unique identifier for the device"
          >
            <input
              type="text"
              value={device.hostname}
              onChange={(e) => updateField('hostname', e.target.value)}
              placeholder="e.g., core-switch-01"
              className="w-full rounded-lg border border-white/10 bg-gray-950/60 p-3 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none"
            />
          </FormField>

          <FormField
            label="MAC Address"
            required
            helpText="Hardware address in format XX:XX:XX:XX:XX:XX"
          >
            <input
              type="text"
              value={device.mac}
              onChange={(e) => updateField('mac', e.target.value.toUpperCase())}
              placeholder="e.g., 00:1A:2B:3C:4D:5E"
              className="w-full rounded-lg border border-white/10 bg-gray-950/60 p-3 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none font-mono"
            />
          </FormField>

          <FormField
            label="Device Type"
            helpText="Category of network device"
          >
            <select
              value={device.type || 'unknown'}
              onChange={(e) => updateField('type', e.target.value as DeviceType)}
              className="w-full rounded-lg border border-white/10 bg-gray-950/60 p-3 text-sm text-white focus:border-violet-400 focus:outline-none"
            >
              {deviceTypes.map((type) => (
                <option key={type.value} value={type.value}>
                  {type.label}
                </option>
              ))}
            </select>
          </FormField>

          <FormField
            label="Primary IP Address"
            helpText="Main management IP address"
          >
            <input
              type="text"
              value={device.ip || ''}
              onChange={(e) => updateField('ip', e.target.value)}
              placeholder="e.g., 192.168.1.1"
              className="w-full rounded-lg border border-white/10 bg-gray-950/60 p-3 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none font-mono"
            />
          </FormField>
        </div>
      </CollapsibleSection>

      {/* SNMP Agent */}
      <CollapsibleSection
        title="SNMP Agent"
        isExpanded={expandedSections.has('snmp')}
        onToggle={() => toggleSection('snmp')}
        enabled={!!device.snmp_agent}
        onEnableChange={(enabled) => {
          if (enabled) {
            updateProtocol('snmp_agent', { community: 'public', sys_name: device.hostname } as SNMPAgent);
          } else {
            updateProtocol('snmp_agent', undefined);
          }
        }}
      >
        {device.snmp_agent && (
          <div className="grid gap-4 md:grid-cols-2">
            <FormField label="Community String" helpText="SNMP v1/v2c community string">
              <input
                type="text"
                value={device.snmp_agent.community || ''}
                onChange={(e) =>
                  updateProtocol('snmp_agent', { ...device.snmp_agent!, community: e.target.value })
                }
                placeholder="public"
                className="w-full rounded-lg border border-white/10 bg-gray-950/60 p-3 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none"
              />
            </FormField>

            <FormField label="System Name" helpText="SNMP sysName value">
              <input
                type="text"
                value={device.snmp_agent.sys_name || ''}
                onChange={(e) =>
                  updateProtocol('snmp_agent', { ...device.snmp_agent!, sys_name: e.target.value })
                }
                placeholder="Device hostname"
                className="w-full rounded-lg border border-white/10 bg-gray-950/60 p-3 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none"
              />
            </FormField>

            <FormField label="Walk File" helpText="Path to SNMP walk file for emulation" className="md:col-span-2">
              <select
                value={device.snmp_agent.walk_file || ''}
                onChange={(e) =>
                  updateProtocol('snmp_agent', { ...device.snmp_agent!, walk_file: e.target.value })
                }
                className="w-full rounded-lg border border-white/10 bg-gray-950/60 p-3 text-sm text-white focus:border-violet-400 focus:outline-none"
              >
                <option value="">Select a walk file...</option>
                {walkFiles?.map((file: FileEntry) => (
                  <option key={file.path} value={file.path}>
                    {file.name}
                  </option>
                ))}
              </select>
            </FormField>
          </div>
        )}
      </CollapsibleSection>

      {/* LLDP */}
      <CollapsibleSection
        title="LLDP (Link Layer Discovery Protocol)"
        isExpanded={expandedSections.has('lldp')}
        onToggle={() => toggleSection('lldp')}
        enabled={device.lldp?.enabled ?? false}
        onEnableChange={(enabled) => {
          updateProtocol('lldp', enabled ? { enabled: true } as LLDPConfig : undefined);
        }}
      >
        {device.lldp?.enabled && (
          <div className="grid gap-4 md:grid-cols-2">
            <FormField label="System Description" helpText="LLDP system description">
              <input
                type="text"
                value={device.lldp.system_description || ''}
                onChange={(e) =>
                  updateProtocol('lldp', { ...device.lldp!, system_description: e.target.value })
                }
                placeholder="Cisco IOS Software..."
                className="w-full rounded-lg border border-white/10 bg-gray-950/60 p-3 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none"
              />
            </FormField>

            <FormField label="Port Description" helpText="LLDP port description">
              <input
                type="text"
                value={device.lldp.port_description || ''}
                onChange={(e) =>
                  updateProtocol('lldp', { ...device.lldp!, port_description: e.target.value })
                }
                placeholder="GigabitEthernet0/1"
                className="w-full rounded-lg border border-white/10 bg-gray-950/60 p-3 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none"
              />
            </FormField>

            <FormField label="Advertise Interval (seconds)" helpText="How often to send LLDP frames">
              <input
                type="number"
                value={device.lldp.advertise_interval || 30}
                onChange={(e) =>
                  updateProtocol('lldp', { ...device.lldp!, advertise_interval: parseInt(e.target.value) })
                }
                min={5}
                max={32768}
                className="w-full rounded-lg border border-white/10 bg-gray-950/60 p-3 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none"
              />
            </FormField>

            <FormField label="TTL (seconds)" helpText="Time-to-live for LLDP information">
              <input
                type="number"
                value={device.lldp.ttl || 120}
                onChange={(e) =>
                  updateProtocol('lldp', { ...device.lldp!, ttl: parseInt(e.target.value) })
                }
                min={1}
                max={65535}
                className="w-full rounded-lg border border-white/10 bg-gray-950/60 p-3 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none"
              />
            </FormField>
          </div>
        )}
      </CollapsibleSection>

      {/* CDP */}
      <CollapsibleSection
        title="CDP (Cisco Discovery Protocol)"
        isExpanded={expandedSections.has('cdp')}
        onToggle={() => toggleSection('cdp')}
        enabled={device.cdp?.enabled ?? false}
        onEnableChange={(enabled) => {
          updateProtocol('cdp', enabled ? { enabled: true } as CDPConfig : undefined);
        }}
      >
        {device.cdp?.enabled && (
          <div className="grid gap-4 md:grid-cols-2">
            <FormField label="Platform" helpText="Hardware platform">
              <input
                type="text"
                value={device.cdp.platform || ''}
                onChange={(e) =>
                  updateProtocol('cdp', { ...device.cdp!, platform: e.target.value })
                }
                placeholder="cisco WS-C3750-48P"
                className="w-full rounded-lg border border-white/10 bg-gray-950/60 p-3 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none"
              />
            </FormField>

            <FormField label="Port ID" helpText="Local port identifier">
              <input
                type="text"
                value={device.cdp.port_id || ''}
                onChange={(e) =>
                  updateProtocol('cdp', { ...device.cdp!, port_id: e.target.value })
                }
                placeholder="GigabitEthernet0/1"
                className="w-full rounded-lg border border-white/10 bg-gray-950/60 p-3 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none"
              />
            </FormField>

            <FormField label="Software Version" helpText="IOS/NX-OS version">
              <input
                type="text"
                value={device.cdp.software_version || ''}
                onChange={(e) =>
                  updateProtocol('cdp', { ...device.cdp!, software_version: e.target.value })
                }
                placeholder="Cisco IOS Software, Version 15.1(4)M"
                className="w-full rounded-lg border border-white/10 bg-gray-950/60 p-3 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none"
              />
            </FormField>

            <FormField label="Holdtime (seconds)" helpText="How long to hold neighbor information">
              <input
                type="number"
                value={device.cdp.holdtime || 180}
                onChange={(e) =>
                  updateProtocol('cdp', { ...device.cdp!, holdtime: parseInt(e.target.value) })
                }
                min={10}
                max={255}
                className="w-full rounded-lg border border-white/10 bg-gray-950/60 p-3 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none"
              />
            </FormField>
          </div>
        )}
      </CollapsibleSection>

      {/* STP */}
      <CollapsibleSection
        title="STP (Spanning Tree Protocol)"
        isExpanded={expandedSections.has('stp')}
        onToggle={() => toggleSection('stp')}
        enabled={device.stp?.enabled ?? false}
        onEnableChange={(enabled) => {
          updateProtocol('stp', enabled ? { enabled: true, bridge_priority: 32768 } as STPConfig : undefined);
        }}
      >
        {device.stp?.enabled && (
          <div className="grid gap-4 md:grid-cols-2">
            <FormField label="Bridge Priority" helpText="STP bridge priority (0-61440, in steps of 4096)">
              <input
                type="number"
                value={device.stp.bridge_priority ?? 32768}
                onChange={(e) =>
                  updateProtocol('stp', { ...device.stp!, bridge_priority: parseInt(e.target.value) })
                }
                min={0}
                max={61440}
                step={4096}
                className="w-full rounded-lg border border-white/10 bg-gray-950/60 p-3 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none"
              />
            </FormField>
          </div>
        )}
      </CollapsibleSection>

      {/* Delete Confirmation Modal */}
      {showDeleteConfirm && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm"
          onClick={() => setShowDeleteConfirm(false)}
          role="dialog"
          aria-modal="true"
        >
          <div
            className="mx-4 w-full max-w-md rounded-2xl border border-white/10 bg-gray-900/95 shadow-2xl"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="p-6 space-y-4">
              <div className="flex items-center gap-3 text-red-400">
                <Trash2 className="h-6 w-6" />
                <h2 className="text-lg font-semibold">Delete Device</h2>
              </div>
              <p className="text-gray-300">
                Are you sure you want to delete <strong>{device.hostname}</strong>? This action cannot be undone.
              </p>
              <div className="flex justify-end gap-3 pt-2">
                <Button variant="outline" onClick={() => setShowDeleteConfirm(false)}>
                  Cancel
                </Button>
                <Button
                  className="bg-red-600 hover:bg-red-700 text-white"
                  onClick={handleDelete}
                  disabled={deleting}
                >
                  {deleting ? 'Deleting...' : 'Delete'}
                </Button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

// Collapsible Section Component
interface CollapsibleSectionProps {
  title: string;
  children: React.ReactNode;
  isExpanded: boolean;
  onToggle: () => void;
  required?: boolean;
  enabled?: boolean;
  onEnableChange?: (enabled: boolean) => void;
}

const CollapsibleSection: FC<CollapsibleSectionProps> = ({
  title,
  children,
  isExpanded,
  onToggle,
  required,
  enabled,
  onEnableChange,
}) => {
  return (
    <Card className="border-white/5 bg-gray-900/70 overflow-hidden">
      <div
        className="flex items-center justify-between px-6 py-4 cursor-pointer hover:bg-white/5 transition-colors"
        onClick={onToggle}
      >
        <div className="flex items-center gap-3">
          {isExpanded ? (
            <ChevronDown className="h-4 w-4 text-gray-400" />
          ) : (
            <ChevronRight className="h-4 w-4 text-gray-400" />
          )}
          <h3 className="font-semibold text-white">{title}</h3>
          {required && <Tag colorScheme="red" className="text-xs">Required</Tag>}
          {onEnableChange && (
            <div
              className="ml-2"
              onClick={(e) => e.stopPropagation()}
            >
              <label className="relative inline-flex items-center cursor-pointer">
                <input
                  type="checkbox"
                  checked={enabled ?? false}
                  onChange={(e) => onEnableChange(e.target.checked)}
                  className="sr-only peer"
                />
                <div className="w-9 h-5 bg-gray-700 rounded-full peer peer-checked:bg-violet-600 peer-focus:ring-2 peer-focus:ring-violet-500 transition-colors">
                  <div className={`absolute top-0.5 left-0.5 w-4 h-4 bg-white rounded-full transition-transform ${enabled ? 'translate-x-4' : ''}`} />
                </div>
              </label>
            </div>
          )}
        </div>
        {enabled !== undefined && (
          <Tag colorScheme={enabled ? 'green' : 'gray'} className="text-xs">
            {enabled ? 'Enabled' : 'Disabled'}
          </Tag>
        )}
      </div>
      {isExpanded && (
        <CardContent className="pt-0 pb-6 px-6">
          {children}
        </CardContent>
      )}
    </Card>
  );
};

// Form Field Component
interface FormFieldProps {
  label: string;
  children: React.ReactNode;
  helpText?: string;
  required?: boolean;
  className?: string;
}

const FormField: FC<FormFieldProps> = ({
  label,
  children,
  helpText,
  required,
  className = '',
}) => {
  return (
    <div className={className}>
      <label className="flex items-center gap-2 text-sm font-medium text-gray-300 mb-2">
        {label}
        {required && <span className="text-red-400">*</span>}
        {helpText && (
          <span className="relative group">
            <HelpCircle className="h-3.5 w-3.5 text-gray-500 cursor-help" />
            <span className="absolute left-1/2 -translate-x-1/2 bottom-full mb-2 px-3 py-1.5 bg-gray-800 text-gray-200 text-xs rounded-lg opacity-0 group-hover:opacity-100 transition-opacity whitespace-nowrap z-10 pointer-events-none">
              {helpText}
            </span>
          </span>
        )}
      </label>
      {children}
    </div>
  );
};

export default DeviceEditorPage;
