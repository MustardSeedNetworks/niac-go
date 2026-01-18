import type { FC } from "react";
import type { LLDPConfig } from "../../api/types";
import { CollapsibleSection, FormField } from "../form";
import type { ProtocolSectionProps } from "./types";
import { inputClassName } from "./types";

export const LldpSection: FC<ProtocolSectionProps> = ({
	device,
	isExpanded,
	onToggle,
	onUpdate,
}) => {
	const getLldpConfig = (): LLDPConfig => ({
		enabled: true,
		...(device.lldp ?? {}),
	});

	const updateLldp = (config: LLDPConfig | undefined) => {
		onUpdate("lldp", config);
	};

	return (
		<CollapsibleSection
			title="LLDP (Link Layer Discovery Protocol)"
			isExpanded={isExpanded}
			onToggle={onToggle}
			enabled={device.lldp?.enabled ?? false}
			onEnableChange={(enabled) => {
				updateLldp(enabled ? getLldpConfig() : undefined);
			}}
		>
			{device.lldp?.enabled && (
				<div className="grid gap-4 md:grid-cols-2">
					<FormField
						label="System Description"
						helpText="LLDP system description"
					>
						<input
							type="text"
							value={device.lldp.systemDescription || ""}
							onChange={(e) =>
								updateLldp({
									...getLldpConfig(),
									systemDescription: e.target.value,
								})
							}
							placeholder="Cisco IOS Software..."
							className={inputClassName}
						/>
					</FormField>

					<FormField label="Port Description" helpText="LLDP port description">
						<input
							type="text"
							value={device.lldp.portDescription || ""}
							onChange={(e) =>
								updateLldp({
									...getLldpConfig(),
									portDescription: e.target.value,
								})
							}
							placeholder="GigabitEthernet0/1"
							className={inputClassName}
						/>
					</FormField>

					<FormField
						label="Advertise Interval (seconds)"
						helpText="How often to send LLDP frames"
					>
						<input
							type="number"
							value={device.lldp.advertiseInterval || 30}
							onChange={(e) =>
								updateLldp({
									...getLldpConfig(),
									advertiseInterval: Number.parseInt(e.target.value, 10),
								})
							}
							min={5}
							max={32768}
							className={inputClassName}
						/>
					</FormField>

					<FormField
						label="TTL (seconds)"
						helpText="Time-to-live for LLDP information"
					>
						<input
							type="number"
							value={device.lldp.ttl || 120}
							onChange={(e) =>
								updateLldp({
									...getLldpConfig(),
									ttl: Number.parseInt(e.target.value, 10),
								})
							}
							min={1}
							max={65535}
							className={inputClassName}
						/>
					</FormField>
				</div>
			)}
		</CollapsibleSection>
	);
};
