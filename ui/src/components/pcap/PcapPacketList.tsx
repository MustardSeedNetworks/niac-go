import { ChevronDown, ChevronUp, Filter, Search } from "lucide-react";
import { type FC, memo, useCallback, useMemo, useState } from "react";
import type { PcapPacket } from "../../api/types";
import { Card, CardContent } from "../../ui/Card";
import { Tag } from "../../ui/Tag";
import { SmallText } from "../../ui/Typography";

interface PcapPacketListProps {
	packets: PcapPacket[];
	selectedPacketId: string | null;
	onSelectPacket: (packet: PcapPacket) => void;
	protocolFilter: string;
	sourceFilter: string;
	destFilter: string;
	searchQuery: string;
	onProtocolFilterChange: (protocol: string) => void;
	onSourceFilterChange: (source: string) => void;
	onDestFilterChange: (dest: string) => void;
	onSearchQueryChange: (query: string) => void;
}

/** Available protocol filters */
const PROTOCOL_OPTIONS = [
	"All",
	"TCP",
	"UDP",
	"ICMP",
	"ARP",
	"DNS",
	"HTTP",
	"HTTPS",
	"DHCP",
	"SSH",
	"TLS",
] as const;

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
 * Format timestamp to show time with milliseconds
 */
function formatTimestamp(timestamp: string): string {
	try {
		const date = new Date(timestamp);
		return date.toLocaleTimeString("en-US", {
			hour12: false,
			hour: "2-digit",
			minute: "2-digit",
			second: "2-digit",
			fractionalSecondDigits: 3,
		});
	} catch {
		return timestamp;
	}
}

/**
 * Individual packet row component - memoized for performance
 */
const PacketRow = memo(
	({
		packet,
		isSelected,
		onClick,
	}: {
		packet: PcapPacket;
		isSelected: boolean;
		onClick: () => void;
	}) => (
		<tr
			onClick={onClick}
			className={`cursor-pointer transition-colors ${
				isSelected ? "bg-violet-900/30" : "hover:bg-gray-900/50"
			}`}
		>
			<td className="px-3 py-2 text-gray-400 text-xs font-mono">
				{packet.number}
			</td>
			<td className="px-3 py-2 text-gray-300 text-xs font-mono">
				{formatTimestamp(packet.timestamp)}
			</td>
			<td className="px-3 py-2 text-white text-sm font-mono">
				{packet.sourceIp}
			</td>
			<td className="px-3 py-2 text-white text-sm font-mono">
				{packet.destIp}
			</td>
			<td className="px-3 py-2">
				<Tag
					colorScheme={getProtocolColor(packet.protocol)}
					className="text-xs"
				>
					{packet.protocol}
				</Tag>
			</td>
			<td className="px-3 py-2 text-gray-400 text-xs text-right">
				{packet.length}
			</td>
			<td className="px-3 py-2 text-gray-400 text-xs truncate max-w-xs">
				{packet.info}
			</td>
		</tr>
	),
);

PacketRow.displayName = "PacketRow";

/**
 * Filter Panel Component
 */
const FilterPanel: FC<{
	protocolFilter: string;
	sourceFilter: string;
	destFilter: string;
	searchQuery: string;
	onProtocolFilterChange: (protocol: string) => void;
	onSourceFilterChange: (source: string) => void;
	onDestFilterChange: (dest: string) => void;
	onSearchQueryChange: (query: string) => void;
	uniqueSources: string[];
	uniqueDests: string[];
}> = memo(
	({
		protocolFilter,
		sourceFilter,
		destFilter,
		searchQuery,
		onProtocolFilterChange,
		onSourceFilterChange,
		onDestFilterChange,
		onSearchQueryChange,
		uniqueSources,
		uniqueDests,
	}) => {
		const [isExpanded, setIsExpanded] = useState(true);

		return (
			<div className="space-y-3">
				<button
					type="button"
					onClick={() => setIsExpanded(!isExpanded)}
					className="flex w-full items-center justify-between text-left"
				>
					<div className="flex items-center gap-2">
						<Filter className="h-4 w-4 text-gray-400" />
						<SmallText className="text-gray-400 uppercase tracking-wide font-semibold">
							Filters
						</SmallText>
					</div>
					{isExpanded ? (
						<ChevronUp className="h-4 w-4 text-gray-400" />
					) : (
						<ChevronDown className="h-4 w-4 text-gray-400" />
					)}
				</button>

				{isExpanded && (
					<div className="space-y-3">
						{/* Search */}
						<div className="relative">
							<Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-500" />
							<input
								type="text"
								placeholder="Search packets..."
								value={searchQuery}
								onChange={(e) => onSearchQueryChange(e.target.value)}
								className="w-full rounded-lg border border-white/10 bg-gray-950/60 py-2 pl-10 pr-3 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none"
							/>
						</div>

						{/* Protocol Filter */}
						<div>
							<SmallText className="text-gray-500 mb-1 block">
								Protocol
							</SmallText>
							<select
								value={protocolFilter}
								onChange={(e) => onProtocolFilterChange(e.target.value)}
								className="w-full rounded-lg border border-white/10 bg-gray-950/60 px-3 py-2 text-sm text-white focus:border-violet-400 focus:outline-none"
							>
								{PROTOCOL_OPTIONS.map((protocol) => (
									<option key={protocol} value={protocol}>
										{protocol}
									</option>
								))}
							</select>
						</div>

						{/* Source Filter */}
						<div>
							<SmallText className="text-gray-500 mb-1 block">
								Source IP
							</SmallText>
							<select
								value={sourceFilter}
								onChange={(e) => onSourceFilterChange(e.target.value)}
								className="w-full rounded-lg border border-white/10 bg-gray-950/60 px-3 py-2 text-sm text-white focus:border-violet-400 focus:outline-none"
							>
								<option value="">All Sources</option>
								{uniqueSources.map((source) => (
									<option key={source} value={source}>
										{source}
									</option>
								))}
							</select>
						</div>

						{/* Destination Filter */}
						<div>
							<SmallText className="text-gray-500 mb-1 block">
								Destination IP
							</SmallText>
							<select
								value={destFilter}
								onChange={(e) => onDestFilterChange(e.target.value)}
								className="w-full rounded-lg border border-white/10 bg-gray-950/60 px-3 py-2 text-sm text-white focus:border-violet-400 focus:outline-none"
							>
								<option value="">All Destinations</option>
								{uniqueDests.map((dest) => (
									<option key={dest} value={dest}>
										{dest}
									</option>
								))}
							</select>
						</div>
					</div>
				)}
			</div>
		);
	},
);

FilterPanel.displayName = "FilterPanel";

/**
 * PCAP Packet List Component
 *
 * Displays a table of captured packets with filtering and selection.
 * Supports filtering by protocol, source IP, destination IP, and search query.
 */
export const PcapPacketList: FC<PcapPacketListProps> = memo(
	({
		packets,
		selectedPacketId,
		onSelectPacket,
		protocolFilter,
		sourceFilter,
		destFilter,
		searchQuery,
		onProtocolFilterChange,
		onSourceFilterChange,
		onDestFilterChange,
		onSearchQueryChange,
	}) => {
		// Extract unique sources and destinations for filter dropdowns
		const { uniqueSources, uniqueDests } = useMemo(() => {
			const sources = new Set<string>();
			const dests = new Set<string>();

			for (const packet of packets) {
				if (packet.sourceIp) {
					sources.add(packet.sourceIp);
				}
				if (packet.destIp) {
					dests.add(packet.destIp);
				}
			}

			return {
				uniqueSources: Array.from(sources).sort(),
				uniqueDests: Array.from(dests).sort(),
			};
		}, [packets]);

		// Filter packets based on all criteria
		const filteredPackets = useMemo(() => {
			return packets.filter((packet) => {
				// Protocol filter
				if (
					protocolFilter !== "All" &&
					packet.protocol.toUpperCase() !== protocolFilter.toUpperCase()
				) {
					return false;
				}

				// Source filter
				if (sourceFilter && packet.sourceIp !== sourceFilter) {
					return false;
				}

				// Destination filter
				if (destFilter && packet.destIp !== destFilter) {
					return false;
				}

				// Search filter
				if (searchQuery) {
					const query = searchQuery.toLowerCase();
					const searchableText = [
						packet.protocol,
						packet.sourceIp,
						packet.destIp,
						packet.info,
						packet.sourcePort?.toString(),
						packet.destPort?.toString(),
					]
						.filter(Boolean)
						.join(" ")
						.toLowerCase();

					return searchableText.includes(query);
				}

				return true;
			});
		}, [packets, protocolFilter, sourceFilter, destFilter, searchQuery]);

		// Handle packet selection
		const handleSelectPacket = useCallback(
			(packet: PcapPacket) => {
				onSelectPacket(packet);
			},
			[onSelectPacket],
		);

		if (packets.length === 0) {
			return (
				<Card className="border-white/5 bg-gray-900/70 h-full">
					<CardContent className="h-full flex items-center justify-center text-gray-400">
						<div className="text-center">
							<p className="text-sm">No packets to display</p>
							<SmallText>Upload a PCAP file to analyze</SmallText>
						</div>
					</CardContent>
				</Card>
			);
		}

		return (
			<Card className="border-white/5 bg-gray-900/70 h-full flex flex-col">
				<CardContent className="h-full flex flex-col">
					{/* Filter Panel */}
					<div className="mb-4">
						<FilterPanel
							protocolFilter={protocolFilter}
							sourceFilter={sourceFilter}
							destFilter={destFilter}
							searchQuery={searchQuery}
							onProtocolFilterChange={onProtocolFilterChange}
							onSourceFilterChange={onSourceFilterChange}
							onDestFilterChange={onDestFilterChange}
							onSearchQueryChange={onSearchQueryChange}
							uniqueSources={uniqueSources}
							uniqueDests={uniqueDests}
						/>
					</div>

					{/* Packet count */}
					<div className="mb-3 flex items-center justify-between">
						<SmallText className="text-gray-400">
							Showing {filteredPackets.length} of {packets.length} packets
						</SmallText>
					</div>

					{/* Packet Table */}
					<div className="flex-1 min-h-0 overflow-auto rounded-lg border border-white/5">
						{filteredPackets.length === 0 ? (
							<div className="h-full flex items-center justify-center text-gray-400">
								<div className="text-center">
									<p className="text-sm">No packets match filter</p>
									<SmallText>Try adjusting the filter criteria</SmallText>
								</div>
							</div>
						) : (
							<table className="min-w-full divide-y divide-white/5">
								<thead className="bg-gray-900/80 sticky top-0 z-10">
									<tr>
										<th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-gray-400">
											#
										</th>
										<th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-gray-400">
											Time
										</th>
										<th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-gray-400">
											Source
										</th>
										<th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-gray-400">
											Destination
										</th>
										<th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-gray-400">
											Protocol
										</th>
										<th className="px-3 py-2 text-right text-xs font-semibold uppercase tracking-wide text-gray-400">
											Length
										</th>
										<th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-gray-400">
											Info
										</th>
									</tr>
								</thead>
								<tbody className="divide-y divide-white/5">
									{filteredPackets.map((packet) => (
										<PacketRow
											key={packet.id}
											packet={packet}
											isSelected={selectedPacketId === packet.id}
											onClick={() => handleSelectPacket(packet)}
										/>
									))}
								</tbody>
							</table>
						)}
					</div>
				</CardContent>
			</Card>
		);
	},
);

PcapPacketList.displayName = "PcapPacketList";

export default PcapPacketList;
