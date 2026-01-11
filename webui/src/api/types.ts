export interface StackStatsResponse {
  timestamp: string;
  interface: string;
  version: string;
  device_count: number;
  stack: {
    packets_sent: number;
    packets_received: number;
    arp_requests: number;
    arp_replies: number;
    icmp_requests: number;
    icmp_replies: number;
    dns_queries: number;
    dhcp_requests: number;
    snmp_queries: number;
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
  started_at: string;
  duration: string;
  interface: string;
  config_name: string;
  device_count: number;
  packets_sent: number;
  packets_received: number;
  errors: number;
}

export interface NeighborRecord {
  Protocol: string;
  LocalDevice: string;
  RemoteDevice: string;
  RemotePort: string;
  RemoteChassisID: string;
  Description: string;
  Capabilities: string[];
  ManagementAddress: string;
  LastSeen: string;
  TTL: number;
}

export interface ConfigDocument {
  path: string;
  filename: string;
  modified_at: string;
  size_bytes: number;
  device_count: number;
  content: string;
}

export interface ConfigUpdateRequest {
  content: string;
}

export interface ReplayState {
  running: boolean;
  file: string;
  loop_ms: number;
  scale: number;
  started_at?: string;
}

export interface ReplayRequest {
  file: string;
  loop_ms?: number;
  scale?: number;
  data?: string;
}

export interface FileEntry {
  path: string;
  name: string;
  size_bytes: number;
  modified_at: string;
}

export interface AlertConfig {
  packets_threshold: number;
  webhook_url: string;
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
  available_types: ErrorType[];
  info: string;
  active_errors?: {
    [deviceIP: string]: {
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
  current_interface: string;
}

export interface RuntimeStatus {
  running: boolean;
  interface: string;
  config_path: string;
  config_name?: string;
  version: string;
  device_count: number;
  packets_sent: number;
  packets_received: number;
  uptime_seconds: number;
}

export interface SimulationStatus {
  running: boolean;
  interface?: string;
  config_path?: string;
  config_name?: string;
  device_count: number;
  started_at?: string;
  uptime_seconds: number;
}

export interface SimulationRequest {
  interface: string;
  config_path?: string;
  config_data?: string;
}

// Debug Console Types
export type LogLevel = 'ERROR' | 'WARN' | 'INFO' | 'DEBUG';

export type Protocol = 'ARP' | 'ICMP' | 'DNS' | 'TCP' | 'UDP' | 'SNMP' | 'DHCP' | 'LLDP' | 'CDP' | 'HTTP' | 'HTTPS' | 'SSH' | 'TELNET';

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
  device_count: number;
  type: 'basic' | 'router' | 'switch' | 'access-point' | 'server' | 'complete' | 'custom';
  tags?: string[];
  created_at?: string;
  modified_at?: string;
}

export interface TemplateContent {
  name: string;
  content: string;
  format: 'yaml' | 'json';
}

export interface UseTemplateRequest {
  template_name: string;
  new_config_name?: string;
}

export interface UseTemplateResponse {
  success: boolean;
  config_path: string;
  message: string;
}

// Protocol Debug Level Types
export type DebugLevel = 'OFF' | 'ERROR' | 'WARN' | 'INFO' | 'DEBUG' | 'TRACE';

export type DebugProtocol =
  | 'SNMP'
  | 'LLDP'
  | 'CDP'
  | 'STP'
  | 'LACP'
  | 'OSPF'
  | 'BGP'
  | 'EIGRP'
  | 'RIP'
  | 'ISIS'
  | 'VRRP'
  | 'HSRP'
  | 'GLBP'
  | 'BFD'
  | 'MPLS'
  | 'PIM'
  | 'IGMP'
  | 'MSDP'
  | 'NetFlow';

export type ProtocolCategory = 'discovery' | 'switching' | 'routing' | 'redundancy' | 'multicast' | 'monitoring';

export interface ProtocolDebugConfig {
  protocol: DebugProtocol;
  level: DebugLevel;
  category: ProtocolCategory;
}

export interface ProtocolDebugLevelsResponse {
  protocols: ProtocolDebugConfig[];
  default_level: DebugLevel;
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
  type?: Template['type'];
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
  sourceIP: string;
  destIP: string;
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
