import {
	AlertCircle,
	ArrowLeft,
	Check,
	Plus,
	RefreshCw,
	Save,
	Trash2,
	X,
} from "lucide-react";
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
import { CdpSection } from "../components/device-editor/CdpSection";
import { DhcpSection } from "../components/device-editor/DhcpSection";
import { DnsSection } from "../components/device-editor/DnsSection";
import { FtpSection } from "../components/device-editor/FtpSection";
import { HttpSection } from "../components/device-editor/HttpSection";
import { LldpSection } from "../components/device-editor/LldpSection";
import { NetBiosSection } from "../components/device-editor/NetBiosSection";
import { SnmpSection } from "../components/device-editor/SnmpSection";
import { StpSection } from "../components/device-editor/StpSection";
import { TrafficSection } from "../components/device-editor/TrafficSection";
import { CollapsibleSection } from "../components/form/CollapsibleSection";
import { FormField } from "../components/form/FormField";
import {
	deviceTypeIcons,
	deviceTypeOptions as deviceTypes,
} from "../constants/device-types";
import { useApiResource } from "../hooks/useApiResource";
import { Button } from "../ui/Button";
import { Card, CardContent } from "../ui/Card";
import { Tag } from "../ui/Tag";
import { H2, SmallText } from "../ui/Typography";
import { getErrorMessage } from "../utils/format";

const appendBaseDeviceLines = (lines: string[], device: Device) => {
	lines.push(`  - hostname: "${device.hostname}"`);
	lines.push(`    mac: "${device.mac}"`);
	if (device.type) {
		lines.push(`    type: ${device.type}`);
	}
	if (device.ip) {
		lines.push(`    ip: "${device.ip}"`);
	}
	if (device.ips && device.ips.length > 0) {
		lines.push("    ips:");
		for (const ip of device.ips) {
			lines.push(`      - "${ip}"`);
		}
	}
};

const appendSnmpLines = (lines: string[], device: Device) => {
	if (!device.snmpAgent) {
		return;
	}
	lines.push("    snmpAgent:");
	if (device.snmpAgent.community) {
		lines.push(`      community: "${device.snmpAgent.community}"`);
	}
	if (device.snmpAgent.sysName) {
		lines.push(`      sysName: "${device.snmpAgent.sysName}"`);
	}
	if (device.snmpAgent.walkFile) {
		lines.push(`      walkFile: "${device.snmpAgent.walkFile}"`);
	}
};

const appendLldpLines = (lines: string[], device: Device) => {
	if (!device.lldp?.enabled) {
		return;
	}
	lines.push("    lldp:");
	lines.push("      enabled: true");
	if (device.lldp.systemDescription) {
		lines.push(`      systemDescription: "${device.lldp.systemDescription}"`);
	}
};

const appendCdpLines = (lines: string[], device: Device) => {
	if (!device.cdp?.enabled) {
		return;
	}
	lines.push("    cdp:");
	lines.push("      enabled: true");
	if (device.cdp.platform) {
		lines.push(`      platform: "${device.cdp.platform}"`);
	}
};

const appendStpLines = (lines: string[], device: Device) => {
	if (!device.stp?.enabled) {
		return;
	}
	lines.push("    stp:");
	lines.push("      enabled: true");
	if (device.stp.bridgePriority !== undefined) {
		lines.push(`      bridgePriority: ${device.stp.bridgePriority}`);
	}
};

const appendDhcpLines = (lines: string[], device: Device) => {
	if (!device.dhcp) {
		return;
	}
	lines.push("    dhcp:");
	if (device.dhcp.subnetMask) {
		lines.push(`      subnetMask: "${device.dhcp.subnetMask}"`);
	}
	if (device.dhcp.router) {
		lines.push(`      router: "${device.dhcp.router}"`);
	}
	if (device.dhcp.domainNameServer) {
		lines.push(`      domainNameServer: "${device.dhcp.domainNameServer}"`);
	}
};

const appendDnsLines = (lines: string[], device: Device) => {
	if (!device.dns) {
		return;
	}
	lines.push("    dns:");
	if (device.dns.forwardRecords && device.dns.forwardRecords.length > 0) {
		lines.push("      forwardRecords:");
		for (const record of device.dns.forwardRecords) {
			lines.push(`        - name: "${record.name}"`);
			lines.push(`          ip: "${record.ip}"`);
		}
	}
};

const appendHttpLines = (lines: string[], device: Device) => {
	if (!device.http?.enabled) {
		return;
	}
	lines.push("    http:");
	lines.push("      enabled: true");
	if (device.http.serverName) {
		lines.push(`      serverName: "${device.http.serverName}"`);
	}
};

const appendFtpLines = (lines: string[], device: Device) => {
	if (!device.ftp?.enabled) {
		return;
	}
	lines.push("    ftp:");
	lines.push("      enabled: true");
	if (device.ftp.welcomeBanner) {
		lines.push(`      welcomeBanner: "${device.ftp.welcomeBanner}"`);
	}
};

const appendNetbiosLines = (lines: string[], device: Device) => {
	if (!device.netbios?.enabled) {
		return;
	}
	lines.push("    netbios:");
	lines.push("      enabled: true");
	if (device.netbios.name) {
		lines.push(`      name: "${device.netbios.name}"`);
	}
	if (device.netbios.workgroup) {
		lines.push(`      workgroup: "${device.netbios.workgroup}"`);
	}
};

const appendTrafficLines = (lines: string[], device: Device) => {
	if (!device.traffic?.enabled) {
		return;
	}
	lines.push("    traffic:");
	lines.push("      enabled: true");
	if (device.traffic.arpAnnouncements?.enabled) {
		lines.push("      arpAnnouncements:");
		lines.push("        enabled: true");
	}
};

const buildYamlPreview = (device: Device): string => {
	try {
		const lines: string[] = ["devices:"];
		appendBaseDeviceLines(lines, device);
		appendSnmpLines(lines, device);
		appendLldpLines(lines, device);
		appendCdpLines(lines, device);
		appendStpLines(lines, device);
		appendDhcpLines(lines, device);
		appendDnsLines(lines, device);
		appendHttpLines(lines, device);
		appendFtpLines(lines, device);
		appendNetbiosLines(lines, device);
		appendTrafficLines(lines, device);
		return lines.join("\n");
	} catch {
		return "# Error generating YAML preview";
	}
};

const getStatusView = ({
	isNewDevice,
	loading,
	error,
	refetch,
	navigate,
}: {
	isNewDevice: boolean;
	loading: boolean;
	error: Error | null;
	refetch: () => void;
	navigate: (path: string) => void;
}): React.ReactNode | null => {
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

	if (!isNewDevice && error) {
		return (
			<div className="space-y-6">
				<Card className="border-red-500/30 bg-red-900/20">
					<CardContent className="space-y-3">
						<div className="flex items-start gap-3">
							<AlertCircle className="mt-1 h-5 w-5 text-red-400" />
							<div>
								<p className="font-semibold text-red-200">
									Failed to Load Device
								</p>
								<SmallText className="text-red-300/90">
									{error.message}
								</SmallText>
								<div className="flex gap-2 mt-3">
									<Button variant="outline" size="sm" onClick={() => refetch()}>
										Retry
									</Button>
									<Button
										variant="outline"
										size="sm"
										onClick={() => navigate("/device-config")}
									>
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

	return null;
};

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
	const [message, setMessage] = useState<{
		type: "success" | "error";
		text: string;
	} | null>(null);
	const [expandedSections, setExpandedSections] = useState<Set<string>>(
		new Set(["basic"]),
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
			return device.hostname.trim() !== "";
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
			setDevice((prev) => ({ ...prev, [field]: value }));
			setMessage(null);
		},
		[],
	);

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
				if (!hostname) {
					setMessage({ type: "error", text: "Missing hostname for update" });
					setSaving(false);
					return;
				}
				await updateDevice(hostname, device);
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
		if (!hostname || isNewDevice) {
			return;
		}

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
	const yamlPreview = useMemo(() => buildYamlPreview(device), [device]);

	const statusView = getStatusView({
		isNewDevice,
		loading,
		error,
		refetch,
		navigate,
	});
	if (statusView) {
		return statusView;
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
								type="button"
								onClick={() => navigate("/device-config")}
								className="p-2 text-gray-400 hover:text-white hover:bg-white/10 rounded-lg transition-colors"
								title="Back to device list"
							>
								<ArrowLeft className="h-5 w-5" />
							</button>
							<DeviceIcon className="h-6 w-6 text-violet-300" />
							<div>
								<H2 className="mb-0">
									{isNewDevice
										? "New Device"
										: device.hostname || "Edit Device"}
								</H2>
								<SmallText className="text-gray-400">
									{isNewDevice
										? "Create a new network device"
										: "Edit device configuration"}
								</SmallText>
							</div>
							{isDirty && (
								<Tag colorScheme="yellow" className="ml-2">
									Unsaved Changes
								</Tag>
							)}
						</div>
						<div className="flex gap-2">
							<Button
								variant="outline"
								onClick={() => setShowYamlPreview(!showYamlPreview)}
							>
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
							<Button
								variant="outline"
								onClick={handleDiscard}
								disabled={!isDirty || saving}
							>
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
							readOnly={true}
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
				required={true}
			>
				<div className="grid gap-4 md:grid-cols-2">
					<FormField
						label="Hostname"
						required={true}
						helpText="Unique identifier for the device"
					>
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
						required={true}
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
							onChange={(e) =>
								updateField("type", e.target.value as DeviceType)
							}
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
							value={device.ip || ""}
							onChange={(e) => updateField("ip", e.target.value)}
							placeholder="e.g., 192.168.1.1"
							className="w-full rounded-lg border border-white/10 bg-gray-950/60 p-3 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none font-mono"
						/>
					</FormField>
				</div>
			</CollapsibleSection>

			<SnmpSection
				device={device}
				isExpanded={expandedSections.has("snmp")}
				onToggle={() => toggleSection("snmp")}
				onUpdate={updateField}
				walkFiles={walkFiles}
			/>

			<LldpSection
				device={device}
				isExpanded={expandedSections.has("lldp")}
				onToggle={() => toggleSection("lldp")}
				onUpdate={updateField}
			/>

			<CdpSection
				device={device}
				isExpanded={expandedSections.has("cdp")}
				onToggle={() => toggleSection("cdp")}
				onUpdate={updateField}
			/>

			<StpSection
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
						<div key={ip} className="flex gap-2">
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
									const newIps = (device.ips || []).filter(
										(_, i) => i !== index,
									);
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

			<DhcpSection
				device={device}
				isExpanded={expandedSections.has("dhcp")}
				onToggle={() => toggleSection("dhcp")}
				onUpdate={updateField}
			/>

			<DnsSection
				device={device}
				isExpanded={expandedSections.has("dns")}
				onToggle={() => toggleSection("dns")}
				onUpdate={updateField}
			/>

			<HttpSection
				device={device}
				isExpanded={expandedSections.has("http")}
				onToggle={() => toggleSection("http")}
				onUpdate={updateField}
			/>

			<FtpSection
				device={device}
				isExpanded={expandedSections.has("ftp")}
				onToggle={() => toggleSection("ftp")}
				onUpdate={updateField}
			/>

			<NetBiosSection
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
				<div className="fixed inset-0 z-50 flex items-center justify-center">
					<button
						type="button"
						className="absolute inset-0 bg-black/70 backdrop-blur-sm"
						onClick={() => setShowDeleteConfirm(false)}
						aria-label="Close delete confirmation"
					/>
					<div
						className="mx-4 w-full max-w-md rounded-2xl border border-white/10 bg-gray-900/95 shadow-2xl"
						role="dialog"
						aria-modal="true"
					>
						<div className="p-6 space-y-4">
							<div className="flex items-center gap-3 text-red-400">
								<Trash2 className="h-6 w-6" />
								<h2 className="text-lg font-semibold">Delete Device</h2>
							</div>
							<p className="text-gray-300">
								Are you sure you want to delete{" "}
								<strong>{device.hostname}</strong>? This action cannot be
								undone.
							</p>
							<div className="flex justify-end gap-3 pt-2">
								<Button
									variant="outline"
									onClick={() => setShowDeleteConfirm(false)}
								>
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
