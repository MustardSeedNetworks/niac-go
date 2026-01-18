import type { FC } from "react";
import type { CDPConfig } from "../../api/types";
import { CollapsibleSection, FormField } from "../form";
import type { ProtocolSectionProps } from "./types";
import { inputClassName } from "./types";

export const CdpSection: FC<ProtocolSectionProps> = ({
	device,
	isExpanded,
	onToggle,
	onUpdate,
}) => {
	const getCdpConfig = (): CDPConfig => ({
		enabled: true,
		...(device.cdp ?? {}),
	});

	const updateCdp = (config: CDPConfig | undefined) => {
		onUpdate("cdp", config);
	};

	return (
		<CollapsibleSection
			title="CDP (Cisco Discovery Protocol)"
			isExpanded={isExpanded}
			onToggle={onToggle}
			enabled={device.cdp?.enabled ?? false}
			onEnableChange={(enabled) => {
				updateCdp(enabled ? getCdpConfig() : undefined);
			}}
		>
			{device.cdp?.enabled && (
				<div className="grid gap-4 md:grid-cols-2">
					<FormField label="Platform" helpText="Hardware platform">
						<input
							type="text"
							value={device.cdp.platform || ""}
							onChange={(e) =>
								updateCdp({ ...getCdpConfig(), platform: e.target.value })
							}
							placeholder="cisco WS-C3750-48P"
							className={inputClassName}
						/>
					</FormField>

					<FormField label="Port ID" helpText="Local port identifier">
						<input
							type="text"
							value={device.cdp.portId || ""}
							onChange={(e) =>
								updateCdp({ ...getCdpConfig(), portId: e.target.value })
							}
							placeholder="GigabitEthernet0/1"
							className={inputClassName}
						/>
					</FormField>

					<FormField label="Software Version" helpText="IOS/NX-OS version">
						<input
							type="text"
							value={device.cdp.softwareVersion || ""}
							onChange={(e) =>
								updateCdp({
									...getCdpConfig(),
									softwareVersion: e.target.value,
								})
							}
							placeholder="Cisco IOS Software, Version 15.1(4)M"
							className={inputClassName}
						/>
					</FormField>

					<FormField
						label="Holdtime (seconds)"
						helpText="How long to hold neighbor information"
					>
						<input
							type="number"
							value={device.cdp.holdtime || 180}
							onChange={(e) =>
								updateCdp({
									...getCdpConfig(),
									holdtime: Number.parseInt(e.target.value, 10),
								})
							}
							min={10}
							max={255}
							className={inputClassName}
						/>
					</FormField>
				</div>
			)}
		</CollapsibleSection>
	);
};
