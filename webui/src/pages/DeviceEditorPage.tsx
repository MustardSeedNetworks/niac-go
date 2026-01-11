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
  Plus,
  X,
  Globe,
  Database,
  FileText,
  Folder,
  Network,
  Radio,
} from 'lucide-react';
import {
  Card,
  CardContent,
  Button,
  H2,
  SmallText,
  Tag,
} from '../ui';
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
  DHCPConfig,
  DHCPLease,
  DNSConfig,
  DNSRecord,
  HTTPConfig,
  HTTPEndpoint,
  FTPConfig,
  FTPUser,
  NetBIOSConfig,
  NetBIOSService,
  TrafficConfig,
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
      if (device.stp?.enabled) {
        lines.push('    stp:');
        lines.push('      enabled: true');
        if (device.stp.bridge_priority !== undefined) lines.push(`      bridge_priority: ${device.stp.bridge_priority}`);
      }
      if (device.dhcp) {
        lines.push('    dhcp:');
        if (device.dhcp.subnet_mask) lines.push(`      subnet_mask: "${device.dhcp.subnet_mask}"`);
        if (device.dhcp.router) lines.push(`      router: "${device.dhcp.router}"`);
        if (device.dhcp.domain_name_server) lines.push(`      domain_name_server: "${device.dhcp.domain_name_server}"`);
      }
      if (device.dns) {
        lines.push('    dns:');
        if (device.dns.forward_records?.length) {
          lines.push('      forward_records:');
          device.dns.forward_records.forEach((r) => {
            lines.push(`        - name: "${r.name}"`);
            lines.push(`          ip: "${r.ip}"`);
          });
        }
      }
      if (device.http?.enabled) {
        lines.push('    http:');
        lines.push('      enabled: true');
        if (device.http.server_name) lines.push(`      server_name: "${device.http.server_name}"`);
      }
      if (device.ftp?.enabled) {
        lines.push('    ftp:');
        lines.push('      enabled: true');
        if (device.ftp.welcome_banner) lines.push(`      welcome_banner: "${device.ftp.welcome_banner}"`);
      }
      if (device.netbios?.enabled) {
        lines.push('    netbios:');
        lines.push('      enabled: true');
        if (device.netbios.name) lines.push(`      name: "${device.netbios.name}"`);
        if (device.netbios.workgroup) lines.push(`      workgroup: "${device.netbios.workgroup}"`);
      }
      if (device.traffic?.enabled) {
        lines.push('    traffic:');
        lines.push('      enabled: true');
        if (device.traffic.arp_announcements?.enabled) {
          lines.push('      arp_announcements:');
          lines.push('        enabled: true');
        }
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

      {/* Additional IP Addresses */}
      <CollapsibleSection
        title="Additional IP Addresses"
        isExpanded={expandedSections.has('ips')}
        onToggle={() => toggleSection('ips')}
      >
        <div className="space-y-4">
          <SmallText className="text-gray-400">
            Add secondary IP addresses for multi-homed or VLAN configurations.
          </SmallText>
          {(device.ips || []).map((ip, index) => (
            <div key={index} className="flex gap-2">
              <input
                type="text"
                value={ip}
                onChange={(e) => {
                  const newIps = [...(device.ips || [])];
                  newIps[index] = e.target.value;
                  updateField('ips', newIps);
                }}
                placeholder="e.g., 192.168.2.1"
                className="flex-1 rounded-lg border border-white/10 bg-gray-950/60 p-3 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none font-mono"
              />
              <Button
                variant="ghost"
                tone="red"
                size="sm"
                onClick={() => {
                  const newIps = (device.ips || []).filter((_, i) => i !== index);
                  updateField('ips', newIps);
                }}
              >
                <X className="h-4 w-4" />
              </Button>
            </div>
          ))}
          <Button
            variant="outline"
            size="sm"
            leftIcon={<Plus className="h-4 w-4" />}
            onClick={() => updateField('ips', [...(device.ips || []), ''])}
          >
            Add IP Address
          </Button>
        </div>
      </CollapsibleSection>

      {/* DHCP Server */}
      <CollapsibleSection
        title="DHCP Server"
        isExpanded={expandedSections.has('dhcp')}
        onToggle={() => toggleSection('dhcp')}
        enabled={!!device.dhcp}
        onEnableChange={(enabled) => {
          if (enabled) {
            updateProtocol('dhcp', { subnet_mask: '255.255.255.0' } as DHCPConfig);
          } else {
            updateProtocol('dhcp', undefined);
          }
        }}
      >
        {device.dhcp && (
          <div className="space-y-6">
            <div className="grid gap-4 md:grid-cols-2">
              <FormField label="Subnet Mask" helpText="DHCP subnet mask">
                <input
                  type="text"
                  value={device.dhcp.subnet_mask || ''}
                  onChange={(e) =>
                    updateProtocol('dhcp', { ...device.dhcp!, subnet_mask: e.target.value })
                  }
                  placeholder="255.255.255.0"
                  className="w-full rounded-lg border border-white/10 bg-gray-950/60 p-3 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none font-mono"
                />
              </FormField>

              <FormField label="Default Gateway" helpText="Router/gateway for clients">
                <input
                  type="text"
                  value={device.dhcp.router || ''}
                  onChange={(e) =>
                    updateProtocol('dhcp', { ...device.dhcp!, router: e.target.value })
                  }
                  placeholder="192.168.1.1"
                  className="w-full rounded-lg border border-white/10 bg-gray-950/60 p-3 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none font-mono"
                />
              </FormField>

              <FormField label="DNS Server" helpText="Domain name server for clients">
                <input
                  type="text"
                  value={device.dhcp.domain_name_server || ''}
                  onChange={(e) =>
                    updateProtocol('dhcp', { ...device.dhcp!, domain_name_server: e.target.value })
                  }
                  placeholder="8.8.8.8"
                  className="w-full rounded-lg border border-white/10 bg-gray-950/60 p-3 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none font-mono"
                />
              </FormField>

              <FormField label="Domain Name" helpText="Domain suffix for clients">
                <input
                  type="text"
                  value={device.dhcp.domain_name || ''}
                  onChange={(e) =>
                    updateProtocol('dhcp', { ...device.dhcp!, domain_name: e.target.value })
                  }
                  placeholder="example.local"
                  className="w-full rounded-lg border border-white/10 bg-gray-950/60 p-3 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none"
                />
              </FormField>

              <FormField label="Pool Start" helpText="Start of DHCP address pool">
                <input
                  type="text"
                  value={device.dhcp.pool_start || ''}
                  onChange={(e) =>
                    updateProtocol('dhcp', { ...device.dhcp!, pool_start: e.target.value })
                  }
                  placeholder="192.168.1.100"
                  className="w-full rounded-lg border border-white/10 bg-gray-950/60 p-3 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none font-mono"
                />
              </FormField>

              <FormField label="Pool End" helpText="End of DHCP address pool">
                <input
                  type="text"
                  value={device.dhcp.pool_end || ''}
                  onChange={(e) =>
                    updateProtocol('dhcp', { ...device.dhcp!, pool_end: e.target.value })
                  }
                  placeholder="192.168.1.200"
                  className="w-full rounded-lg border border-white/10 bg-gray-950/60 p-3 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none font-mono"
                />
              </FormField>
            </div>

            {/* Static Leases */}
            <div className="space-y-3">
              <h4 className="text-sm font-medium text-white flex items-center gap-2">
                <Database className="h-4 w-4 text-violet-400" />
                Static Leases
              </h4>
              {(device.dhcp.client_leases || []).map((lease, index) => (
                <div key={index} className="flex gap-2 items-center">
                  <input
                    type="text"
                    value={lease.mac_address || ''}
                    onChange={(e) => {
                      const leases = [...(device.dhcp!.client_leases || [])];
                      leases[index] = { ...leases[index], mac_address: e.target.value };
                      updateProtocol('dhcp', { ...device.dhcp!, client_leases: leases });
                    }}
                    placeholder="MAC Address"
                    className="flex-1 rounded-lg border border-white/10 bg-gray-950/60 p-2 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none font-mono"
                  />
                  <input
                    type="text"
                    value={lease.client_ip || ''}
                    onChange={(e) => {
                      const leases = [...(device.dhcp!.client_leases || [])];
                      leases[index] = { ...leases[index], client_ip: e.target.value };
                      updateProtocol('dhcp', { ...device.dhcp!, client_leases: leases });
                    }}
                    placeholder="IP Address"
                    className="flex-1 rounded-lg border border-white/10 bg-gray-950/60 p-2 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none font-mono"
                  />
                  <Button
                    variant="ghost"
                    tone="red"
                    size="sm"
                    onClick={() => {
                      const leases = (device.dhcp!.client_leases || []).filter((_, i) => i !== index);
                      updateProtocol('dhcp', { ...device.dhcp!, client_leases: leases });
                    }}
                  >
                    <X className="h-4 w-4" />
                  </Button>
                </div>
              ))}
              <Button
                variant="outline"
                size="sm"
                leftIcon={<Plus className="h-4 w-4" />}
                onClick={() => {
                  const leases = [...(device.dhcp!.client_leases || []), { mac_address: '', client_ip: '' } as DHCPLease];
                  updateProtocol('dhcp', { ...device.dhcp!, client_leases: leases });
                }}
              >
                Add Static Lease
              </Button>
            </div>
          </div>
        )}
      </CollapsibleSection>

      {/* DNS Server */}
      <CollapsibleSection
        title="DNS Server"
        isExpanded={expandedSections.has('dns')}
        onToggle={() => toggleSection('dns')}
        enabled={!!device.dns}
        onEnableChange={(enabled) => {
          if (enabled) {
            updateProtocol('dns', { forward_records: [], reverse_records: [] } as DNSConfig);
          } else {
            updateProtocol('dns', undefined);
          }
        }}
      >
        {device.dns && (
          <div className="space-y-6">
            {/* Forward Records (A records) */}
            <div className="space-y-3">
              <h4 className="text-sm font-medium text-white flex items-center gap-2">
                <Globe className="h-4 w-4 text-violet-400" />
                Forward Records (A Records)
              </h4>
              {(device.dns.forward_records || []).map((record, index) => (
                <div key={index} className="flex gap-2 items-center">
                  <input
                    type="text"
                    value={record.name || ''}
                    onChange={(e) => {
                      const records = [...(device.dns!.forward_records || [])];
                      records[index] = { ...records[index], name: e.target.value };
                      updateProtocol('dns', { ...device.dns!, forward_records: records });
                    }}
                    placeholder="Hostname (e.g., www.example.com)"
                    className="flex-1 rounded-lg border border-white/10 bg-gray-950/60 p-2 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none"
                  />
                  <input
                    type="text"
                    value={record.ip || ''}
                    onChange={(e) => {
                      const records = [...(device.dns!.forward_records || [])];
                      records[index] = { ...records[index], ip: e.target.value };
                      updateProtocol('dns', { ...device.dns!, forward_records: records });
                    }}
                    placeholder="IP Address"
                    className="w-40 rounded-lg border border-white/10 bg-gray-950/60 p-2 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none font-mono"
                  />
                  <input
                    type="number"
                    value={record.ttl ?? 300}
                    onChange={(e) => {
                      const records = [...(device.dns!.forward_records || [])];
                      records[index] = { ...records[index], ttl: parseInt(e.target.value) };
                      updateProtocol('dns', { ...device.dns!, forward_records: records });
                    }}
                    placeholder="TTL"
                    className="w-24 rounded-lg border border-white/10 bg-gray-950/60 p-2 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none"
                  />
                  <Button
                    variant="ghost"
                    tone="red"
                    size="sm"
                    onClick={() => {
                      const records = (device.dns!.forward_records || []).filter((_, i) => i !== index);
                      updateProtocol('dns', { ...device.dns!, forward_records: records });
                    }}
                  >
                    <X className="h-4 w-4" />
                  </Button>
                </div>
              ))}
              <Button
                variant="outline"
                size="sm"
                leftIcon={<Plus className="h-4 w-4" />}
                onClick={() => {
                  const records = [...(device.dns!.forward_records || []), { name: '', ip: '', ttl: 300 } as DNSRecord];
                  updateProtocol('dns', { ...device.dns!, forward_records: records });
                }}
              >
                Add Forward Record
              </Button>
            </div>

            {/* Reverse Records (PTR records) */}
            <div className="space-y-3">
              <h4 className="text-sm font-medium text-white flex items-center gap-2">
                <Globe className="h-4 w-4 text-violet-400" />
                Reverse Records (PTR Records)
              </h4>
              {(device.dns.reverse_records || []).map((record, index) => (
                <div key={index} className="flex gap-2 items-center">
                  <input
                    type="text"
                    value={record.ip || ''}
                    onChange={(e) => {
                      const records = [...(device.dns!.reverse_records || [])];
                      records[index] = { ...records[index], ip: e.target.value };
                      updateProtocol('dns', { ...device.dns!, reverse_records: records });
                    }}
                    placeholder="IP Address"
                    className="w-40 rounded-lg border border-white/10 bg-gray-950/60 p-2 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none font-mono"
                  />
                  <input
                    type="text"
                    value={record.name || ''}
                    onChange={(e) => {
                      const records = [...(device.dns!.reverse_records || [])];
                      records[index] = { ...records[index], name: e.target.value };
                      updateProtocol('dns', { ...device.dns!, reverse_records: records });
                    }}
                    placeholder="Hostname (e.g., server.example.com)"
                    className="flex-1 rounded-lg border border-white/10 bg-gray-950/60 p-2 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none"
                  />
                  <Button
                    variant="ghost"
                    tone="red"
                    size="sm"
                    onClick={() => {
                      const records = (device.dns!.reverse_records || []).filter((_, i) => i !== index);
                      updateProtocol('dns', { ...device.dns!, reverse_records: records });
                    }}
                  >
                    <X className="h-4 w-4" />
                  </Button>
                </div>
              ))}
              <Button
                variant="outline"
                size="sm"
                leftIcon={<Plus className="h-4 w-4" />}
                onClick={() => {
                  const records = [...(device.dns!.reverse_records || []), { ip: '', name: '', ttl: 300 } as DNSRecord];
                  updateProtocol('dns', { ...device.dns!, reverse_records: records });
                }}
              >
                Add Reverse Record
              </Button>
            </div>
          </div>
        )}
      </CollapsibleSection>

      {/* HTTP Server */}
      <CollapsibleSection
        title="HTTP Server"
        isExpanded={expandedSections.has('http')}
        onToggle={() => toggleSection('http')}
        enabled={device.http?.enabled ?? false}
        onEnableChange={(enabled) => {
          updateProtocol('http', enabled ? { enabled: true, server_name: 'NIAC/1.0' } as HTTPConfig : undefined);
        }}
      >
        {device.http?.enabled && (
          <div className="space-y-6">
            <div className="grid gap-4 md:grid-cols-2">
              <FormField label="Server Name" helpText="HTTP Server header value">
                <input
                  type="text"
                  value={device.http.server_name || ''}
                  onChange={(e) =>
                    updateProtocol('http', { ...device.http!, server_name: e.target.value })
                  }
                  placeholder="Apache/2.4.41"
                  className="w-full rounded-lg border border-white/10 bg-gray-950/60 p-3 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none"
                />
              </FormField>
            </div>

            {/* Endpoints */}
            <div className="space-y-3">
              <h4 className="text-sm font-medium text-white flex items-center gap-2">
                <FileText className="h-4 w-4 text-violet-400" />
                Endpoints
              </h4>
              {(device.http.endpoints || []).map((endpoint, index) => (
                <div key={index} className="rounded-lg border border-white/5 bg-gray-950/40 p-4 space-y-3">
                  <div className="flex gap-2 items-center">
                    <select
                      value={endpoint.method || 'GET'}
                      onChange={(e) => {
                        const endpoints = [...(device.http!.endpoints || [])];
                        endpoints[index] = { ...endpoints[index], method: e.target.value as HTTPEndpoint['method'] };
                        updateProtocol('http', { ...device.http!, endpoints });
                      }}
                      className="w-24 rounded-lg border border-white/10 bg-gray-950/60 p-2 text-sm text-white focus:border-violet-400 focus:outline-none"
                    >
                      <option value="GET">GET</option>
                      <option value="POST">POST</option>
                      <option value="PUT">PUT</option>
                      <option value="DELETE">DELETE</option>
                    </select>
                    <input
                      type="text"
                      value={endpoint.path || ''}
                      onChange={(e) => {
                        const endpoints = [...(device.http!.endpoints || [])];
                        endpoints[index] = { ...endpoints[index], path: e.target.value };
                        updateProtocol('http', { ...device.http!, endpoints });
                      }}
                      placeholder="/api/status"
                      className="flex-1 rounded-lg border border-white/10 bg-gray-950/60 p-2 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none font-mono"
                    />
                    <input
                      type="number"
                      value={endpoint.status_code ?? 200}
                      onChange={(e) => {
                        const endpoints = [...(device.http!.endpoints || [])];
                        endpoints[index] = { ...endpoints[index], status_code: parseInt(e.target.value) };
                        updateProtocol('http', { ...device.http!, endpoints });
                      }}
                      placeholder="Status"
                      className="w-20 rounded-lg border border-white/10 bg-gray-950/60 p-2 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none"
                    />
                    <Button
                      variant="ghost"
                      tone="red"
                      size="sm"
                      onClick={() => {
                        const endpoints = (device.http!.endpoints || []).filter((_, i) => i !== index);
                        updateProtocol('http', { ...device.http!, endpoints });
                      }}
                    >
                      <X className="h-4 w-4" />
                    </Button>
                  </div>
                  <div className="flex gap-2">
                    <input
                      type="text"
                      value={endpoint.content_type || ''}
                      onChange={(e) => {
                        const endpoints = [...(device.http!.endpoints || [])];
                        endpoints[index] = { ...endpoints[index], content_type: e.target.value };
                        updateProtocol('http', { ...device.http!, endpoints });
                      }}
                      placeholder="Content-Type (e.g., application/json)"
                      className="w-64 rounded-lg border border-white/10 bg-gray-950/60 p-2 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none"
                    />
                    <input
                      type="text"
                      value={endpoint.body || ''}
                      onChange={(e) => {
                        const endpoints = [...(device.http!.endpoints || [])];
                        endpoints[index] = { ...endpoints[index], body: e.target.value };
                        updateProtocol('http', { ...device.http!, endpoints });
                      }}
                      placeholder="Response body"
                      className="flex-1 rounded-lg border border-white/10 bg-gray-950/60 p-2 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none"
                    />
                  </div>
                </div>
              ))}
              <Button
                variant="outline"
                size="sm"
                leftIcon={<Plus className="h-4 w-4" />}
                onClick={() => {
                  const endpoints = [...(device.http!.endpoints || []), { path: '/', method: 'GET', status_code: 200, content_type: 'text/html' } as HTTPEndpoint];
                  updateProtocol('http', { ...device.http!, endpoints });
                }}
              >
                Add Endpoint
              </Button>
            </div>
          </div>
        )}
      </CollapsibleSection>

      {/* FTP Server */}
      <CollapsibleSection
        title="FTP Server"
        isExpanded={expandedSections.has('ftp')}
        onToggle={() => toggleSection('ftp')}
        enabled={device.ftp?.enabled ?? false}
        onEnableChange={(enabled) => {
          updateProtocol('ftp', enabled ? { enabled: true, system_type: 'UNIX Type: L8' } as FTPConfig : undefined);
        }}
      >
        {device.ftp?.enabled && (
          <div className="space-y-6">
            <div className="grid gap-4 md:grid-cols-2">
              <FormField label="Welcome Banner" helpText="FTP welcome message">
                <input
                  type="text"
                  value={device.ftp.welcome_banner || ''}
                  onChange={(e) =>
                    updateProtocol('ftp', { ...device.ftp!, welcome_banner: e.target.value })
                  }
                  placeholder="Welcome to FTP Server"
                  className="w-full rounded-lg border border-white/10 bg-gray-950/60 p-3 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none"
                />
              </FormField>

              <FormField label="System Type" helpText="SYST response">
                <input
                  type="text"
                  value={device.ftp.system_type || ''}
                  onChange={(e) =>
                    updateProtocol('ftp', { ...device.ftp!, system_type: e.target.value })
                  }
                  placeholder="UNIX Type: L8"
                  className="w-full rounded-lg border border-white/10 bg-gray-950/60 p-3 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none"
                />
              </FormField>

              <FormField label="Allow Anonymous" helpText="Allow anonymous FTP access">
                <label className="relative inline-flex items-center cursor-pointer mt-2">
                  <input
                    type="checkbox"
                    checked={device.ftp.allow_anonymous ?? false}
                    onChange={(e) =>
                      updateProtocol('ftp', { ...device.ftp!, allow_anonymous: e.target.checked })
                    }
                    className="sr-only peer"
                  />
                  <div className="w-9 h-5 bg-gray-700 rounded-full peer peer-checked:bg-violet-600 peer-focus:ring-2 peer-focus:ring-violet-500 transition-colors">
                    <div className={`absolute top-0.5 left-0.5 w-4 h-4 bg-white rounded-full transition-transform ${device.ftp.allow_anonymous ? 'translate-x-4' : ''}`} />
                  </div>
                  <span className="ml-3 text-sm text-gray-300">{device.ftp.allow_anonymous ? 'Enabled' : 'Disabled'}</span>
                </label>
              </FormField>
            </div>

            {/* FTP Users */}
            <div className="space-y-3">
              <h4 className="text-sm font-medium text-white flex items-center gap-2">
                <Folder className="h-4 w-4 text-violet-400" />
                FTP Users
              </h4>
              {(device.ftp.users || []).map((user, index) => (
                <div key={index} className="flex gap-2 items-center">
                  <input
                    type="text"
                    value={user.username || ''}
                    onChange={(e) => {
                      const users = [...(device.ftp!.users || [])];
                      users[index] = { ...users[index], username: e.target.value };
                      updateProtocol('ftp', { ...device.ftp!, users });
                    }}
                    placeholder="Username"
                    className="flex-1 rounded-lg border border-white/10 bg-gray-950/60 p-2 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none"
                  />
                  <input
                    type="text"
                    value={user.password || ''}
                    onChange={(e) => {
                      const users = [...(device.ftp!.users || [])];
                      users[index] = { ...users[index], password: e.target.value };
                      updateProtocol('ftp', { ...device.ftp!, users });
                    }}
                    placeholder="Password"
                    className="flex-1 rounded-lg border border-white/10 bg-gray-950/60 p-2 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none"
                  />
                  <input
                    type="text"
                    value={user.home_dir || ''}
                    onChange={(e) => {
                      const users = [...(device.ftp!.users || [])];
                      users[index] = { ...users[index], home_dir: e.target.value };
                      updateProtocol('ftp', { ...device.ftp!, users });
                    }}
                    placeholder="Home Directory"
                    className="flex-1 rounded-lg border border-white/10 bg-gray-950/60 p-2 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none font-mono"
                  />
                  <Button
                    variant="ghost"
                    tone="red"
                    size="sm"
                    onClick={() => {
                      const users = (device.ftp!.users || []).filter((_, i) => i !== index);
                      updateProtocol('ftp', { ...device.ftp!, users });
                    }}
                  >
                    <X className="h-4 w-4" />
                  </Button>
                </div>
              ))}
              <Button
                variant="outline"
                size="sm"
                leftIcon={<Plus className="h-4 w-4" />}
                onClick={() => {
                  const users = [...(device.ftp!.users || []), { username: '', password: '', home_dir: '/' } as FTPUser];
                  updateProtocol('ftp', { ...device.ftp!, users });
                }}
              >
                Add User
              </Button>
            </div>
          </div>
        )}
      </CollapsibleSection>

      {/* NetBIOS */}
      <CollapsibleSection
        title="NetBIOS"
        isExpanded={expandedSections.has('netbios')}
        onToggle={() => toggleSection('netbios')}
        enabled={device.netbios?.enabled ?? false}
        onEnableChange={(enabled) => {
          updateProtocol('netbios', enabled ? { enabled: true, node_type: 'B' } as NetBIOSConfig : undefined);
        }}
      >
        {device.netbios?.enabled && (
          <div className="space-y-6">
            <div className="grid gap-4 md:grid-cols-2">
              <FormField label="NetBIOS Name" helpText="NetBIOS computer name (max 15 chars)">
                <input
                  type="text"
                  value={device.netbios.name || ''}
                  onChange={(e) =>
                    updateProtocol('netbios', { ...device.netbios!, name: e.target.value.toUpperCase().slice(0, 15) })
                  }
                  placeholder="FILESERVER"
                  maxLength={15}
                  className="w-full rounded-lg border border-white/10 bg-gray-950/60 p-3 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none uppercase"
                />
              </FormField>

              <FormField label="Workgroup" helpText="NetBIOS workgroup/domain">
                <input
                  type="text"
                  value={device.netbios.workgroup || ''}
                  onChange={(e) =>
                    updateProtocol('netbios', { ...device.netbios!, workgroup: e.target.value.toUpperCase() })
                  }
                  placeholder="WORKGROUP"
                  className="w-full rounded-lg border border-white/10 bg-gray-950/60 p-3 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none uppercase"
                />
              </FormField>

              <FormField label="Node Type" helpText="NetBIOS node type">
                <select
                  value={device.netbios.node_type || 'B'}
                  onChange={(e) =>
                    updateProtocol('netbios', { ...device.netbios!, node_type: e.target.value as NetBIOSConfig['node_type'] })
                  }
                  className="w-full rounded-lg border border-white/10 bg-gray-950/60 p-3 text-sm text-white focus:border-violet-400 focus:outline-none"
                >
                  <option value="B">B-node (Broadcast)</option>
                  <option value="P">P-node (Point-to-Point)</option>
                  <option value="M">M-node (Mixed)</option>
                  <option value="H">H-node (Hybrid)</option>
                </select>
              </FormField>

              <FormField label="TTL (seconds)" helpText="Time-to-live for NetBIOS announcements">
                <input
                  type="number"
                  value={device.netbios.ttl ?? 300000}
                  onChange={(e) =>
                    updateProtocol('netbios', { ...device.netbios!, ttl: parseInt(e.target.value) })
                  }
                  min={60000}
                  max={604800000}
                  className="w-full rounded-lg border border-white/10 bg-gray-950/60 p-3 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none"
                />
              </FormField>
            </div>

            {/* Services */}
            <div className="space-y-3">
              <h4 className="text-sm font-medium text-white flex items-center gap-2">
                <Network className="h-4 w-4 text-violet-400" />
                NetBIOS Services
              </h4>
              <div className="flex flex-wrap gap-4">
                {(['workstation', 'fileserver', 'messenger'] as NetBIOSService[]).map((service) => (
                  <label key={service} className="flex items-center gap-2 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={(device.netbios!.services || []).includes(service)}
                      onChange={(e) => {
                        const services = device.netbios!.services || [];
                        if (e.target.checked) {
                          updateProtocol('netbios', { ...device.netbios!, services: [...services, service] });
                        } else {
                          updateProtocol('netbios', { ...device.netbios!, services: services.filter(s => s !== service) });
                        }
                      }}
                      className="w-4 h-4 rounded border-gray-600 bg-gray-800 text-violet-600 focus:ring-violet-500"
                    />
                    <span className="text-sm text-gray-300 capitalize">{service}</span>
                  </label>
                ))}
              </div>

              <label className="flex items-center gap-2 cursor-pointer mt-2">
                <input
                  type="checkbox"
                  checked={device.netbios.msbrowse ?? false}
                  onChange={(e) =>
                    updateProtocol('netbios', { ...device.netbios!, msbrowse: e.target.checked })
                  }
                  className="w-4 h-4 rounded border-gray-600 bg-gray-800 text-violet-600 focus:ring-violet-500"
                />
                <span className="text-sm text-gray-300">Master Browser (MSBROWSE)</span>
              </label>
            </div>
          </div>
        )}
      </CollapsibleSection>

      {/* Traffic Patterns */}
      <CollapsibleSection
        title="Traffic Patterns"
        isExpanded={expandedSections.has('traffic')}
        onToggle={() => toggleSection('traffic')}
        enabled={device.traffic?.enabled ?? false}
        onEnableChange={(enabled) => {
          updateProtocol('traffic', enabled ? { enabled: true } as TrafficConfig : undefined);
        }}
      >
        {device.traffic?.enabled && (
          <div className="space-y-6">
            {/* ARP Announcements */}
            <div className="rounded-lg border border-white/5 bg-gray-950/40 p-4 space-y-3">
              <div className="flex items-center justify-between">
                <h4 className="text-sm font-medium text-white flex items-center gap-2">
                  <Radio className="h-4 w-4 text-violet-400" />
                  ARP Announcements
                </h4>
                <label className="relative inline-flex items-center cursor-pointer">
                  <input
                    type="checkbox"
                    checked={device.traffic.arp_announcements?.enabled ?? false}
                    onChange={(e) =>
                      updateProtocol('traffic', {
                        ...device.traffic!,
                        arp_announcements: { ...device.traffic!.arp_announcements, enabled: e.target.checked }
                      })
                    }
                    className="sr-only peer"
                  />
                  <div className="w-9 h-5 bg-gray-700 rounded-full peer peer-checked:bg-violet-600 peer-focus:ring-2 peer-focus:ring-violet-500 transition-colors">
                    <div className={`absolute top-0.5 left-0.5 w-4 h-4 bg-white rounded-full transition-transform ${device.traffic.arp_announcements?.enabled ? 'translate-x-4' : ''}`} />
                  </div>
                </label>
              </div>
              {device.traffic.arp_announcements?.enabled && (
                <FormField label="Interval (seconds)" helpText="Time between ARP announcements">
                  <input
                    type="number"
                    value={device.traffic.arp_announcements.interval ?? 60}
                    onChange={(e) =>
                      updateProtocol('traffic', {
                        ...device.traffic!,
                        arp_announcements: { ...device.traffic!.arp_announcements!, interval: parseInt(e.target.value) }
                      })
                    }
                    min={1}
                    className="w-32 rounded-lg border border-white/10 bg-gray-950/60 p-2 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none"
                  />
                </FormField>
              )}
            </div>

            {/* Periodic Pings */}
            <div className="rounded-lg border border-white/5 bg-gray-950/40 p-4 space-y-3">
              <div className="flex items-center justify-between">
                <h4 className="text-sm font-medium text-white flex items-center gap-2">
                  <Radio className="h-4 w-4 text-violet-400" />
                  Periodic Pings
                </h4>
                <label className="relative inline-flex items-center cursor-pointer">
                  <input
                    type="checkbox"
                    checked={device.traffic.periodic_pings?.enabled ?? false}
                    onChange={(e) =>
                      updateProtocol('traffic', {
                        ...device.traffic!,
                        periodic_pings: { ...device.traffic!.periodic_pings, enabled: e.target.checked }
                      })
                    }
                    className="sr-only peer"
                  />
                  <div className="w-9 h-5 bg-gray-700 rounded-full peer peer-checked:bg-violet-600 peer-focus:ring-2 peer-focus:ring-violet-500 transition-colors">
                    <div className={`absolute top-0.5 left-0.5 w-4 h-4 bg-white rounded-full transition-transform ${device.traffic.periodic_pings?.enabled ? 'translate-x-4' : ''}`} />
                  </div>
                </label>
              </div>
              {device.traffic.periodic_pings?.enabled && (
                <div className="grid gap-4 md:grid-cols-2">
                  <FormField label="Interval (seconds)" helpText="Time between pings">
                    <input
                      type="number"
                      value={device.traffic.periodic_pings.interval ?? 30}
                      onChange={(e) =>
                        updateProtocol('traffic', {
                          ...device.traffic!,
                          periodic_pings: { ...device.traffic!.periodic_pings!, interval: parseInt(e.target.value) }
                        })
                      }
                      min={1}
                      className="w-full rounded-lg border border-white/10 bg-gray-950/60 p-2 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none"
                    />
                  </FormField>
                  <FormField label="Payload Size (bytes)" helpText="Size of ICMP payload">
                    <input
                      type="number"
                      value={device.traffic.periodic_pings.payload_size ?? 56}
                      onChange={(e) =>
                        updateProtocol('traffic', {
                          ...device.traffic!,
                          periodic_pings: { ...device.traffic!.periodic_pings!, payload_size: parseInt(e.target.value) }
                        })
                      }
                      min={0}
                      max={65507}
                      className="w-full rounded-lg border border-white/10 bg-gray-950/60 p-2 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none"
                    />
                  </FormField>
                </div>
              )}
            </div>

            {/* Random Traffic */}
            <div className="rounded-lg border border-white/5 bg-gray-950/40 p-4 space-y-3">
              <div className="flex items-center justify-between">
                <h4 className="text-sm font-medium text-white flex items-center gap-2">
                  <Radio className="h-4 w-4 text-violet-400" />
                  Random Traffic
                </h4>
                <label className="relative inline-flex items-center cursor-pointer">
                  <input
                    type="checkbox"
                    checked={device.traffic.random_traffic?.enabled ?? false}
                    onChange={(e) =>
                      updateProtocol('traffic', {
                        ...device.traffic!,
                        random_traffic: { ...device.traffic!.random_traffic, enabled: e.target.checked }
                      })
                    }
                    className="sr-only peer"
                  />
                  <div className="w-9 h-5 bg-gray-700 rounded-full peer peer-checked:bg-violet-600 peer-focus:ring-2 peer-focus:ring-violet-500 transition-colors">
                    <div className={`absolute top-0.5 left-0.5 w-4 h-4 bg-white rounded-full transition-transform ${device.traffic.random_traffic?.enabled ? 'translate-x-4' : ''}`} />
                  </div>
                </label>
              </div>
              {device.traffic.random_traffic?.enabled && (
                <div className="space-y-4">
                  <div className="grid gap-4 md:grid-cols-2">
                    <FormField label="Interval (seconds)" helpText="Time between traffic bursts">
                      <input
                        type="number"
                        value={device.traffic.random_traffic.interval ?? 60}
                        onChange={(e) =>
                          updateProtocol('traffic', {
                            ...device.traffic!,
                            random_traffic: { ...device.traffic!.random_traffic!, interval: parseInt(e.target.value) }
                          })
                        }
                        min={1}
                        className="w-full rounded-lg border border-white/10 bg-gray-950/60 p-2 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none"
                      />
                    </FormField>
                    <FormField label="Packet Count" helpText="Packets per burst">
                      <input
                        type="number"
                        value={device.traffic.random_traffic.packet_count ?? 5}
                        onChange={(e) =>
                          updateProtocol('traffic', {
                            ...device.traffic!,
                            random_traffic: { ...device.traffic!.random_traffic!, packet_count: parseInt(e.target.value) }
                          })
                        }
                        min={1}
                        max={100}
                        className="w-full rounded-lg border border-white/10 bg-gray-950/60 p-2 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none"
                      />
                    </FormField>
                  </div>
                  <div className="space-y-2">
                    <h5 className="text-xs font-medium text-gray-400">Traffic Patterns</h5>
                    <div className="flex flex-wrap gap-4">
                      {(['broadcast_arp', 'multicast', 'udp'] as const).map((pattern) => (
                        <label key={pattern} className="flex items-center gap-2 cursor-pointer">
                          <input
                            type="checkbox"
                            checked={(device.traffic!.random_traffic?.patterns || []).includes(pattern)}
                            onChange={(e) => {
                              const patterns = device.traffic!.random_traffic?.patterns || [];
                              if (e.target.checked) {
                                updateProtocol('traffic', {
                                  ...device.traffic!,
                                  random_traffic: { ...device.traffic!.random_traffic!, patterns: [...patterns, pattern] }
                                });
                              } else {
                                updateProtocol('traffic', {
                                  ...device.traffic!,
                                  random_traffic: { ...device.traffic!.random_traffic!, patterns: patterns.filter(p => p !== pattern) }
                                });
                              }
                            }}
                            className="w-4 h-4 rounded border-gray-600 bg-gray-800 text-violet-600 focus:ring-violet-500"
                          />
                          <span className="text-sm text-gray-300">
                            {pattern === 'broadcast_arp' ? 'Broadcast ARP' : pattern === 'multicast' ? 'Multicast' : 'UDP'}
                          </span>
                        </label>
                      ))}
                    </div>
                  </div>
                </div>
              )}
            </div>
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
