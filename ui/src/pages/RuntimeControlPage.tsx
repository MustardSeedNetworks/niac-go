import { Activity, BellRing, FileCog, Network, PlugZap } from "lucide-react";
import { type FC, memo, useCallback, useState } from "react";
import {
	fetchInterfaces,
	fetchSimulationStatus,
	startSimulation,
	stopSimulation,
} from "../api/client";
import type { NetworkInterface } from "../api/types";
import { StatBlock } from "../components/StatBlock";
import { POLL_INTERVALS } from "../constants/polling";
import { useApiResource } from "../hooks/useApiResource";
import { Button } from "../ui/Button";
import { Card, CardContent } from "../ui/Card";
import { Tag } from "../ui/Tag";
import { H2, P, SmallText } from "../ui/Typography";
import { fileToText } from "../utils/file";
import {
	formatBytes,
	formatTime,
	formatUptime,
	getErrorMessage,
} from "../utils/format";

/**
 * Runtime Control Page
 *
 * Allows starting/stopping simulations and viewing network interfaces.
 * Requires NIAC to be running in daemon mode.
 */
export const RuntimeControlPage: FC = () => {
	const [refetchTrigger, setRefetchTrigger] = useState(0);
	const { data: simStatus } = useApiResource(
		fetchSimulationStatus,
		[refetchTrigger],
		{
			intervalMs: POLL_INTERVALS.FAST,
		},
	);
	const { data: interfaces } = useApiResource(fetchInterfaces, []);

	const [selectedInterface, setSelectedInterface] = useState("");
	const [configPath, setConfigPath] = useState("");
	const [configFile, setConfigFile] = useState<File | null>(null);
	const [starting, setStarting] = useState(false);
	const [stopping, setStopping] = useState(false);
	const [message, setMessage] = useState<{
		tone: "success" | "error";
		text: string;
	} | null>(null);

	const isDaemonMode = simStatus !== null;

	const handleStart = useCallback(async () => {
		if (!(selectedInterface || configPath || configFile)) {
			setMessage({
				tone: "error",
				text: "Please select an interface and provide a config file",
			});
			return;
		}

		setStarting(true);
		setMessage(null);

		try {
			let configData: string | undefined;

			if (configFile) {
				configData = await fileToText(configFile);
			}

			await startSimulation({
				interface: selectedInterface,
				configPath: configPath || undefined,
				configData: configData,
			});

			setMessage({ tone: "success", text: "Simulation started successfully!" });
			setRefetchTrigger((t) => t + 1);
			setConfigFile(null);
		} catch (err) {
			setMessage({ tone: "error", text: getErrorMessage(err) });
		} finally {
			setStarting(false);
		}
	}, [selectedInterface, configPath, configFile]);

	const handleStop = useCallback(async () => {
		if (
			!window.confirm(
				"Are you sure you want to stop the simulation? This will interrupt the current run.",
			)
		) {
			return;
		}

		setStopping(true);
		setMessage(null);

		try {
			await stopSimulation();
			setMessage({ tone: "success", text: "Simulation stopped" });
			setRefetchTrigger((t) => t + 1);
		} catch (err) {
			setMessage({ tone: "error", text: getErrorMessage(err) });
		} finally {
			setStopping(false);
		}
	}, []);

	return (
		<div className="space-y-6">
			{!isDaemonMode && (
				<Card className="border-yellow-500/30 bg-yellow-900/20">
					<CardContent className="space-y-3">
						<div className="flex items-start gap-3">
							<BellRing className="mt-1 h-5 w-5 text-yellow-400" />
							<div>
								<p className="font-semibold text-yellow-200">
									Daemon Mode Not Detected
								</p>
								<SmallText className="text-yellow-300/90">
									To use simulation controls, start NIAC in daemon mode:
								</SmallText>
								<code className="mt-2 block rounded bg-black/40 p-3 font-mono text-sm text-yellow-100">
									niac daemon --listen :8080 --token yourtoken
								</code>
								<SmallText className="mt-2 text-yellow-300/80">
									Legacy mode (<code>niac --api-listen</code>) doesn't support
									start/stop controls.
								</SmallText>
							</div>
						</div>
					</CardContent>
				</Card>
			)}

			{isDaemonMode && !simStatus?.running && (
				<Card className="border-white/5 bg-gradient-to-br from-violet-900/30 to-gray-900/70">
					<CardContent className="space-y-5">
						<div className="flex items-center gap-3">
							<PlugZap className="h-6 w-6 text-violet-400" />
							<H2 className="mb-0">Start Simulation</H2>
						</div>

						<div className="space-y-4">
							<div>
								<label
									htmlFor="network-interface"
									className="block text-sm text-gray-400"
								>
									Network Interface *
								</label>
								<select
									id="network-interface"
									className="mt-1 w-full rounded-lg border border-white/10 bg-gray-950/60 p-2 text-sm text-white focus:border-violet-400 focus:outline-none"
									value={selectedInterface}
									onChange={(e) => setSelectedInterface(e.target.value)}
									aria-required="true"
									aria-label="Select network interface for simulation"
								>
									<option value="">Select interface...</option>
									{interfaces?.interfaces.map((iface) => (
										<option key={iface.name} value={iface.name}>
											{iface.name}{" "}
											{iface.description ? `- ${iface.description}` : ""}{" "}
											{iface.addresses.length > 0
												? `(${iface.addresses[0]})`
												: ""}
										</option>
									))}
								</select>
							</div>

							<div>
								<label
									htmlFor="config-path"
									className="block text-sm text-gray-400"
								>
									Config File Path
								</label>
								<input
									id="config-path"
									type="text"
									className="mt-1 w-full rounded-lg border border-white/10 bg-gray-950/60 p-2 font-mono text-sm text-white focus:border-violet-400 focus:outline-none"
									placeholder="/path/to/config.yaml"
									value={configPath}
									onChange={(e) => setConfigPath(e.target.value)}
									aria-describedby="config-path-help"
								/>
								<SmallText id="config-path-help" className="text-gray-500">
									Or upload a config file below
								</SmallText>
							</div>

							<div>
								<label
									htmlFor="config-file-upload"
									className="block text-sm text-gray-400"
								>
									Upload Config File
								</label>
								<input
									id="config-file-upload"
									type="file"
									accept=".yaml,.yml"
									className="mt-1 w-full cursor-pointer rounded-lg border border-dashed border-white/10 bg-gray-950/40 p-2 text-sm text-white file:mr-3 file:rounded-md file:border-0 file:bg-violet-600 file:px-3 file:py-1 file:text-sm file:font-medium"
									onChange={(e) => {
										const file = e.target.files?.[0];
										if (!file) {
											setConfigFile(null);
											return;
										}

										const MaxSize = 10 * 1024 * 1024;
										if (file.size > MaxSize) {
											setMessage({
												tone: "error",
												text: `File too large. Maximum size is ${formatBytes(MaxSize)}`,
											});
											e.target.value = "";
											return;
										}

										if (!file.name.match(/\.(yaml|yml)$/i)) {
											setMessage({
												tone: "error",
												text: "Please select a YAML file (.yaml or .yml)",
											});
											e.target.value = "";
											return;
										}

										setConfigFile(file);
									}}
									aria-describedby="config-file-help"
								/>
								{configFile && (
									<SmallText
										id="config-file-help"
										className="mt-1 text-green-400"
									>
										Selected: {configFile.name}
									</SmallText>
								)}
							</div>

							{message && (
								<SmallText
									className={
										message.tone === "success"
											? "text-emerald-300"
											: "text-red-400"
									}
									role="alert"
									aria-live="polite"
								>
									{message.text}
								</SmallText>
							)}

							<Button
								tone="violet"
								size="lg"
								disabled={
									!(selectedInterface && (configPath || configFile)) || starting
								}
								onClick={handleStart}
								leftIcon={<Activity className="h-5 w-5" />}
							>
								{starting ? "Starting..." : "Start Simulation"}
							</Button>
						</div>
					</CardContent>
				</Card>
			)}

			{isDaemonMode && simStatus?.running && (
				<Card className="border-green-500/30 bg-gradient-to-br from-green-900/30 to-gray-900/70">
					<CardContent className="space-y-5">
						<div className="flex items-center justify-between">
							<div className="flex items-center gap-3">
								<div className="h-3 w-3 animate-pulse rounded-full bg-green-400" />
								<H2 className="mb-0">Simulation Running</H2>
							</div>
							<Tag colorScheme="green">ACTIVE</Tag>
						</div>

						<div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
							<StatBlock
								label="Interface"
								value={simStatus.interface || "—"}
								helper="Network interface"
							/>
							<StatBlock
								label="Config"
								value={simStatus.configName || "—"}
								helper={simStatus.configPath || "Configuration file"}
							/>
							<StatBlock
								label="Devices"
								value={simStatus.deviceCount.toString()}
								helper="Simulated devices"
							/>
							<StatBlock
								label="Uptime"
								value={formatUptime(simStatus.uptimeSeconds)}
								helper="Time running"
							/>
							<StatBlock
								label="Started"
								value={
									simStatus.startedAt ? formatTime(simStatus.startedAt) : "—"
								}
								helper="Start time"
							/>
						</div>

						{message && (
							<SmallText
								className={
									message.tone === "success"
										? "text-emerald-300"
										: "text-red-400"
								}
							>
								{message.text}
							</SmallText>
						)}

						<div className="flex flex-wrap gap-3">
							<Button
								variant="outline"
								disabled={stopping}
								onClick={handleStop}
								leftIcon={<Activity className="h-4 w-4" />}
							>
								{stopping ? "Stopping..." : "Stop Simulation"}
							</Button>
							<Button
								variant="ghost"
								leftIcon={<FileCog className="h-4 w-4" />}
								onClick={() => {
									window.location.href = "/devices";
								}}
							>
								View Devices
							</Button>
						</div>
					</CardContent>
				</Card>
			)}

			{interfaces && (
				<Card className="border-white/5 bg-gray-900/70">
					<CardContent className="space-y-4">
						<H2 className="mb-0 flex items-center gap-2">
							<Network className="h-5 w-5 text-cyan-300" />
							Available Network Interfaces
						</H2>
						<P className="text-gray-300">
							Network interfaces available on this system. Select one to use for
							simulation.
						</P>
						<InterfaceList interfaces={interfaces.interfaces} />
					</CardContent>
				</Card>
			)}
		</div>
	);
};

const InterfaceList = memo(
	({ interfaces }: { interfaces: NetworkInterface[] }) => {
		if (interfaces.length === 0) {
			return (
				<SmallText className="text-gray-400">
					No network interfaces found.
				</SmallText>
			);
		}

		return (
			<div className="space-y-2">
				{interfaces.map((iface) => (
					<div
						key={iface.name}
						className={`rounded-lg border p-4 ${
							iface.current
								? "border-violet-500/50 bg-violet-900/20"
								: "border-white/10 bg-gray-950/50"
						}`}
					>
						<div className="flex items-center justify-between">
							<div>
								<div className="flex items-center gap-2">
									<p className="font-semibold text-white">{iface.name}</p>
									{iface.current && <Tag colorScheme="purple">ACTIVE</Tag>}
								</div>
								{iface.description && (
									<SmallText className="text-gray-400">
										{iface.description}
									</SmallText>
								)}
								{iface.addresses.length > 0 && (
									<div className="mt-1 flex flex-wrap gap-2">
										{iface.addresses.map((addr) => (
											<code
												key={addr}
												className="rounded bg-black/30 px-2 py-0.5 font-mono text-xs text-blue-300"
											>
												{addr}
											</code>
										))}
									</div>
								)}
							</div>
						</div>
					</div>
				))}
			</div>
		);
	},
);

InterfaceList.displayName = "InterfaceList";

export default RuntimeControlPage;
