import { AlertCircle, ArrowLeft, Check, Plus, RefreshCw, Save, Trash2, X } from "lucide-react";
import { type FC, useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import {
  createDevice,
  deleteDevice,
  fetchConfigDevice,
  fetchWalkFiles,
  updateDevice,
} from "../api/client";
import type { Device, DeviceType } from "../api/types";
import { YamlEditor } from "../components/config/YamlEditor";
import {
  CDPSection,
  DHCPSection,
  DNSSection,
  FTPSection,
  HTTPSection,
  LLDPSection,
  NetBIOSSection,
  SNMPSection,
  STPSection,
  TrafficSection,
} from "../components/device-editor";
import { CollapsibleSection, FormField } from "../components/form";
import { deviceTypeIcons, deviceTypeOptions as deviceTypes } from "../constants/deviceTypes";
import { useApiResource } from "../hooks/useApiResource";
import { Button, Card, CardContent, H2, SmallText, Tag } from "../ui";
import { getErrorMessage } from "../utils";

// Default empty device
const createEmptyDevice = (): Device => ({
  hostname: "",
  mac: "",
  type: "switch",
  ip: "",
  ips: [],
});

export const DeviceEditorPage: FC = () => {
  const { hostname } = useParams<{ hostname: string }>();
  const navigate = useNavigate();
  const isNewDevice = hostname === "new";

  // State
  const [device, setDevice] = useState<Device>(createEmptyDevice());
  const [originalDevice, setOriginalDevice] = useState<Device | null>(null);
  const [saving, setSaving] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [message, setMessage] = useState<{ type: "success" | "error"; text: string } | null>(null);
  const [expandedSections, setExpandedSections] = useState<Set<string>>(new Set(["basic"]));
  const [showYamlPreview, setShowYamlPreview] = useState(false);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);

  // Fetch existing device if editing
  const {
    data: fetchedDevice,
    loading,
    error,
    refetch,
  } = useApiResource(
    () =>
      isNewDevice ? Promise.resolve({ device: createEmptyDevice() }) : fetchConfigDevice(hostname!),
    [hostname, isNewDevice],
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
    if (isNewDevice) return device.hostname.trim() !== "";
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

  // Handle save
  const handleSave = useCallback(async () => {
    if (!device.hostname.trim()) {
      setMessage({ type: "error", text: "Hostname is required" });
      return;
    }

    if (!device.mac.trim()) {
      setMessage({ type: "error", text: "MAC address is required" });
      return;
    }

    setSaving(true);
    setMessage(null);

    try {
      if (isNewDevice) {
        await createDevice(device);
        setMessage({ type: "success", text: "Device created successfully" });
        // Navigate to the new device's edit page
        setTimeout(() => {
          navigate(`/device-config/${encodeURIComponent(device.hostname)}`);
        }, 500);
      } else {
        await updateDevice(hostname!, device);
        setMessage({ type: "success", text: "Device updated successfully" });
        setOriginalDevice(device);
        // If hostname changed, navigate to new URL
        if (device.hostname !== hostname) {
          setTimeout(() => {
            navigate(`/device-config/${encodeURIComponent(device.hostname)}`);
          }, 500);
        }
      }
    } catch (err) {
      setMessage({ type: "error", text: getErrorMessage(err) });
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
      navigate("/device-config");
    } catch (err) {
      setMessage({ type: "error", text: getErrorMessage(err) });
      setDeleting(false);
    }
  }, [hostname, isNewDevice, navigate]);

  // Handle cancel/discard
  const handleDiscard = useCallback(() => {
    if (isNewDevice) {
      navigate("/device-config");
    } else if (originalDevice) {
      setDevice(originalDevice);
      setMessage(null);
    }
  }, [isNewDevice, originalDevice, navigate]);

  // Generate YAML preview
  const yamlPreview = useMemo(() => {
    try {
      // Simple YAML generation (in production, use a proper YAML library)
      const lines: string[] = ["devices:"];
      lines.push(`  - hostname: "${device.hostname}"`);
      lines.push(`    mac: "${device.mac}"`);
      if (device.type) lines.push(`    type: ${device.type}`);
      if (device.ip) lines.push(`    ip: "${device.ip}"`);
      if (device.ips && device.ips.length > 0) {
        lines.push("    ips:");
        device.ips.forEach((ip) => lines.push(`      - "${ip}"`));
      }
      if (device.snmp_agent) {
        lines.push("    snmp_agent:");
        if (device.snmp_agent.community)
          lines.push(`      community: "${device.snmp_agent.community}"`);
        if (device.snmp_agent.sys_name)
          lines.push(`      sys_name: "${device.snmp_agent.sys_name}"`);
        if (device.snmp_agent.walk_file)
          lines.push(`      walk_file: "${device.snmp_agent.walk_file}"`);
      }
      if (device.lldp?.enabled) {
        lines.push("    lldp:");
        lines.push("      enabled: true");
        if (device.lldp.system_description)
          lines.push(`      system_description: "${device.lldp.system_description}"`);
      }
      if (device.cdp?.enabled) {
        lines.push("    cdp:");
        lines.push("      enabled: true");
        if (device.cdp.platform) lines.push(`      platform: "${device.cdp.platform}"`);
      }
      if (device.stp?.enabled) {
        lines.push("    stp:");
        lines.push("      enabled: true");
        if (device.stp.bridge_priority !== undefined)
          lines.push(`      bridge_priority: ${device.stp.bridge_priority}`);
      }
      if (device.dhcp) {
        lines.push("    dhcp:");
        if (device.dhcp.subnet_mask) lines.push(`      subnet_mask: "${device.dhcp.subnet_mask}"`);
        if (device.dhcp.router) lines.push(`      router: "${device.dhcp.router}"`);
        if (device.dhcp.domain_name_server)
          lines.push(`      domain_name_server: "${device.dhcp.domain_name_server}"`);
      }
      if (device.dns) {
        lines.push("    dns:");
        if (device.dns.forward_records && device.dns.forward_records.length > 0) {
          lines.push("      forward_records:");
          device.dns.forward_records.forEach((r: { name: string; ip: string }) => {
            lines.push(`        - name: "${r.name}"`);
            lines.push(`          ip: "${r.ip}"`);
          });
        }
      }
      if (device.http?.enabled) {
        lines.push("    http:");
        lines.push("      enabled: true");
        if (device.http.server_name) lines.push(`      server_name: "${device.http.server_name}"`);
      }
      if (device.ftp?.enabled) {
        lines.push("    ftp:");
        lines.push("      enabled: true");
        if (device.ftp.welcome_banner)
          lines.push(`      welcome_banner: "${device.ftp.welcome_banner}"`);
      }
      if (device.netbios?.enabled) {
        lines.push("    netbios:");
        lines.push("      enabled: true");
        if (device.netbios.name) lines.push(`      name: "${device.netbios.name}"`);
        if (device.netbios.workgroup) lines.push(`      workgroup: "${device.netbios.workgroup}"`);
      }
      if (device.traffic?.enabled) {
        lines.push("    traffic:");
        lines.push("      enabled: true");
        if (device.traffic.arp_announcements?.enabled) {
          lines.push("      arp_announcements:");
          lines.push("        enabled: true");
        }
      }
      return lines.join("\n");
    } catch {
      return "# Error generating YAML preview";
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
                  <Button variant="outline" size="sm" onClick={() => navigate("/device-config")}>
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

  const DeviceIcon = deviceTypeIcons[device.type ?? "unknown"];

  return (
    <div className="space-y-6">
      {/* Header */}
      <Card className="border-white/5 bg-gray-900/70">
        <CardContent className="space-y-4">
          <div className="flex flex-wrap items-center justify-between gap-4">
            <div className="flex items-center gap-3">
              <button
                onClick={() => navigate("/device-config")}
                className="p-2 text-gray-400 hover:text-white hover:bg-white/10 rounded-lg transition-colors"
                title="Back to device list"
              >
                <ArrowLeft className="h-5 w-5" />
              </button>
              <DeviceIcon className="h-6 w-6 text-violet-300" />
              <div>
                <H2 className="mb-0">
                  {isNewDevice ? "New Device" : device.hostname || "Edit Device"}
                </H2>
                <SmallText className="text-gray-400">
                  {isNewDevice ? "Create a new network device" : "Edit device configuration"}
                </SmallText>
              </div>
              {isDirty && (
                <Tag colorScheme="yellow" className="ml-2">
                  Unsaved Changes
                </Tag>
              )}
            </div>
            <div className="flex gap-2">
              <Button variant="outline" onClick={() => setShowYamlPreview(!showYamlPreview)}>
                {showYamlPreview ? "Hide YAML" : "Show YAML"}
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
              <Button variant="outline" onClick={handleDiscard} disabled={!isDirty || saving}>
                Discard
              </Button>
              <Button
                tone="violet"
                leftIcon={
                  saving ? (
                    <RefreshCw className="h-4 w-4 animate-spin" />
                  ) : (
                    <Save className="h-4 w-4" />
                  )
                }
                onClick={handleSave}
                disabled={!isDirty || saving}
              >
                {saving ? "Saving..." : isNewDevice ? "Create" : "Save"}
              </Button>
            </div>
          </div>

          {/* Status message */}
          {message && (
            <div
              className={`flex items-center gap-2 rounded-lg p-3 ${
                message.type === "success"
                  ? "border border-green-500/30 bg-green-500/10 text-green-300"
                  : "border border-red-500/30 bg-red-500/10 text-red-300"
              }`}
              role="alert"
            >
              {message.type === "success" ? (
                <Check className="h-4 w-4" />
              ) : (
                <AlertCircle className="h-4 w-4" />
              )}
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
        isExpanded={expandedSections.has("basic")}
        onToggle={() => toggleSection("basic")}
        required
      >
        <div className="grid gap-4 md:grid-cols-2">
          <FormField label="Hostname" required helpText="Unique identifier for the device">
            <input
              type="text"
              value={device.hostname}
              onChange={(e) => updateField("hostname", e.target.value)}
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
              onChange={(e) => updateField("mac", e.target.value.toUpperCase())}
              placeholder="e.g., 00:1A:2B:3C:4D:5E"
              className="w-full rounded-lg border border-white/10 bg-gray-950/60 p-3 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none font-mono"
            />
          </FormField>

          <FormField label="Device Type" helpText="Category of network device">
            <select
              value={device.type || "unknown"}
              onChange={(e) => updateField("type", e.target.value as DeviceType)}
              className="w-full rounded-lg border border-white/10 bg-gray-950/60 p-3 text-sm text-white focus:border-violet-400 focus:outline-none"
            >
              {deviceTypes.map((type) => (
                <option key={type.value} value={type.value}>
                  {type.label}
                </option>
              ))}
            </select>
          </FormField>

          <FormField label="Primary IP Address" helpText="Main management IP address">
            <input
              type="text"
              value={device.ip || ""}
              onChange={(e) => updateField("ip", e.target.value)}
              placeholder="e.g., 192.168.1.1"
              className="w-full rounded-lg border border-white/10 bg-gray-950/60 p-3 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none font-mono"
            />
          </FormField>
        </div>
      </CollapsibleSection>

      <SNMPSection
        device={device}
        isExpanded={expandedSections.has("snmp")}
        onToggle={() => toggleSection("snmp")}
        onUpdate={updateField}
        walkFiles={walkFiles}
      />

      <LLDPSection
        device={device}
        isExpanded={expandedSections.has("lldp")}
        onToggle={() => toggleSection("lldp")}
        onUpdate={updateField}
      />

      <CDPSection
        device={device}
        isExpanded={expandedSections.has("cdp")}
        onToggle={() => toggleSection("cdp")}
        onUpdate={updateField}
      />

      <STPSection
        device={device}
        isExpanded={expandedSections.has("stp")}
        onToggle={() => toggleSection("stp")}
        onUpdate={updateField}
      />

      {/* Additional IP Addresses */}
      <CollapsibleSection
        title="Additional IP Addresses"
        isExpanded={expandedSections.has("ips")}
        onToggle={() => toggleSection("ips")}
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
                  updateField("ips", newIps);
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
                  updateField("ips", newIps);
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
            onClick={() => updateField("ips", [...(device.ips || []), ""])}
          >
            Add IP Address
          </Button>
        </div>
      </CollapsibleSection>

      <DHCPSection
        device={device}
        isExpanded={expandedSections.has("dhcp")}
        onToggle={() => toggleSection("dhcp")}
        onUpdate={updateField}
      />

      <DNSSection
        device={device}
        isExpanded={expandedSections.has("dns")}
        onToggle={() => toggleSection("dns")}
        onUpdate={updateField}
      />

      <HTTPSection
        device={device}
        isExpanded={expandedSections.has("http")}
        onToggle={() => toggleSection("http")}
        onUpdate={updateField}
      />

      <FTPSection
        device={device}
        isExpanded={expandedSections.has("ftp")}
        onToggle={() => toggleSection("ftp")}
        onUpdate={updateField}
      />

      <NetBIOSSection
        device={device}
        isExpanded={expandedSections.has("netbios")}
        onToggle={() => toggleSection("netbios")}
        onUpdate={updateField}
      />

      <TrafficSection
        device={device}
        isExpanded={expandedSections.has("traffic")}
        onToggle={() => toggleSection("traffic")}
        onUpdate={updateField}
      />

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
                Are you sure you want to delete <strong>{device.hostname}</strong>? This action
                cannot be undone.
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
                  {deleting ? "Deleting..." : "Delete"}
                </Button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default DeviceEditorPage;
