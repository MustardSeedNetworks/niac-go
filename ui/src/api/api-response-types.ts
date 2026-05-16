/**
 * API Response and Request Types
 * General API types for simulation control, history, topology, etc.
 */

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

/**
 * One issue found by the SNMP walk-file validator. Severity is one of
 * "error" | "warning" | "info"; "autoFix" means the validator can rewrite
 * the line via POST /api/v1/walk/fix.
 */
export interface WalkValidationIssue {
  line: number;
  severity: string;
  message: string;
  original: string;
  suggestion?: string;
  autoFix: boolean;
}

/**
 * Result of validating (or auto-fixing) a walk file via
 * POST /api/v1/walk/{validate,fix}.
 */
export interface WalkValidationResult {
  filename: string;
  valid: boolean;
  totalLines: number;
  validLines: number;
  issues: WalkValidationIssue[];
  fixedCount?: number;
  fixedLines?: number[];
}

export interface WalkValidationResponse {
  success: boolean;
  message?: string;
  result?: WalkValidationResult;
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
  /** True when the link came from runtime LLDP/CDP/EDP/FDP discovery
   *  rather than a config-declared trunk_port. Drives the
   *  declared-vs-discovered visual distinction (solid vs dashed). */
  discovered?: boolean;
  /** "trunk" | "lag" | "" — set by BuildTopology from the trunk_ports
   *  declaration. Drives the edge colour: trunks get the cyan link-type
   *  hue, LAGs the same with thicker stroke. Empty falls through to
   *  speed-based colouring. */
  linkType?: string;
  sourceInterface?: string;
  targetInterface?: string;
  vlans?: number[];
  nativeVlan?: number;
  /** Speed in Mbps as a string ("100", "1000", "10000", "25000", ...).
   *  When the device has an Interface declaration with a Speed field,
   *  the daemon includes it; otherwise omitted and the edge falls
   *  through to the default colour. */
  speed?: string;
  duplex?: string;
  status?: string;
  utilization?: number;
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

/**
 * StandaloneCaptureStatus mirrors the daemon's api.CaptureStatus. It
 * powers the "sniff this interface" mode of the PCAP Inspector — i.e.
 * packet capture without a simulation.
 */
export interface StandaloneCaptureStatus {
  running: boolean;
  interface?: string;
  filter?: string;
  startedAt?: string;
  packets: number;
}

export interface StandaloneCaptureRequest {
  interface: string;
  /** Optional libpcap BPF expression. Empty captures everything. */
  filter?: string;
}

export interface SimulationRequest {
  interface: string;
  configPath?: string;
  configData?: string;
  /**
   * Built-in template to load directly from disk. Mutually exclusive
   * with configPath / configData. Lets the daemon resolve relative
   * include_path / walk_file references against the template's own
   * source directory rather than the inline-config cache.
   */
  templateName?: string;
}
