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
  /** Hardware MAC, lowercase-colon-separated; omitted when device has none. */
  mac?: string;
  /** Vendor key from the device's `properties.vendor` (e.g. "cisco"). */
  vendor?: string;
  /** Model from the device's `properties.model` (e.g. "C9300-48P"). */
  model?: string;
  /** Arbitrary device properties from the YAML config. */
  properties?: Record<string, string>;
}

/**
 * One multi-VLAN segment (ADR 0008) — a VLAN tag and the device set
 * grouped under it, as returned by GET /api/v1/segments. A flat
 * (non-segmented) config still reports exactly one untagged segment
 * wrapping every device, so the wire shape is uniform whether or not
 * `segments:` appears in the YAML.
 */
export interface SegmentSummary {
  /** 0 = untagged/native VLAN, else 1..4094. */
  vlanTag: number;
  /** True only for the native/untagged segment. */
  untagged?: boolean;
  /** Same shape as GET /api/v1/devices — see DeviceSummary. */
  devices: DeviceSummary[];
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
  oid?: string;
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

/**
 * Response from POST /api/v1/walk/validate-all — validates every walk file
 * referenced by the running config in one call. `success` is true only when
 * every file validated clean (mirrors the backend's invalidFiles==0 check,
 * unlike WalkValidationResponse.success which just means "request succeeded").
 */
export interface WalkBatchValidationResponse {
  success: boolean;
  message: string;
  totalFiles: number;
  invalidFiles: number;
  totalIssues: number;
  results: Record<string, WalkValidationResult>;
}

/**
 * One (vendor, model) baseline-walk profile, as returned by
 * GET /api/v1/synthesize-walk/models. Powers the "Synthesize baseline
 * walk" picker in the device editor's SNMP section. Mirrors
 * synth.ModelDescriptor server-side; vendor/model/type are free-form
 * strings owned by the synth package, not a fixed frontend enum.
 */
export interface ModelDescriptor {
  vendor: string;
  model: string;
  /** Human label, e.g. "Catalyst 9300-48P (48× 1G PoE+)". */
  label: string;
  /** Device type this model implies (switch, router, firewall, ...). */
  type: string;
  /** Default ifTable row count for this model. */
  ifCount: number;
  /** Per-port speed in Mbps. */
  speedMbps: number;
}

/**
 * Request body for POST /api/v1/devices/{hostname}/synthesize-walk.
 * Model is optional — omitting it falls back to the vendor's generic
 * per-type profile server-side rather than a specific platform.
 */
export interface SynthesizeWalkRequest {
  vendor: string;
  model?: string;
  interfaceCount?: number;
  community?: string;
}

/**
 * 201 response from POST /api/v1/devices/{hostname}/synthesize-walk.
 * walkPath is library-relative and already attached to the device's
 * walk_file server-side; originalPreserved is true when a pristine
 * ".orig" now exists so the walk can be reverted later.
 */
export interface SynthesizeWalkResponse {
  walkPath: string;
  oidCount: number;
  sizeBytes: number;
  originalPreserved: boolean;
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

/** Replay pacing mode; '' / 'timing' honors the captured inter-packet timing. */
export type ReplayRateMode = '' | 'timing' | 'topspeed' | 'pps' | 'mbps';

export interface ReplayState {
  running: boolean;
  file: string;
  loopMs: number;
  scale: number;
  rateMode?: ReplayRateMode;
  pps?: number;
  mbpsCap?: number;
  loopCount?: number;
  bpfFilter?: string;
  startedAt?: string;
  packetsSent: number;
  bytesSent: number;
  packetsTotal: number;
  bytesTotal: number;
  /** Omitted by the backend (not sent as 0) whenever packetsTotal is unknown. */
  percentComplete?: number;
  /** Completed replay iterations across the run ("iteration N"). */
  passes: number;
  /** Packets skipped by bpfFilter in the current pass. */
  packetsFiltered: number;
}

export interface ReplayRequest {
  file: string;
  loopMs?: number;
  scale?: number;
  data?: string;
  rateMode?: ReplayRateMode;
  pps?: number;
  mbpsCap?: number;
  loopCount?: number;
  bpfFilter?: string;
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
  /** Link utilization, 0–100. Source = the daemon's json:"utilizationPercent"
   *  on api.TopologyLink; the request layer's snake→camel converter rewrites
   *  the wire field to utilizationPercent. Today the daemon emits 0 for
   *  every link (counter wiring is a separate work item); when live counters
   *  land, the topology page already styles edges based on this value
   *  (thickness + amber/red tint at high utilisation). */
  utilizationPercent?: number;
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
