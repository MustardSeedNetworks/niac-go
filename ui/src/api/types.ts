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
  map_to_ip?: string;
  snmp_agent?: SNMPAgent;
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
  os_fingerprint?: OSFingerprintConfig;
  iperf3?: IPerf3Config;
}

export type DeviceType =
  | 'router'
  | 'switch'
  | 'access_point'
  | 'firewall'
  | 'server'
  | 'workstation'
  | 'iot'
  | 'unknown';

/**
 * SNMP Agent configuration
 */
export interface SNMPAgent {
  community?: string;
  sys_name?: string;
  sys_descr?: string;
  sys_contact?: string;
  sys_location?: string;
  walk_file?: string;
  walk_files?: string[];
  add_mibs?: AddMib[];
  access_list?: string[];
  snmp_addr?: string;
  dot1d_fdb_table?: FdbTableConfig;
  dot1q_fdb_table?: FdbTableConfig;
  traps?: TrapConfig;
}

export interface AddMib {
  oid: string;
  type: MibType;
  value: string;
}

export type MibType =
  | 'STRING'
  | 'INTEGER'
  | 'Counter32'
  | 'Counter64'
  | 'Gauge32'
  | 'TimeTicks'
  | 'OID'
  | 'IpAddress'
  | 'Hex-STRING';

export interface FdbTableConfig {
  port: number;
  vlan: number;
}

export interface TrapConfig {
  enabled: boolean;
  receivers?: string[];
  community?: string;
  cold_start?: TrapTriggerConfig;
  link_state?: LinkStateTrapConfig;
  authentication_failure?: TrapTriggerConfig;
  high_cpu?: ThresholdTrapConfig;
  high_memory?: ThresholdTrapConfig;
  interface_errors?: ThresholdTrapConfig;
}

export interface TrapTriggerConfig {
  enabled: boolean;
  on_startup?: boolean;
}

export interface LinkStateTrapConfig {
  enabled: boolean;
  link_down?: boolean;
  link_up?: boolean;
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
  advertise_interval?: number;
  ttl?: number;
  chassis_id_type?: ChassisIDType;
  system_description?: string;
  port_description?: string;
}

export type ChassisIDType = 'mac' | 'local' | 'network_address';

/**
 * CDP (Cisco Discovery Protocol) configuration
 */
export interface CDPConfig {
  enabled: boolean;
  advertise_interval?: number;
  holdtime?: number;
  version?: 1 | 2;
  platform?: string;
  software_version?: string;
  port_id?: string;
}

/**
 * EDP (Extreme Discovery Protocol) configuration
 */
export interface EDPConfig {
  enabled: boolean;
  advertise_interval?: number;
  version_string?: string;
  display_string?: string;
}

/**
 * FDP (Foundry Discovery Protocol) configuration
 */
export interface FDPConfig {
  enabled: boolean;
  advertise_interval?: number;
  holdtime?: number;
  software_version?: string;
  platform?: string;
  port_id?: string;
}

/**
 * STP (Spanning Tree Protocol) configuration
 */
export interface STPConfig {
  enabled: boolean;
  version?: STPVersion;
  bridge_priority?: number;
  hello_time?: number;
  max_age?: number;
  forward_delay?: number;
}

export type STPVersion = 'stp' | 'rstp' | 'mstp';

/**
 * DHCP Server configuration
 */
export interface DHCPConfig {
  subnet_mask?: string;
  router?: string;
  domain_name_server?: string;
  server_identifier?: string;
  next_server_ip?: string;
  pool_start?: string;
  pool_end?: string;
  domain_name?: string;
  ntp_servers?: string[];
  domain_search?: string[];
  tftp_server_name?: string;
  bootfile_name?: string;
  vendor_specific?: string;
  client_leases?: DHCPLease[];
}

export interface DHCPLease {
  client_ip: string;
  mac_address: string;
  mac_mask?: string;
}

/**
 * DNS Server configuration
 */
export interface DNSConfig {
  forward_records?: DNSRecord[];
  reverse_records?: DNSRecord[];
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
  server_name?: string;
  endpoints?: HTTPEndpoint[];
}

export interface HTTPEndpoint {
  path: string;
  method?: 'GET' | 'POST' | 'PUT' | 'DELETE';
  status_code?: number;
  content_type?: string;
  body?: string;
}

/**
 * FTP Server configuration
 */
export interface FTPConfig {
  enabled: boolean;
  welcome_banner?: string;
  system_type?: string;
  allow_anonymous?: boolean;
  users?: FTPUser[];
}

export interface FTPUser {
  username: string;
  password: string;
  home_dir?: string;
}

/**
 * NetBIOS Service configuration
 */
export interface NetBIOSConfig {
  enabled: boolean;
  name?: string;
  workgroup?: string;
  node_type?: NetBIOSNodeType;
  services?: NetBIOSService[];
  ttl?: number;
  msbrowse?: boolean;
  names?: NetBIOSName[];
}

export type NetBIOSNodeType = 'B' | 'P' | 'M' | 'H';
export type NetBIOSService = 'workstation' | 'fileserver' | 'messenger';

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
  rate_limit?: number;
  address_mask_reply?: string;
  router_advertisement?: ICMPRouterAdvertisement;
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
  hop_limit?: number;
  rate_limit?: number;
  router_advertisement?: ICMPv6RouterAdvertisement;
}

export interface ICMPv6RouterAdvertisement {
  period?: number;
  cur_hop_limit?: number;
  managed?: number;
  other?: number;
  lifetime?: number;
  reachable_time?: number;
  retrans_timer?: number;
  mtu?: number;
  prefix_info?: ICMPv6PrefixInfo[];
}

export interface ICMPv6PrefixInfo {
  prefix_length?: number;
  onlink?: number;
  auto?: number;
  valid_lifetime?: number;
  preferred_lifetime?: number;
  prefix?: string;
}

/**
 * DHCPv6 Server configuration
 */
export interface DHCPv6Config {
  enabled: boolean;
  pools?: DHCPv6Pool[];
  preferred_lifetime?: number;
  valid_lifetime?: number;
  preference?: number;
  dns_servers?: string[];
  domain_list?: string[];
  sntp_servers?: string[];
  ntp_servers?: string[];
  sip_servers?: string[];
  sip_domains?: string[];
}

export interface DHCPv6Pool {
  network: string;
  range_start?: string;
  range_end?: string;
}

/**
 * Traffic Pattern configuration
 */
export interface TrafficConfig {
  enabled: boolean;
  arp_announcements?: ARPAnnouncementConfig;
  periodic_pings?: PeriodicPingConfig;
  random_traffic?: RandomTrafficConfig;
}

export interface ARPAnnouncementConfig {
  enabled: boolean;
  interval?: number;
}

export interface PeriodicPingConfig {
  enabled: boolean;
  interval?: number;
  payload_size?: number;
}

export interface RandomTrafficConfig {
  enabled: boolean;
  interval?: number;
  packet_count?: number;
  patterns?: RandomTrafficPattern[];
}

export type RandomTrafficPattern = 'broadcast_arp' | 'multicast' | 'udp';

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
  os_type?: OSType;
  ttl?: number;
  window_size?: number;
  window_scale?: number;
  mss?: number;
  ssh_banner?: string;
  http_server?: string;
  ftp_banner?: string;
  smtp_banner?: string;
  telnet_banner?: string;
  dont_fragment?: boolean;
}

export type OSType =
  | 'linux'
  | 'windows'
  | 'macos'
  | 'freebsd'
  | 'cisco-ios'
  | 'cisco-nxos'
  | 'juniper-junos'
  | 'arista-eos';

/**
 * iPerf3 Server configuration
 */
export interface IPerf3Config {
  enabled: boolean;
  port?: number;
  max_bandwidth_mbps?: number;
  typical_latency_ms?: number;
  jitter_ms?: number;
  packet_loss_percent?: number;
  upload_mbps?: number;
  download_mbps?: number;
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
  new_hostname: string;
  new_mac?: string;
  new_ip?: string;
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
  'ui:widget'?: string;
  'ui:help'?: string;
  'ui:placeholder'?: string;
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
