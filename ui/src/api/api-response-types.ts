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
