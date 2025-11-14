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
