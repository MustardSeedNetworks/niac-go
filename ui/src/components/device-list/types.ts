import type { Device } from "../../api/types";

export interface DeviceViewProps {
	devices: Device[];
	selectedDevices: Set<string>;
	onSelectDevice: (hostname: string) => void;
	onSelectAll: () => void;
	onEdit: (hostname: string) => void;
	onClone: (hostname: string) => void;
	onDelete: (hostname: string) => void;
	getDeviceProtocols: (device: Device) => string[];
}
