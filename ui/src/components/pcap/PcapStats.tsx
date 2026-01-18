import {
	ArrowRightLeft,
	BarChart3,
	Clock,
	FileText,
	Network,
	Server,
} from "lucide-react";
import { type FC, memo } from "react";
import type { PcapStats as PcapStatsType } from "../../api/types";
import { Card, CardContent } from "../../ui/Card";
import { Tag } from "../../ui/Tag";
import { H2, SmallText } from "../../ui/Typography";

interface PcapStatsProps {
	stats: PcapStatsType | null;
	filename: string | null;
	fileSize: number | null;
}

/**
 * Format bytes to human-readable string
 */
function formatBytes(bytes: number): string {
	if (bytes < 1024) {
		return `${bytes} B`;
	}
	if (bytes < 1024 * 1024) {
		return `${(bytes / 1024).toFixed(1)} KB`;
	}
	if (bytes < 1024 * 1024 * 1024) {
		return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
	}
	return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`;
}

/**
 * Format duration in milliseconds to human-readable string
 */
function formatDuration(ms: number): string {
	if (ms < 1000) {
		return `${ms.toFixed(0)} ms`;
	}
	if (ms < 60000) {
		return `${(ms / 1000).toFixed(2)} s`;
	}
	if (ms < 3600000) {
		return `${(ms / 60000).toFixed(1)} min`;
	}
	return `${(ms / 3600000).toFixed(1)} hr`;
}

/**
 * Format timestamp to readable string
 */
function formatTimestamp(timestamp: string): string {
	try {
		const date = new Date(timestamp);
		return date.toLocaleString("en-US", {
			month: "short",
			day: "numeric",
			hour: "2-digit",
			minute: "2-digit",
			second: "2-digit",
			hour12: false,
		});
	} catch {
		return timestamp;
	}
}

/**
 * Get color scheme for protocol tag
 */
function getProtocolColor(
	protocol: string,
): "blue" | "green" | "yellow" | "purple" | "gray" | "red" {
	const colors: Record<
		string,
		"blue" | "green" | "yellow" | "purple" | "gray" | "red"
	> = {
		arp: "yellow",
		icmp: "blue",
		dns: "green",
		tcp: "purple",
		udp: "gray",
		http: "blue",
		https: "green",
		dhcp: "yellow",
		ssh: "red",
		tls: "green",
	};
	return colors[protocol.toLowerCase()] || "gray";
}

/**
 * Stat Block Component - displays a single statistic
 */
const StatBlock = memo(
	({
		icon,
		label,
		value,
		helper,
	}: {
		icon: React.ReactNode;
		label: string;
		value: string | number;
		helper?: string;
	}) => (
		<div className="rounded-lg border border-white/5 bg-gray-950/50 p-4">
			<div className="flex items-center gap-2 mb-2">
				{icon}
				<SmallText className="text-gray-400 uppercase tracking-wide">
					{label}
				</SmallText>
			</div>
			<p className="text-2xl font-bold text-white">{value}</p>
			{helper && <SmallText className="text-gray-500">{helper}</SmallText>}
		</div>
	),
);

StatBlock.displayName = "StatBlock";

/**
 * Protocol Breakdown Component
 */
const ProtocolBreakdown: FC<{
	protocols: Record<string, number>;
	total: number;
}> = memo(({ protocols, total }) => {
	// Sort protocols by count (descending)
	const sortedProtocols = Object.entries(protocols)
		.sort(([, a], [, b]) => b - a)
		.slice(0, 10);

	if (sortedProtocols.length === 0) {
		return (
			<SmallText className="text-gray-400">
				No protocol data available
			</SmallText>
		);
	}

	return (
		<div className="space-y-2">
			{sortedProtocols.map(([protocol, count]) => {
				const percentage = total > 0 ? (count / total) * 100 : 0;

				return (
					<div key={protocol} className="space-y-1">
						<div className="flex items-center justify-between">
							<div className="flex items-center gap-2">
								<Tag
									colorScheme={getProtocolColor(protocol)}
									className="text-xs"
								>
									{protocol}
								</Tag>
								<SmallText className="text-gray-400">
									{count.toLocaleString()}
								</SmallText>
							</div>
							<SmallText className="text-gray-500">
								{percentage.toFixed(1)}%
							</SmallText>
						</div>
						<div className="h-1.5 w-full rounded-full bg-gray-800 overflow-hidden">
							<div
								className="h-full rounded-full bg-violet-500 transition-all duration-500"
								style={{ width: `${percentage}%` }}
							/>
						</div>
					</div>
				);
			})}
		</div>
	);
});

ProtocolBreakdown.displayName = "ProtocolBreakdown";

/**
 * Top Endpoints Component
 */
const TopEndpoints: FC<{
	sources: Array<{ ip: string; count: number }>;
	destinations: Array<{ ip: string; count: number }>;
}> = memo(({ sources, destinations }) => {
	return (
		<div className="grid gap-4 md:grid-cols-2">
			{/* Top Sources */}
			<div className="space-y-2">
				<div className="flex items-center gap-2 mb-3">
					<Server className="h-4 w-4 text-blue-400" />
					<SmallText className="text-gray-400 uppercase tracking-wide font-semibold">
						Top Sources
					</SmallText>
				</div>
				{sources.length === 0 ? (
					<SmallText className="text-gray-500">No source data</SmallText>
				) : (
					<div className="space-y-1">
						{sources.slice(0, 5).map((item) => (
							<div
								key={item.ip}
								className="flex items-center justify-between rounded-lg border border-white/5 bg-gray-950/50 px-3 py-2"
							>
								<span className="font-mono text-sm text-white">{item.ip}</span>
								<Tag colorScheme="blue" className="text-xs">
									{item.count.toLocaleString()}
								</Tag>
							</div>
						))}
					</div>
				)}
			</div>

			{/* Top Destinations */}
			<div className="space-y-2">
				<div className="flex items-center gap-2 mb-3">
					<Network className="h-4 w-4 text-green-400" />
					<SmallText className="text-gray-400 uppercase tracking-wide font-semibold">
						Top Destinations
					</SmallText>
				</div>
				{destinations.length === 0 ? (
					<SmallText className="text-gray-500">No destination data</SmallText>
				) : (
					<div className="space-y-1">
						{destinations.slice(0, 5).map((item) => (
							<div
								key={item.ip}
								className="flex items-center justify-between rounded-lg border border-white/5 bg-gray-950/50 px-3 py-2"
							>
								<span className="font-mono text-sm text-white">{item.ip}</span>
								<Tag colorScheme="green" className="text-xs">
									{item.count.toLocaleString()}
								</Tag>
							</div>
						))}
					</div>
				)}
			</div>
		</div>
	);
});

TopEndpoints.displayName = "TopEndpoints";

/**
 * PCAP Statistics Component
 *
 * Displays summary statistics from PCAP analysis including:
 * - Total packets and bytes
 * - Time range and duration
 * - Protocol breakdown with percentages
 * - Top source and destination endpoints
 */
export const PcapStats: FC<PcapStatsProps> = memo(
	({ stats, filename, fileSize }) => {
		if (!stats) {
			return (
				<Card className="border-white/5 bg-gray-900/70">
					<CardContent className="py-12 text-center">
						<BarChart3 className="h-12 w-12 text-gray-600 mx-auto mb-4" />
						<p className="text-gray-400">No statistics available</p>
						<SmallText className="text-gray-500">
							Upload and analyze a PCAP file to see statistics
						</SmallText>
					</CardContent>
				</Card>
			);
		}

		return (
			<div className="space-y-6">
				{/* File Info */}
				{filename && (
					<Card className="border-white/5 bg-gray-900/70">
						<CardContent>
							<div className="flex items-center gap-3">
								<FileText className="h-5 w-5 text-violet-400" />
								<div>
									<p className="font-medium text-white">{filename}</p>
									{fileSize && (
										<SmallText className="text-gray-400">
											{formatBytes(fileSize)}
										</SmallText>
									)}
								</div>
							</div>
						</CardContent>
					</Card>
				)}

				{/* Summary Statistics */}
				<Card className="border-white/5 bg-gray-900/70">
					<CardContent className="space-y-4">
						<H2 className="flex items-center gap-2 mb-0">
							<BarChart3 className="h-5 w-5 text-cyan-300" />
							Summary Statistics
						</H2>

						<div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
							<StatBlock
								icon={<ArrowRightLeft className="h-4 w-4 text-violet-400" />}
								label="Total Packets"
								value={stats.totalPackets.toLocaleString()}
							/>
							<StatBlock
								icon={<FileText className="h-4 w-4 text-blue-400" />}
								label="Total Bytes"
								value={formatBytes(stats.totalBytes)}
								helper={`${stats.totalBytes.toLocaleString()} bytes`}
							/>
							<StatBlock
								icon={<Clock className="h-4 w-4 text-green-400" />}
								label="Duration"
								value={formatDuration(stats.timeRange.durationMs)}
							/>
							<StatBlock
								icon={<Network className="h-4 w-4 text-yellow-400" />}
								label="Protocols"
								value={Object.keys(stats.protocols).length}
								helper="Unique protocols"
							/>
						</div>
					</CardContent>
				</Card>

				{/* Time Range */}
				<Card className="border-white/5 bg-gray-900/70">
					<CardContent className="space-y-4">
						<div className="flex items-center gap-2">
							<Clock className="h-5 w-5 text-emerald-300" />
							<H2 className="mb-0">Time Range</H2>
						</div>

						<div className="grid gap-4 sm:grid-cols-2">
							<div className="rounded-lg border border-white/5 bg-gray-950/50 p-4">
								<SmallText className="text-gray-400 uppercase tracking-wide">
									Start Time
								</SmallText>
								<p className="text-lg font-mono text-white mt-1">
									{formatTimestamp(stats.timeRange.start)}
								</p>
							</div>
							<div className="rounded-lg border border-white/5 bg-gray-950/50 p-4">
								<SmallText className="text-gray-400 uppercase tracking-wide">
									End Time
								</SmallText>
								<p className="text-lg font-mono text-white mt-1">
									{formatTimestamp(stats.timeRange.end)}
								</p>
							</div>
						</div>
					</CardContent>
				</Card>

				{/* Protocol Breakdown */}
				<Card className="border-white/5 bg-gray-900/70">
					<CardContent className="space-y-4">
						<div className="flex items-center gap-2">
							<BarChart3 className="h-5 w-5 text-violet-300" />
							<H2 className="mb-0">Protocol Breakdown</H2>
						</div>

						<ProtocolBreakdown
							protocols={stats.protocols}
							total={stats.totalPackets}
						/>
					</CardContent>
				</Card>

				{/* Top Endpoints */}
				<Card className="border-white/5 bg-gray-900/70">
					<CardContent className="space-y-4">
						<div className="flex items-center gap-2">
							<Network className="h-5 w-5 text-blue-300" />
							<H2 className="mb-0">Top Endpoints</H2>
						</div>

						<TopEndpoints
							sources={stats.topSources}
							destinations={stats.topDestinations}
						/>
					</CardContent>
				</Card>
			</div>
		);
	},
);

PcapStats.displayName = "PcapStats";

export default PcapStats;
