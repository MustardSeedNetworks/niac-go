export interface StackStatsResponse {
	timestamp: string;
	interface: string;
	version: string;
	deviceCount: number;
	stack: {
		packetsSent: number;
		packetsReceived: number;
		arpRequests: number;
		arpReplies: number;
		icmpRequests: number;
		icmpReplies: number;
		dnsQueries: number;
		dhcpRequests: number;
		snmpQueries: number;
		errors: number;
	};
}

export interface DeviceSummary {
	name: string;
	type: string;
	ips: string[];
	protocols: string[];
}

export interface HistoryRecord {
	id: number;
	startedAt: string;
	duration: string;
	interface: string;
	configName: string;
	deviceCount: number;
	packetsSent: number;
	packetsReceived: number;
	errors: number;
}

export interface NeighborRecord {
	protocol: string;
	localDevice: string;
	remoteDevice: string;
	remotePort: string;
	remoteChassisId: string;
	description: string;
	capabilities: string[];
	managementAddress: string;
	lastSeen: string;
	ttl: number;
}

export interface ConfigDocument {
	path: string;
	filename: string;
	modifiedAt: string;
	sizeBytes: number;
	deviceCount: number;
	content: string;
}

export interface ConfigUpdateRequest {
	content: string;
}

export interface ReplayState {
	running: boolean;
	file: string;
	loopMs: number;
	scale: number;
	startedAt?: string;
}

export interface ReplayRequest {
	file: string;
	loopMs?: number;
	scale?: number;
	data?: string;
}

export interface FileEntry {
	path: string;
	name: string;
	sizeBytes: number;
	modifiedAt: string;
}

export interface AlertConfig {
	packetsThreshold: number;
	webhookUrl: string;
}

export interface VersionInfo {
	version: string;
}

export interface TopologyGraph {
	nodes: TopologyNode[];
	links: TopologyLink[];
}

export interface TopologyNode {
	name: string;
	type: string;
}

export interface TopologyLink {
	source: string;
	target: string;
	label: string;
}

export interface ErrorType {
	type: string;
	description: string;
}

export interface ErrorInjectionInfo {
	availableTypes: ErrorType[];
	info: string;
	activeErrors?: {
		[deviceIp: string]: {
			[interfaceName: string]: {
				[errorType: string]: number;
			};
		};
	};
}

export interface NetworkInterface {
	name: string;
	description: string;
	addresses: string[];
	current: boolean;
}

export interface InterfacesResponse {
	interfaces: NetworkInterface[];
	currentInterface: string;
}

export interface RuntimeStatus {
	running: boolean;
	interface: string;
	configPath: string;
	configName?: string;
	version: string;
	deviceCount: number;
	packetsSent: number;
	packetsReceived: number;
	uptimeSeconds: number;
}

export interface SimulationStatus {
	running: boolean;
	interface?: string;
	configPath?: string;
	configName?: string;
	deviceCount: number;
	startedAt?: string;
	uptimeSeconds: number;
}

export interface SimulationRequest {
	interface: string;
	configPath?: string;
	configData?: string;
}

// Debug Console Types
export type LogLevel = "ERROR" | "WARN" | "INFO" | "DEBUG";

export type Protocol =
	| "ARP"
	| "ICMP"
	| "DNS"
	| "TCP"
	| "UDP"
	| "SNMP"
	| "DHCP"
	| "LLDP"
	| "CDP"
	| "HTTP"
	| "HTTPS"
	| "SSH"
	| "TELNET";

export interface LogEntry {
	id: string;
	timestamp: string;
	level: LogLevel;
	protocol: Protocol | string;
	message: string;
	source?: string;
	details?: Record<string, unknown>;
}

// Template Types
export interface Template {
	name: string;
	description: string;
	deviceCount: number;
	type:
		| "basic"
		| "router"
		| "switch"
		| "access-point"
		| "server"
		| "complete"
		| "custom";
	tags?: string[];
	createdAt?: string;
	modifiedAt?: string;
}

export interface TemplateContent {
	name: string;
	content: string;
	format: "yaml" | "json";
}

export interface UseTemplateRequest {
	templateName: string;
	newConfigName?: string;
}

export interface UseTemplateResponse {
	success: boolean;
	configPath: string;
	message: string;
}

// Protocol Debug Level Types
export type DebugLevel = "OFF" | "ERROR" | "WARN" | "INFO" | "DEBUG" | "TRACE";

export type DebugProtocol =
	| "SNMP"
	| "LLDP"
	| "CDP"
	| "STP"
	| "LACP"
	| "OSPF"
	| "BGP"
	| "EIGRP"
	| "RIP"
	| "ISIS"
	| "VRRP"
	| "HSRP"
	| "GLBP"
	| "BFD"
	| "MPLS"
	| "PIM"
	| "IGMP"
	| "MSDP"
	| "NetFlow";

export type ProtocolCategory =
	| "discovery"
	| "switching"
	| "routing"
	| "redundancy"
	| "multicast"
	| "monitoring";

export interface ProtocolDebugConfig {
	protocol: DebugProtocol;
	level: DebugLevel;
	category: ProtocolCategory;
}

export interface ProtocolDebugLevelsResponse {
	protocols: ProtocolDebugConfig[];
	defaultLevel: DebugLevel;
}

export interface UpdateProtocolDebugLevelRequest {
	protocol: DebugProtocol;
	level: DebugLevel;
}

export interface UpdateProtocolDebugLevelsRequest {
	protocols: UpdateProtocolDebugLevelRequest[];
}

export interface ResetProtocolDebugLevelsResponse {
	success: boolean;
	message: string;
	protocols: ProtocolDebugConfig[];
}

export interface UploadTemplateRequest {
	name: string;
	description: string;
	content: string;
	type?: Template["type"];
}

export interface UploadTemplateResponse {
	success: boolean;
	template: Template;
	message: string;
}

// PCAP Analyzer Types
export interface PcapPacket {
	id: string;
	number: number;
	timestamp: string;
	sourceIp: string;
	destIp: string;
	sourcePort?: number;
	destPort?: number;
	protocol: string;
	length: number;
	info: string;
	rawData?: string;
	headers?: Record<string, unknown>;
}

export interface PcapAnalysisResult {
	filename: string;
	fileSize: number;
	packets: PcapPacket[];
	stats: PcapStats;
}

export interface PcapStats {
	totalPackets: number;
	totalBytes: number;
	timeRange: {
		start: string;
		end: string;
		durationMs: number;
	};
	protocols: Record<string, number>;
	topSources: Array<{ ip: string; count: number }>;
	topDestinations: Array<{ ip: string; count: number }>;
}

export interface PcapUploadRequest {
	filename: string;
	data: string; // Base64 encoded PCAP data
}

export interface PcapUploadResponse {
	success: boolean;
	analysisId: string;
	message: string;
}

// ============================================================================
// Device Configuration Types (matches Go config.go)
// ============================================================================

/**
 * Device represents a complete network device configuration
 */
export interface Device {
	hostname: string;
	mac: string;
	ip?: string;
	ips?: string[];
	type?: DeviceType;
	vlan?: number;
	babble?: boolean;
	mapToIp?: string;
	snmpAgent?: SNMPAgent;
	lldp?: LLDPConfig;
	cdp?: CDPConfig;
	edp?: EDPConfig;
	fdp?: FDPConfig;
	stp?: STPConfig;
	dhcp?: DHCPConfig;
	dns?: DNSConfig;
	http?: HTTPConfig;
	ftp?: FTPConfig;
	netbios?: NetBIOSConfig;
	icmp?: ICMPConfig;
	icmpv6?: ICMPv6Config;
	dhcpv6?: DHCPv6Config;
	traffic?: TrafficConfig;
	ttl?: TTLConfig;
	osFingerprint?: OSFingerprintConfig;
	iperf3?: IPerf3Config;
}

export type DeviceType =
	| "router"
	| "switch"
	| "access_point"
	| "firewall"
	| "server"
	| "workstation"
	| "iot"
	| "unknown";

/**
 * SNMP Agent configuration
 */
export interface SNMPAgent {
	community?: string;
	sysName?: string;
	sysDescr?: string;
	sysContact?: string;
	sysLocation?: string;
	walkFile?: string;
	walkFiles?: string[];
	addMibs?: AddMib[];
	accessList?: string[];
	snmpAddr?: string;
	dot1dFdbTable?: FdbTableConfig;
	dot1qFdbTable?: FdbTableConfig;
	traps?: TrapConfig;
}

export interface AddMib {
	oid: string;
	type: MibType;
	value: string;
}

export type MibType =
	| "STRING"
	| "INTEGER"
	| "Counter32"
	| "Counter64"
	| "Gauge32"
	| "TimeTicks"
	| "OID"
	| "IpAddress"
	| "Hex-STRING";

export interface FdbTableConfig {
	port: number;
	vlan: number;
}

export interface TrapConfig {
	enabled: boolean;
	receivers?: string[];
	community?: string;
	coldStart?: TrapTriggerConfig;
	linkState?: LinkStateTrapConfig;
	authenticationFailure?: TrapTriggerConfig;
	highCpu?: ThresholdTrapConfig;
	highMemory?: ThresholdTrapConfig;
	interfaceErrors?: ThresholdTrapConfig;
}

export interface TrapTriggerConfig {
	enabled: boolean;
	onStartup?: boolean;
}

export interface LinkStateTrapConfig {
	enabled: boolean;
	linkDown?: boolean;
	linkUp?: boolean;
}

export interface ThresholdTrapConfig {
	enabled: boolean;
	threshold?: number;
	interval?: number;
}

/**
 * LLDP (Link Layer Discovery Protocol) configuration
 */
export interface LLDPConfig {
	enabled: boolean;
	advertiseInterval?: number;
	ttl?: number;
	chassisIdType?: ChassisIDType;
	systemDescription?: string;
	portDescription?: string;
}

export type ChassisIDType = "mac" | "local" | "network_address";

/**
 * CDP (Cisco Discovery Protocol) configuration
 */
export interface CDPConfig {
	enabled: boolean;
	advertiseInterval?: number;
	holdtime?: number;
	version?: 1 | 2;
	platform?: string;
	softwareVersion?: string;
	portId?: string;
}

/**
 * EDP (Extreme Discovery Protocol) configuration
 */
export interface EDPConfig {
	enabled: boolean;
	advertiseInterval?: number;
	versionString?: string;
	displayString?: string;
}

/**
 * FDP (Foundry Discovery Protocol) configuration
 */
export interface FDPConfig {
	enabled: boolean;
	advertiseInterval?: number;
	holdtime?: number;
	softwareVersion?: string;
	platform?: string;
	portId?: string;
}

/**
 * STP (Spanning Tree Protocol) configuration
 */
export interface STPConfig {
	enabled: boolean;
	version?: STPVersion;
	bridgePriority?: number;
	helloTime?: number;
	maxAge?: number;
	forwardDelay?: number;
}

export type STPVersion = "stp" | "rstp" | "mstp";

/**
 * DHCP Server configuration
 */
export interface DHCPConfig {
	subnetMask?: string;
	router?: string;
	domainNameServer?: string;
	serverIdentifier?: string;
	nextServerIp?: string;
	poolStart?: string;
	poolEnd?: string;
	domainName?: string;
	ntpServers?: string[];
	domainSearch?: string[];
	tftpServerName?: string;
	bootfileName?: string;
	vendorSpecific?: string;
	clientLeases?: DHCPLease[];
}

export interface DHCPLease {
	clientIp: string;
	macAddress: string;
	macMask?: string;
}

/**
 * DNS Server configuration
 */
export interface DNSConfig {
	forwardRecords?: DNSRecord[];
	reverseRecords?: DNSRecord[];
}

export interface DNSRecord {
	name: string;
	ip: string;
	ttl?: number;
	rcode?: number;
}

/**
 * HTTP Server configuration
 */
export interface HTTPConfig {
	enabled: boolean;
	serverName?: string;
	endpoints?: HTTPEndpoint[];
}

export interface HTTPEndpoint {
	path: string;
	method?: "GET" | "POST" | "PUT" | "DELETE";
	statusCode?: number;
	contentType?: string;
	body?: string;
}

/**
 * FTP Server configuration
 */
export interface FTPConfig {
	enabled: boolean;
	welcomeBanner?: string;
	systemType?: string;
	allowAnonymous?: boolean;
	users?: FTPUser[];
}

export interface FTPUser {
	username: string;
	password: string;
	homeDir?: string;
}

/**
 * NetBIOS Service configuration
 */
export interface NetBIOSConfig {
	enabled: boolean;
	name?: string;
	workgroup?: string;
	nodeType?: NetBIOSNodeType;
	services?: NetBIOSService[];
	ttl?: number;
	msbrowse?: boolean;
	names?: NetBIOSName[];
}

export type NetBIOSNodeType = "B" | "P" | "M" | "H";
export type NetBIOSService = "workstation" | "fileserver" | "messenger";

export interface NetBIOSName {
	name: string;
	suffix: string;
	group?: boolean;
}

/**
 * ICMP (v4) configuration
 */
export interface ICMPConfig {
	enabled: boolean;
	ttl?: number;
	rateLimit?: number;
	addressMaskReply?: string;
	routerAdvertisement?: ICMPRouterAdvertisement;
}

export interface ICMPRouterAdvertisement {
	period?: number;
	lifetime?: number;
	routers?: ICMPRouter[];
}

export interface ICMPRouter {
	address: string;
	preference?: number;
}

/**
 * ICMPv6 configuration
 */
export interface ICMPv6Config {
	enabled: boolean;
	hopLimit?: number;
	rateLimit?: number;
	routerAdvertisement?: ICMPv6RouterAdvertisement;
}

export interface ICMPv6RouterAdvertisement {
	period?: number;
	curHopLimit?: number;
	managed?: number;
	other?: number;
	lifetime?: number;
	reachableTime?: number;
	retransTimer?: number;
	mtu?: number;
	prefixInfo?: ICMPv6PrefixInfo[];
}

export interface ICMPv6PrefixInfo {
	prefixLength?: number;
	onlink?: number;
	auto?: number;
	validLifetime?: number;
	preferredLifetime?: number;
	prefix?: string;
}

/**
 * DHCPv6 Server configuration
 */
export interface DHCPv6Config {
	enabled: boolean;
	pools?: DHCPv6Pool[];
	preferredLifetime?: number;
	validLifetime?: number;
	preference?: number;
	dnsServers?: string[];
	domainList?: string[];
	sntpServers?: string[];
	ntpServers?: string[];
	sipServers?: string[];
	sipDomains?: string[];
}

export interface DHCPv6Pool {
	network: string;
	rangeStart?: string;
	rangeEnd?: string;
}

/**
 * Traffic Pattern configuration
 */
export interface TrafficConfig {
	enabled: boolean;
	arpAnnouncements?: ARPAnnouncementConfig;
	periodicPings?: PeriodicPingConfig;
	randomTraffic?: RandomTrafficConfig;
}

export interface ARPAnnouncementConfig {
	enabled: boolean;
	interval?: number;
}

export interface PeriodicPingConfig {
	enabled: boolean;
	interval?: number;
	payloadSize?: number;
}

export interface RandomTrafficConfig {
	enabled: boolean;
	interval?: number;
	packetCount?: number;
	patterns?: RandomTrafficPattern[];
}

export type RandomTrafficPattern = "broadcast_arp" | "multicast" | "udp";

/**
 * TTL Configuration for traceroute simulation
 */
export interface TTLConfig {
	ttl: number;
	ip?: string;
	mask?: string;
}

/**
 * OS Fingerprint configuration
 */
export interface OSFingerprintConfig {
	osType?: OSType;
	ttl?: number;
	windowSize?: number;
	windowScale?: number;
	mss?: number;
	sshBanner?: string;
	httpServer?: string;
	ftpBanner?: string;
	smtpBanner?: string;
	telnetBanner?: string;
	dontFragment?: boolean;
}

export type OSType =
	| "linux"
	| "windows"
	| "macos"
	| "freebsd"
	| "cisco-ios"
	| "cisco-nxos"
	| "juniper-junos"
	| "arista-eos";

/**
 * iPerf3 Server configuration
 */
export interface IPerf3Config {
	enabled: boolean;
	port?: number;
	maxBandwidthMbps?: number;
	typicalLatencyMs?: number;
	jitterMs?: number;
	packetLossPercent?: number;
	uploadMbps?: number;
	downloadMbps?: number;
}

// ============================================================================
// Device API Response Types
// ============================================================================

/**
 * Response from GET /api/v1/config/devices
 */
export interface DeviceListResponse {
	devices: Device[];
	total: number;
}

/**
 * Response from GET /api/v1/config/devices/:id
 */
export interface DeviceDetailResponse {
	device: Device;
}

/**
 * Request body for POST /api/v1/config/devices
 */
export interface CreateDeviceRequest {
	device: Device;
}

/**
 * Request body for PUT /api/v1/config/devices/:id
 */
export interface UpdateDeviceRequest {
	device: Partial<Device>;
}

/**
 * Response from device mutations
 */
export interface DeviceMutationResponse {
	success: boolean;
	device?: Device;
	message?: string;
}

/**
 * Request body for POST /api/v1/config/devices/:id/clone
 */
export interface CloneDeviceRequest {
	newHostname: string;
	newMac?: string;
	newIp?: string;
}

/**
 * JSON Schema types for dynamic form generation
 */
export interface JSONSchemaProperty {
	type: string;
	title?: string;
	description?: string;
	default?: unknown;
	enum?: unknown[];
	enumNames?: string[];
	minimum?: number;
	maximum?: number;
	minLength?: number;
	maxLength?: number;
	pattern?: string;
	format?: string;
	items?: JSONSchemaProperty;
	properties?: Record<string, JSONSchemaProperty>;
	required?: string[];
	"ui:widget"?: string;
	"ui:help"?: string;
	"ui:placeholder"?: string;
}

export interface ConfigSchema {
	$schema: string;
	type: string;
	title: string;
	description?: string;
	properties: Record<string, JSONSchemaProperty>;
	required: string[];
	definitions?: Record<string, JSONSchemaProperty>;
}
